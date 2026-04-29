package actor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/hybridgroup/yzma/pkg/download"
	"github.com/hybridgroup/yzma/pkg/llama"
	"github.com/hybridgroup/yzma/pkg/message"
	"github.com/hybridgroup/yzma/pkg/template"
)

const (
	DefaultTemperature = float32(0.6)
	DefaultTopP        = float32(0.95)
	DefaultTopK        = int32(20)
	DefaultMaxTokens   = 2048
	DefaultContextSize = 4096
)

// Config holds the tunable parameters for an Actor.
type Config struct {
	Temperature float32
	TopP        float32
	TopK        int32
	MaxTokens   int
	ContextSize uint32
	// ModelFormat controls which tool-call grammar instructions are injected
	// into the system prompt. Leave as message.FormatAuto (zero value) to
	// auto-detect from the model path.
	ModelFormat message.Format
	// InjectTools controls whether tool definitions and usage instructions are
	// appended to the system prompt. Defaults to true. Set to false for models
	// that have native tool-call support baked into their chat template.
	InjectTools bool
	// EnableThinking controls whether thinking/reasoning mode is enabled for
	// models that support it (e.g. Qwen3). When false the model is instructed
	// to skip chain-of-thought reasoning and respond directly. Defaults to
	// false to avoid spoken thinking content being published via MQTT.
	EnableThinking bool
}

// DefaultConfig returns a Config populated with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Temperature:    DefaultTemperature,
		TopP:           DefaultTopP,
		TopK:           DefaultTopK,
		MaxTokens:      DefaultMaxTokens,
		ContextSize:    DefaultContextSize,
		InjectTools:    true,
		EnableThinking: false,
	}
}

// Actor drives a conversation using a local llama.cpp model loaded via yzma.
type Actor struct {
	cfg          Config
	llamaModel   llama.Model
	llamaCtx     llama.Context
	vocab        llama.Vocab
	sampler      llama.Sampler
	chatTemplate string

	moreConversationFunc func(conversation *[]message.Message)
	outputFunc           func(content string)
	tools                map[string]Tool
	toolsJSON            string
}

// NewActor creates a new instance of Actor.
// modelPath is the local path to a GGUF model file.
// The llama.cpp shared libraries are loaded from the YZMA_LIB environment variable.
func NewActor(modelPath string, cfg Config, commander Commander, moreFunc func(conversation *[]message.Message), outputFunc func(content string)) (*Actor, error) {
	libPath := os.Getenv("YZMA_LIB")
	if libPath == "" {
		return nil, fmt.Errorf("YZMA_LIB environment variable not set")
	}

	if err := llama.Load(libPath); err != nil {
		return nil, fmt.Errorf("unable to load llama.cpp: %w", err)
	}

	llama.LogSet(llama.LogSilent())
	llama.Init()

	log.Println("loading model...")

	mdl, err := llama.ModelLoadFromFile(modelPath, llama.ModelDefaultParams())
	if err != nil {
		return nil, fmt.Errorf("unable to load model: %w", err)
	}

	ctxParams := llama.ContextDefaultParams()
	ctxParams.NCtx = cfg.ContextSize
	lctx, err := llama.InitFromModel(mdl, ctxParams)
	if err != nil {
		llama.ModelFree(mdl)
		return nil, fmt.Errorf("unable to create context: %w", err)
	}

	vocab := llama.ModelGetVocab(mdl)

	chatTmpl := llama.ModelChatTemplate(mdl, "")
	if chatTmpl == "" {
		chatTmpl = "chatml"
	}

	sp := llama.DefaultSamplerParams()
	sp.Temp = cfg.Temperature
	sp.TopP = cfg.TopP
	sp.TopK = cfg.TopK
	smpl := llama.NewSampler(mdl, llama.DefaultSamplers, sp)

	// Auto-detect model format from the model path when not explicitly set.
	if cfg.ModelFormat == message.FormatAuto {
		cfg.ModelFormat = message.DetectFormat(modelPath)
	}

	toolsMap := make(map[string]Tool)
	toolDocs := []message.ToolDefinition{
		RegisterMovement(toolsMap, commander),
	}

	return &Actor{
		cfg:                  cfg,
		llamaModel:           mdl,
		llamaCtx:             lctx,
		vocab:                vocab,
		sampler:              smpl,
		chatTemplate:         chatTmpl,
		moreConversationFunc: moreFunc,
		outputFunc:           outputFunc,
		tools:                toolsMap,
		toolsJSON:            marshalToolDocs(toolDocs),
	}, nil
}

