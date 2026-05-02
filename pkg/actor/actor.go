package actor

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/hybridgroup/yzma/pkg/llama"
	"github.com/hybridgroup/yzma/pkg/message"
	"github.com/hybridgroup/yzma/pkg/template"
)

const (
	DefaultTemperature     = float32(0.6)
	DefaultTopP            = float32(0.95)
	DefaultTopK            = int32(20)
	DefaultMaxTokens       = 2048
	DefaultContextSize     = 4096
	DefaultRepeatPenalty   = float32(1.0) // 1.0 = disabled
	DefaultFreqPenalty     = float32(0.0) // 0.0 = disabled
	DefaultPresencePenalty = float32(0.0) // 0.0 = disabled
	DefaultDryMultiplier   = float32(0.0) // 0.0 = disabled
	DefaultBatchSize       = uint32(0)    // 0 = use llama.cpp default
	DefaultUBatchSize      = uint32(0)    // 0 = use llama.cpp default
)

// Config holds the tunable parameters for an Actor.
type Config struct {
	Temperature float32
	TopP        float32
	TopK        int32
	MaxTokens   int
	ContextSize uint32
	// BatchSize is the logical maximum batch size (n_batch). 0 = use llama.cpp default.
	BatchSize uint32
	// UBatchSize is the physical maximum micro-batch size (n_ubatch). 0 = use llama.cpp default.
	UBatchSize uint32
	// RepeatPenalty penalises recently-seen tokens to reduce repetition.
	// 1.0 = disabled; values around 1.1–1.3 are effective for verbose models.
	RepeatPenalty float32
	// FreqPenalty penalises tokens proportional to how often they have appeared.
	// 0.0 = disabled.
	FreqPenalty float32
	// PresencePenalty penalises any token that has appeared at all.
	// 0.0 = disabled.
	PresencePenalty float32
	// DryMultiplier enables DRY (Don't Repeat Yourself) repetition penalty.
	// 0.0 = disabled; values around 0.8 are a good starting point.
	DryMultiplier float32
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
	// UseMmap controls whether the model file is loaded via mmap.
	// Defaults to true. Set to false to disable mmap (e.g. when loading from
	// a network filesystem that does not support mmap).
	UseMmap bool
	// Verbose enables verbose logging for debugging.
	Verbose bool
}

