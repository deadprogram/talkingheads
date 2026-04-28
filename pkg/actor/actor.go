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
	defaultTemperature = float32(0.6)
	defaultTopP        = float32(0.95)
	defaultTopK        = int32(20)
	maxPredictTokens   = 2048
)

// Actor drives a conversation using a local llama.cpp model loaded via yzma.
type Actor struct {
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
func NewActor(modelPath string, commander Commander, moreFunc func(conversation *[]message.Message), outputFunc func(content string)) (*Actor, error) {
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
	ctxParams.NCtx = 4096
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
	sp.Temp = defaultTemperature
	sp.TopP = defaultTopP
	sp.TopK = defaultTopK
	smpl := llama.NewSampler(mdl, llama.DefaultSamplers, sp)

	toolsMap := make(map[string]Tool)
	toolDocs := []ToolDoc{
		RegisterMovement(toolsMap, commander),
	}

	return &Actor{
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
	conversation := []message.Message{
		message.Chat{Role: "system", Content: injectToolsIntoSystemPrompt(systemPrompt, a.toolsJSON)},
	}

	needMoreInput := true

	for {
		if needMoreInput {
			if ok := a.GetMore(&conversation); !ok {
				return nil
			}
		}

		content, toolCalls, err := a.generateTurn(ctx, &conversation)
		if err != nil {
			return err
		}

		if len(toolCalls) > 0 {
			a.appendToolCalls(&conversation, toolCalls)
			conversation = append(conversation, a.callTools(ctx, toolCalls)...)
			needMoreInput = false
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
// content and any tool calls parsed from the response.
// The conversation pointer may be updated to trim old messages if the prompt
// exceeds the model's context window.
func (a *Actor) generateTurn(ctx context.Context, conversation *[]message.Message) (string, []message.ToolCall, error) {
	prompt, err := template.Apply(a.chatTemplate, *conversation, true)
	if err != nil {
		return "", nil, fmt.Errorf("error applying chat template: %w", err)
	}

	mem, err := llama.GetMemory(a.llamaCtx)
	if err != nil {
		return "", nil, fmt.Errorf("error getting memory: %w", err)
	}
	if err := llama.MemoryClear(mem, true); err != nil {
		return "", nil, fmt.Errorf("error clearing memory: %w", err)
	}

	llama.SamplerReset(a.sampler)

	tokens := llama.Tokenize(a.vocab, prompt, true, false)

	// Trim oldest non-system messages if the prompt exceeds the context window.
	if nCtx := int(llama.NCtx(a.llamaCtx)); nCtx > 0 {
		maxPromptTokens := nCtx - maxPredictTokens
		for len(tokens) > maxPromptTokens && len(*conversation) > 2 {
			// Drop the second message (oldest non-system entry) and re-tokenize.
			log.Println("trimming oldest message from conversation to fit context window")
			*conversation = append((*conversation)[:1], (*conversation)[2:]...)
			prompt, err = template.Apply(a.chatTemplate, *conversation, true)
			if err != nil {
				return "", nil, fmt.Errorf("error applying chat template after trim: %w", err)
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
			return "", nil, fmt.Errorf("error decoding prompt: %w", err)
		}
	}

	var chunks []string
	var sentenceBuf string
	var inThinking bool
	var thinkHold string
	pieceBuf := make([]byte, 128)

	emitSentence := func(s string) {
		if a.outputFunc != nil {
			a.outputFunc(s)
		}
	}

	for i := 0; i < maxPredictTokens; i++ {
		select {
		case <-ctx.Done():
			return "", nil, ctx.Err()
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
			visible := filterThinkingPiece(piece, &inThinking, &thinkHold)
			if visible != "" {
				sentenceBuf += visible
				sentenceBuf = flushSentences(sentenceBuf, emitSentence)
			}
		}

		if _, err := llama.Decode(a.llamaCtx, llama.BatchGetOne([]llama.Token{token})); err != nil {
			break
		}
	}

	// Flush any partial tag hold buffer that turned out not to be a tag.
	if !inThinking && thinkHold != "" {
		sentenceBuf += thinkHold
	}

	text := strings.TrimSpace(strings.TrimLeft(strings.Join(chunks, ""), "\n"))

	toolCalls := message.ParseToolCalls(text)
	if len(toolCalls) > 0 {
		return "", toolCalls, nil
	}

	if remaining := strings.TrimSpace(sentenceBuf); remaining != "" {
		emitSentence(remaining)
	}

	return text, nil, nil
}

// appendToolCalls adds the assistant's tool call request to the conversation.
func (a *Actor) appendToolCalls(conversation *[]message.Message, toolCalls []message.ToolCall) {
	*conversation = append(*conversation, message.Tool{
		Role:      "assistant",
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
func marshalToolDocs(docs []ToolDoc) string {
	b, err := json.MarshalIndent(docs, "", "  ")
	if err != nil {
		return "[]"
	}
	return string(b)
}

// injectToolsIntoSystemPrompt appends the tool definitions and usage instructions
// to the system prompt.
func injectToolsIntoSystemPrompt(systemPrompt, toolsJSON string) string {
	if toolsJSON == "" || toolsJSON == "[]" {
		return systemPrompt
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

// filterThinkingPiece processes a newly arrived piece of text, filtering out
// content inside <think>...</think> blocks. inThinking and holdBuf are
// persistent state across calls. Returns the visible (non-thinking) content.
func filterThinkingPiece(piece string, inThinking *bool, holdBuf *string) string {
	combined := *holdBuf + piece
	*holdBuf = ""
	result := ""

	for combined != "" {
		if *inThinking {
			idx := strings.Index(combined, "</think>")
			if idx >= 0 {
				*inThinking = false
				combined = combined[idx+len("</think>"):]
			} else {
				// No closing tag yet; hold any partial suffix that could start one.
				*holdBuf = longestTagPrefix(combined, "</think>")
				combined = ""
			}
		} else {
			idx := strings.Index(combined, "<think>")
			if idx >= 0 {
				result += combined[:idx]
				*inThinking = true
				combined = combined[idx+len("<think>"):]
			} else {
				// No opening tag; hold any partial suffix that could start one.
				partial := longestTagPrefix(combined, "<think>")
				result += combined[:len(combined)-len(partial)]
				*holdBuf = partial
				combined = ""
			}
		}
	}
	return result
}

// longestTagPrefix returns the longest suffix of s that is a proper prefix of tag.
func longestTagPrefix(s, tag string) string {
	for i := len(tag) - 1; i > 0; i-- {
		if strings.HasSuffix(s, tag[:i]) {
			return tag[:i]
		}
	}
	return ""
}

// flushSentences calls fn for each complete sentence found in buf (delimited
// by '.', '!' or '?' followed by whitespace or end-of-string) and returns any
// remaining partial sentence that has not yet ended.
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