// Close releases model and context resources.
func (a *Actor) Close() {
	llama.SamplerFree(a.sampler)
	llama.Free(a.llamaCtx)
	llama.ModelFree(a.llamaModel)
	llama.Close()
}

// Run starts the actor and runs the chat loop.
func (a *Actor) Run(ctx context.Context, systemPrompt string) error {
	sysContent := systemPrompt
	if a.cfg.InjectTools {
		sysContent = injectToolsIntoSystemPrompt(systemPrompt, a.toolsJSON, a.cfg.ModelFormat)
	}
	conversation := []message.Message{
		message.Chat{Role: "system", Content: sysContent},
	}

	needMoreInput := true
	consecutiveToolOnlyTurns := 0
	const maxConsecutiveToolOnlyTurns = 2

	for {
		if needMoreInput {
			consecutiveToolOnlyTurns = 0
			if ok := a.GetMore(&conversation); !ok {
				return nil
			}
		}

		content, hadText, toolCalls, err := a.generateTurn(ctx, &conversation)
		if err != nil {
			return err
		}

		if len(toolCalls) > 0 {
			toolResults := a.callTools(ctx, toolCalls)

			if hadText {
				// Text was spoken alongside the tool calls — record the full
				// exchange (including spoken text) so subsequent turns have correct context.
				a.appendToolCalls(&conversation, toolCalls, content)
				conversation = append(conversation, toolResults...)
				consecutiveToolOnlyTurns = 0
				needMoreInput = true
			} else {
				consecutiveToolOnlyTurns++
				a.appendToolCalls(&conversation, toolCalls, "")
				conversation = append(conversation, toolResults...)
				if consecutiveToolOnlyTurns >= maxConsecutiveToolOnlyTurns {
					// Still no text after nudge — give up and wait for new input.
					log.Printf("breaking tool-call loop after %d consecutive tool-only turns", consecutiveToolOnlyTurns)
					consecutiveToolOnlyTurns = 0
					needMoreInput = true
				} else {
					// Inject a user nudge so the model understands it must also
					// give a verbal response — tool calls alone are not enough.
					conversation = append(conversation, message.Chat{
						Role:    "user",
						Content: "You used tools but said nothing. You MUST include spoken words in your response — answer the question with actual sentences, not just tool calls.",
					})
					log.Printf("tool-only turn %d/%d, nudging for verbal response", consecutiveToolOnlyTurns, maxConsecutiveToolOnlyTurns)
					needMoreInput = false
				}
			}
			continue
		}

		a.appendAssistant(&conversation, content)
		needMoreInput = true
	}
}

// GetMore gets more input and appends it to the conversation.
func (a *Actor) GetMore(conversation *[]message.Message) bool {
	if a.moreConversationFunc == nil {
		return false
	}

	a.moreConversationFunc(conversation)
	return true
}