// DefaultConfig returns a Config populated with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Temperature:     DefaultTemperature,
		TopP:            DefaultTopP,
		TopK:            DefaultTopK,
		MaxTokens:       DefaultMaxTokens,
		ContextSize:     DefaultContextSize,
		BatchSize:       DefaultBatchSize,
		UBatchSize:      DefaultUBatchSize,
		RepeatPenalty:   DefaultRepeatPenalty,
		FreqPenalty:     DefaultFreqPenalty,
		PresencePenalty: DefaultPresencePenalty,
		DryMultiplier:   DefaultDryMultiplier,
		InjectTools:     true,
		EnableThinking:  false,
		UseMmap:         true,
		Verbose:         false,
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

	// nCachedPrompt is the number of prompt tokens currently held in the KV /
	// recurrent cache from the previous turn. It is used to implement
	// incremental decoding: only tokens beyond this offset need to be decoded
	// each turn, saving the cost of re-decoding the entire conversation history.
	nCachedPrompt int

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

	if !cfg.Verbose {
		llama.LogSet(llama.LogSilent())
	}
	llama.Init()

	log.Println("loading model...")

	modelParams := llama.ModelDefaultParams()
	if !cfg.UseMmap {
		modelParams.UseMmap = 0
	}
	mdl, err := llama.ModelLoadFromFile(modelPath, modelParams)
	if err != nil {
		return nil, fmt.Errorf("unable to load model: %w", err)
	}

	ctxParams := llama.ContextDefaultParams()
	ctxParams.NCtx = cfg.ContextSize
	if cfg.BatchSize > 0 {
		ctxParams.NBatch = cfg.BatchSize
	}
	if cfg.UBatchSize > 0 {
		ctxParams.NUbatch = cfg.UBatchSize
	}
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
	sp.PenaltyRepeat = cfg.RepeatPenalty
	sp.PenaltyFreq = cfg.FreqPenalty
	sp.PenaltyPresent = cfg.PresencePenalty
	sp.DryMultiplier = cfg.DryMultiplier
	smpl := llama.NewSampler(mdl, llama.DefaultSamplers, sp)

	// Auto-detect model format from the model path when not explicitly set.
	if cfg.ModelFormat == message.FormatAuto {
		cfg.ModelFormat = message.DetectFormatFromPath(modelPath)
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
	// Set the abort callback once for the duration of the run so that
	// llama.Decode is interrupted immediately when the context is cancelled.
	// Done here rather than per-turn to avoid a log line on every turn.
	llama.SetAbortCallback(a.llamaCtx, func() bool { return ctx.Err() != nil })
	defer llama.SetAbortCallback(a.llamaCtx, nil)

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
			before := len(conversation)
			if ok := a.GetMore(&conversation); !ok {
				return nil
			}
			if len(conversation) == before {
				// moreFunc returned without adding anything — check if context
				// was cancelled, otherwise loop back and wait for the next message.
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					continue
				}
			}
		}

		content, hadText, toolCalls, err := a.generateTurn(ctx, &conversation)
		if err != nil {
			return err
		}

		if len(toolCalls) > 0 {
			toolResults := a.callTools(ctx, toolCalls)

			if !a.templateSupportsToolMessages() {
				// Format has no tool-role support (e.g. Gemma 3). Tool calls are
				// fire-and-forget physical action cues; never append tool call or
				// tool result messages to the conversation history.
				if hadText {
					a.appendAssistant(&conversation, content)
					consecutiveToolOnlyTurns = 0
					needMoreInput = true
				} else {
					consecutiveToolOnlyTurns++
					if consecutiveToolOnlyTurns >= maxConsecutiveToolOnlyTurns {
						log.Printf("breaking tool-call loop after %d consecutive tool-only turns", consecutiveToolOnlyTurns)
						consecutiveToolOnlyTurns = 0
						needMoreInput = true
					} else {
						conversation = append(conversation, message.Chat{
							Role:    "user",
							Content: "You called motion tools but included no spoken words. You MUST write your actual answer as plain text. Reply now with spoken sentences.",
						})
						log.Printf("tool-only turn %d/%d, nudging for verbal response", consecutiveToolOnlyTurns, maxConsecutiveToolOnlyTurns)
						needMoreInput = false
					}
				}
				continue
			}

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
						Content: "You called motion tools but included no spoken words. Note: calling tool_movement with command 'speak' is a head-motion cue — it is NOT a verbal response. You MUST write your actual answer as plain text outside any function blocks. Reply now with spoken sentences.",
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

	llama.SamplerReset(a.sampler)

	tokens := llama.Tokenize(a.vocab, prompt, true, true)

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

	mem, err := llama.GetMemory(a.llamaCtx)
	if err != nil {
		return "", false, nil, fmt.Errorf("error getting memory: %w", err)
	}

	// Incremental decode: reuse the cached prompt prefix from the previous turn
	// and only decode the tokens added since then. This avoids re-decoding the
	// entire conversation history on every turn (the main source of latency on
	// slow hardware). When the prompt shrank (context trimming dropped old
	// messages), the cached prefix is no longer valid — fall back to full clear.
	var decodeFrom int
	t0 := time.Now()
	if a.nCachedPrompt > 0 && len(tokens) >= a.nCachedPrompt {
		// Trim the tail of the cache (generated tokens from last turn) so that
		// positions nCachedPrompt..∞ are freed, keeping the prompt prefix intact.
		if ok, rmErr := llama.MemorySeqRm(mem, 0, llama.Pos(a.nCachedPrompt), -1); ok && rmErr == nil {
			decodeFrom = a.nCachedPrompt
			if a.cfg.Verbose {
				log.Printf("[verbose] cache trim: kept %d cached tokens, removed tail (%v)", a.nCachedPrompt, time.Since(t0))
			}
		} else {
			// MemorySeqRm failed (shouldn't happen): full clear.
			if clearErr := llama.MemoryClear(mem, true); clearErr != nil {
				return "", false, nil, fmt.Errorf("error clearing memory: %w", clearErr)
			}
			a.nCachedPrompt = 0
			if a.cfg.Verbose {
				log.Printf("[verbose] cache trim failed, full clear (%v)", time.Since(t0))
			}
		}
	} else {
		// First turn or prompt shrank due to context trimming: full clear.
		if clearErr := llama.MemoryClear(mem, true); clearErr != nil {
			return "", false, nil, fmt.Errorf("error clearing memory: %w", clearErr)
		}
		a.nCachedPrompt = 0
		if a.cfg.Verbose {
			log.Printf("[verbose] memory clear: %v", time.Since(t0))
		}
	}

	nBatch := int(llama.NBatch(a.llamaCtx))
	if nBatch <= 0 {
		nBatch = 512
	}
	if a.cfg.Verbose {
		log.Printf("[verbose] prompt decode: %d new tokens (of %d total), batch size %d", len(tokens)-decodeFrom, len(tokens), nBatch)
	}
	t1 := time.Now()
	for i := decodeFrom; i < len(tokens); i += nBatch {
		end := i + nBatch
		if end > len(tokens) {
			end = len(tokens)
		}
		if _, err := llama.Decode(a.llamaCtx, llama.BatchGetOne(tokens[i:end])); err != nil {
			return "", false, nil, fmt.Errorf("error decoding prompt: %w", err)
		}
	}
	if a.cfg.Verbose {
		log.Printf("[verbose] prompt decode done: %v", time.Since(t1))
	}

	a.nCachedPrompt = len(tokens)

	var chunks []string
	pieceBuf := make([]byte, 128)

	generationStopMarkers := message.StopMarkers(a.vocab, a.cfg.ModelFormat)

	t2 := time.Now()
	tokenCount := 0

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
		tokenCount++

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
			toolCallCount := strings.Count(full, "</tool_call>") + strings.Count(full, "</function>")
			// Also count Gemma/Gemma3 call: blocks, which use a closing brace
			// instead of an XML tag.
			if a.cfg.ModelFormat == message.FormatGemma || a.cfg.ModelFormat == message.FormatGemma3 {
				toolCallCount += strings.Count(full, "call:tool_")
			}
			if toolCallCount >= maxToolCallBlocksPerGeneration {
				log.Printf("stopping generation: accumulated %d tool call blocks", toolCallCount)
				break generateLoop
			}
		}

		if _, err := llama.Decode(a.llamaCtx, llama.BatchGetOne([]llama.Token{token})); err != nil {
			break
		}
	}

	if a.cfg.Verbose {
		elapsed := time.Since(t2)
		tps := float64(tokenCount) / elapsed.Seconds()
		log.Printf("[verbose] generation: %d tokens in %v (%.2f t/s)", tokenCount, elapsed, tps)
	}

	text := strings.TrimSpace(strings.TrimLeft(strings.Join(chunks, ""), "\n"))

	// Strip <think> / </think> tags before any further processing. Qwen3 with
	// no_think still emits </think> after function blocks, which confuses
	// StripMarkup: its orphaned-</think> handler strips everything before the
	// first </think> (including the text preceding the first function call).
	text = strings.ReplaceAll(text, "<think>", "")
	text = strings.ReplaceAll(text, "</think>", "")
	text = strings.TrimSpace(text)

	log.Printf("raw generation: %q", text)

	toolCalls := message.ParseToolCalls(text)
	if len(toolCalls) > 0 {
		var hadText bool
		var spokenText string
		if spokenText = stripActorMarkup(text); spokenText != "" && a.outputFunc != nil {
			remaining := flushSentences(spokenText, a.outputFunc)
			if remaining != "" {
				a.outputFunc(remaining)
			}
			hadText = true
		}
		return spokenText, hadText, toolCalls, nil
	}

	content := stripActorMarkup(text)
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
			// Fallback: some models (e.g. Phi-4) call movement commands directly
			// by name (e.g. name="slowlook") instead of using tool_movement with
			// command="slowlook". Reroute those to tool_movement.
			if movementTool, ok := a.tools["tool_movement"]; ok {
				switch toolCall.Function.Name {
				case "look", "slowlook", "headshake":
					if toolCall.Function.Arguments == nil {
						toolCall.Function.Arguments = map[string]string{}
					}
					toolCall.Function.Arguments["command"] = toolCall.Function.Name
					toolCall.Function.Name = "tool_movement"
					tool = movementTool
					exists = true
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

// normalizeToolName returns a lowercase version of name with underscores and
// hyphens removed, used as a fallback key when a model elides punctuation from
// tool names (e.g. Gemma 4 outputs "toolmovement" for "tool_movement").
func normalizeToolName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "_", "")
	name = strings.ReplaceAll(name, "-", "")
	return name
}

// templateSupportsToolMessages reports whether the model's chat template
// supports dedicated tool and tool-result conversation roles. When false
// (e.g. Gemma 3), tool calls are treated as fire-and-forget physical action
// cues and their results are never appended to the conversation history.
func (a *Actor) templateSupportsToolMessages() bool {
	return a.cfg.ModelFormat != message.FormatGemma3
}

// orphanAngleRE matches bare "angle:N" tokens that models write outside the
// tool-call braces instead of inside them — never spoken text.
var orphanAngleRE = regexp.MustCompile(`\bangle:\d+\b`)

// stageDirectionRE matches multi-word parenthetical stage directions such as
// "(character tilts head to the left)" that models write instead of tool calls.
// Single-word parentheticals like "(five)" are preserved.
var stageDirectionRE = regexp.MustCompile(`\([^)]*\s[^)]*\)`)

// stripActorMarkup calls message.StripMarkup and then removes artefacts that
// are specific to the talkingheads actor (orphaned angle parameters, stage
// directions). This keeps the yzma library general-purpose.
func stripActorMarkup(s string) string {
	s = message.StripMarkup(s)
	s = orphanAngleRE.ReplaceAllString(s, "")
	s = stageDirectionRE.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
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