// generateTurn applies the chat template, runs inference, and returns the text
// content and any tool calls parsed from the response. hadText is true when
// spoken text was flushed to outputFunc alongside tool calls in the same turn.
// The conversation pointer may be updated to trim old messages if the prompt
// exceeds the model's context window.
func (a *Actor) generateTurn(ctx context.Context, conversation *[]message.Message) (string, bool, []message.ToolCall, error) {
	tmplOpts := template.Options{EnableThinking: a.cfg.EnableThinking}
	prompt, err := template.ApplyWithOptions(a.chatTemplate, *conversation, true, tmplOpts)
	if err != nil {
		return "", false, nil, fmt.Errorf("error applying chat template: %w", err)
	}

	mem, err := llama.GetMemory(a.llamaCtx)
	if err != nil {
		return "", false, nil, fmt.Errorf("error getting memory: %w", err)
	}
	if err := llama.MemoryClear(mem, true); err != nil {
		return "", false, nil, fmt.Errorf("error clearing memory: %w", err)
	}

	llama.SamplerReset(a.sampler)

	tokens := llama.Tokenize(a.vocab, prompt, true, false)

	// Trim oldest non-system messages if the prompt exceeds the context window.
	if nCtx := int(llama.NCtx(a.llamaCtx)); nCtx > 0 {
		maxPromptTokens := nCtx - a.cfg.MaxTokens
		for len(tokens) > maxPromptTokens && len(*conversation) > 2 {
			// Drop the second message (oldest non-system entry) and re-tokenize.
			log.Println("trimming oldest message from conversation to fit context window")
			*conversation = append((*conversation)[:1], (*conversation)[2:]...)
			prompt, err = template.ApplyWithOptions(a.chatTemplate, *conversation, true, tmplOpts)
			if err != nil {
				return "", false, nil, fmt.Errorf("error applying chat template after trim: %w", err)
			}
			tokens = llama.Tokenize(a.vocab, prompt, true, false)
		}
	}

	nBatch := int(llama.NBatch(a.llamaCtx))
	if nBatch <= 0 {
		nBatch = 512
	}
	for i := 0; i < len(tokens); i += nBatch {
		end := i + nBatch
		if end > len(tokens) {
			end = len(tokens)
		}
		if _, err := llama.Decode(a.llamaCtx, llama.BatchGetOne(tokens[i:end])); err != nil {
			return "", false, nil, fmt.Errorf("error decoding prompt: %w", err)
		}
	}

	var chunks []string
	pieceBuf := make([]byte, 128)

	// Gemma 4 turn markers use <|turn>role format; also include the
	// decoded form <turn>role (TokenToPiece strips | on some builds) and
	// the closing <turn|> token so any variant halts generation.
	// Also stop at <|channel>thought (and decoded form) since everything
	// inside a thought channel is internal reasoning, not spoken text.
	generationStopMarkers := []string{
		"<|turn>user", "<|turn>model",
		"<turn>user", "<turn>model",
		"<turn|>",
		// Bare turn token without role suffix — emitted by Gemma 4 fine-tunes
		// as a turn-end marker after the last tool call.
		"<|turn|>", "<|turn>",
		"<|channel>thought", "<channel>thought",
		// ChatML new-message token — stop if Qwen/ChatML model starts
		// simulating the next conversation turn.
		"<|im_start|>", "<|im_end|>",
		// Stop if the model starts simulating tool results — the inline JSON
		// tool call has already been captured and faking results would just
		// pollute the generation with garbage tokens.
		// Both bare forms and underscore forms are listed because the Gemma
		// template may render tool results as <tool_result>…</tool_result> and
		// the model echoes that exact form in subsequent generations.
		"<toolresult", "<|toolresult", "<tool_result",
		"<toolresponse", "<|toolresponse", "<tool_response",
		// Stop if the model echoes back a tool result in bare word{...} form.
		`tool{"status"`,
		// Stop at Gemma 4 turn-end marker.
		"<turnend>", "<|turnend>",
	}

generateLoop:
	for i := 0; i < a.cfg.MaxTokens; i++ {
		select {
		case <-ctx.Done():
			return "", false, nil, ctx.Err()
		default:
		}

		token := llama.SamplerSample(a.sampler, a.llamaCtx, -1)
		if llama.VocabIsEOG(a.vocab, token) {
			break
		}

		n := llama.TokenToPiece(a.vocab, token, pieceBuf, 0, true)
		if n > 0 {
			piece := string(pieceBuf[:n])
			chunks = append(chunks, piece)

			full := strings.Join(chunks, "")
			for _, marker := range generationStopMarkers {
				if idx := strings.Index(full, marker); idx >= 0 {
					chunks = []string{full[:idx]}
					break generateLoop
				}
			}
			// Stop generation if too many tool call blocks have accumulated.
			// Models that flood the output with identical tool calls (e.g.,
			// dozens of consecutive "wait" commands) would otherwise consume
			// the entire MaxTokens budget without producing any spoken text.
			const maxToolCallBlocksPerGeneration = 8
			if strings.Count(full, "</tool_call>") >= maxToolCallBlocksPerGeneration {
				log.Printf("stopping generation: accumulated %d tool call blocks", maxToolCallBlocksPerGeneration)
				break generateLoop
			}
		}

		if _, err := llama.Decode(a.llamaCtx, llama.BatchGetOne([]llama.Token{token})); err != nil {
			break
		}
	}

	text := strings.TrimSpace(strings.TrimLeft(strings.Join(chunks, ""), "\n"))
	log.Printf("raw generation: %q", text)

	toolCalls := message.ParseToolCalls(text)
	if len(toolCalls) > 0 {
		var hadText bool
		var spokenText string
		if spokenText = message.StripMarkup(text); spokenText != "" && a.outputFunc != nil {
			remaining := flushSentences(spokenText, a.outputFunc)
			if remaining != "" {
				a.outputFunc(remaining)
			}
			hadText = true
		}
		return spokenText, hadText, toolCalls, nil
	}

	content := message.StripMarkup(text)
	if content != "" && a.outputFunc != nil {
		remaining := flushSentences(content, a.outputFunc)
		if remaining != "" {
			a.outputFunc(remaining)
		}
	}

	return content, false, nil, nil
}

// appendToolCalls adds the assistant's tool call request to the conversation.
// spokenText may be non-empty when the model included spoken words in the same
// turn; it is preserved in the message so subsequent turns see the full context.
func (a *Actor) appendToolCalls(conversation *[]message.Message, toolCalls []message.ToolCall, spokenText string) {
	*conversation = append(*conversation, message.Tool{
		Role:      "assistant",
		Content:   spokenText,
		ToolCalls: toolCalls,
	})
}

// appendAssistant adds the actor's text response to the conversation.
func (a *Actor) appendAssistant(conversation *[]message.Message, content string) {
	if content == "" {
		return
	}

	*conversation = append(*conversation, message.Chat{Role: "assistant", Content: content})
}

// callTools looks up requested tools by name and executes them.
func (a *Actor) callTools(ctx context.Context, toolCalls []message.ToolCall) []message.Message {
	resps := make([]message.Message, 0, len(toolCalls))

	for _, toolCall := range toolCalls {
		tool, exists := a.tools[toolCall.Function.Name]
		if !exists {
			// Fallback: try a normalized name match (strip underscores/hyphens,
			// lowercase) to handle models that elide punctuation in tool names
			// (e.g. Gemma 4 outputs "toolmovement" for "tool_movement").
			norm := normalizeToolName(toolCall.Function.Name)
			for registeredName, t := range a.tools {
				if normalizeToolName(registeredName) == norm {
					tool = t
					exists = true
					break
				}
			}
		}
		if !exists {
			log.Printf("\u001b[91mUnknown tool: %s\u001b[0m\n", toolCall.Function.Name)
			continue
		}

		log.Printf("\u001b[92m%s(%v)\u001b[0m: ", toolCall.Function.Name, toolCall.Function.Arguments)

		content := tool.Call(ctx, toolCall)
		if strings.Contains(content, `"FAILED"`) {
			log.Printf("\u001b[91m%s\u001b[0m\n", content)
		}

		resps = append(resps, message.ToolResponse{
			Role:    "tool",
			Name:    toolCall.Function.Name,
			Content: content,
		})
	}

	return resps
}

// InstallSystem downloads the llama.cpp libraries (to YZMA_LIB) and a model
// from the given URL (to ~/models/). It returns the local path to the model file.
func InstallSystem(modelURL string) (string, error) {
	libPath := os.Getenv("YZMA_LIB")
	if libPath == "" {
		return "", fmt.Errorf("YZMA_LIB environment variable not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancel()

	if !download.AlreadyInstalled(libPath) {
		version, err := download.LlamaLatestVersion()
		if err != nil {
			return "", fmt.Errorf("unable to get latest llama.cpp version: %w", err)
		}

		log.Println("installing llama.cpp libraries...")
		if err := download.GetWithContext(ctx, runtime.GOARCH, runtime.GOOS, "cpu", version, libPath, download.ProgressTracker); err != nil {
			return "", fmt.Errorf("unable to install llama.cpp: %w", err)
		}
	}

	modelsDir := download.DefaultModelsDir()
	if err := os.MkdirAll(modelsDir, 0o750); err != nil {
		return "", fmt.Errorf("unable to create models directory: %w", err)
	}

	modelPath := filepath.Join(modelsDir, modelFilename(modelURL))
	log.Println("downloading model...")
	if err := download.GetModelWithContext(ctx, modelURL, modelPath, download.ProgressTracker); err != nil {
		return "", fmt.Errorf("unable to download model: %w", err)
	}

	return modelPath, nil
}

// marshalToolDocs formats a slice of tool documents as a JSON array string for
// injection into the system prompt.
func marshalToolDocs(docs []message.ToolDefinition) string {
	b, err := json.MarshalIndent(docs, "", "  ")
	if err != nil {
		return "[]"
	}
	return string(b)
}

// injectToolsIntoSystemPrompt appends tool definitions and usage instructions
// to the system prompt using the format appropriate for the model.
func injectToolsIntoSystemPrompt(systemPrompt, toolsJSON string, format message.Format) string {
	if toolsJSON == "" || toolsJSON == "[]" {
		return systemPrompt
	}
	if format == message.FormatGemma {
		return systemPrompt + fmt.Sprintf(`

You have access to the following tools:
%s

To use a tool, include call: syntax directly in your response alongside your spoken text:
  call:funcname{key:<|"|>value<|"|>}

Example of a correct response — tool calls and spoken text in the same turn:
  call:tool_movement{command:<|"|>speak<|"|>}
  Hello! I'm doing wonderfully today.
  call:tool_movement{command:<|"|>wait<|"|>}

You MUST always include spoken text in the same response as any tool calls.
A response containing only tool calls with no spoken text is incomplete and invalid.`, toolsJSON)
	}
	return systemPrompt + fmt.Sprintf(`

You have access to the following tools:
%s

When you need to use a tool, respond with a tool call in the following format:
<tool_call>
{"name": "function_name", "arguments": {"arg1": "value1"}}
</tool_call>
After receiving tool results, continue your response normally.`, toolsJSON)
}

// modelFilename extracts a safe filename from a model URL.
func modelFilename(rawURL string) string {
	base := filepath.Base(rawURL)
	if i := strings.IndexByte(base, '?'); i >= 0 {
		base = base[:i]
	}
	return base
}

// normalizeToolName returns a lowercase version of name with underscores and
// hyphens removed, used as a fallback key when a model elides punctuation from
// tool names (e.g. Gemma 4 outputs "toolmovement" for "tool_movement").
func normalizeToolName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "_", "")
	name = strings.ReplaceAll(name, "-", "")
	return name
}

// flushSentences calls fn for each complete sentence found in buf (delimited
// by '.', '!' or '?' followed by whitespace or end-of-string) and returns any
// remaining partial sentence.
func flushSentences(buf string, fn func(string)) string {
	for {
		idx := -1
		for i := 0; i < len(buf); i++ {
			c := buf[i]
			if c == '.' || c == '!' || c == '?' {
				if i+1 >= len(buf) || buf[i+1] == ' ' || buf[i+1] == '\n' || buf[i+1] == '\t' {
					idx = i
					break
				}
			}
		}
		if idx < 0 {
			break
		}
		if sentence := strings.TrimSpace(buf[:idx+1]); sentence != "" {
			fn(sentence)
		}
		buf = strings.TrimLeft(buf[idx+1:], " \n\t")
	}
	return buf
}
