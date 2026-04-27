package actor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ardanlabs/kronk/sdk/kronk"
	"github.com/ardanlabs/kronk/sdk/kronk/model"
	"github.com/ardanlabs/kronk/sdk/tools/defaults"
	"github.com/ardanlabs/kronk/sdk/tools/libs"
	"github.com/ardanlabs/kronk/sdk/tools/models"
)

const (
	defaultTemperature = 0.0
	defaultTopP        = 0.1
	defaultTopK        = 1
)

type Actor struct {
	krn *kronk.Kronk

	// function that gets more conversation input and appends it to the conversation.
	// This allows the actor to get more input from the user or from MQTT server.
	moreConversationFunc func(conversation *[]model.D)
	// outputFunc is called with the actor's text response each time the model produces one.
	outputFunc    func(content string)
	tools         map[string]Tool
	toolDocuments []model.D
}

// NewActor creates a new instance of Actor.
func NewActor(mp models.Path, commander Commander, moreFunc func(conversation *[]model.D), outputFunc func(content string)) (*Actor, error) {
	if err := kronk.Init(); err != nil {
		return nil, fmt.Errorf("unable to init kronk: %w", err)
	}

	krn, err := newKronk(mp)
	if err != nil {
		return nil, fmt.Errorf("unable to create kronk instance: %w", err)
	}

	// Build tool documents by registering each tool with its own tools map.
	toolsMap := make(map[string]Tool)
	toolDocuments := []model.D{
		RegisterMovement(toolsMap, commander),
	}

	actor := Actor{
		krn:                  krn,
		moreConversationFunc: moreFunc,
		outputFunc:           outputFunc,
		tools:                toolsMap,
		toolDocuments:        toolDocuments,
	}

	return &actor, nil
}

// Run starts the actor and runs the chat loop.
func (a *Actor) Run(ctx context.Context, systemPrompt string) error {
	conversation := []model.D{
		{"role": "system", "content": systemPrompt},
	}

	needMoreInput := true

	for {
		if needMoreInput {
			if ok := a.GetMore(&conversation); !ok {
				return nil
			}
		}

		content, toolCalls, _, err := a.streamModelTurn(ctx, conversation)
		if err != nil {
			return err
		}

		if len(toolCalls) > 0 {
			a.appendToolCalls(&conversation, toolCalls)

			results := a.callTools(ctx, toolCalls)
			if len(results) > 0 {
				conversation = append(conversation, results...)
			}

			needMoreInput = false
			continue
		}

		a.appendAssistant(&conversation, content)
		needMoreInput = true
	}
}

// GetMore gets more input and appends it to the conversation.
func (a *Actor) GetMore(conversation *[]model.D) bool {
	if a.moreConversationFunc == nil {
		return false
	}

	a.moreConversationFunc(conversation)
	return true
}

// streamModelTurn sends the conversation to the model and streams back the
// response. It returns the assembled text content, any tool calls, and usage.
func (a *Actor) streamModelTurn(ctx context.Context, conversation []model.D) (string, []model.ResponseToolCall, *model.Usage, error) {
	d := model.D{
		"messages":       conversation,
		"temperature":    defaultTemperature,
		"top_p":          defaultTopP,
		"top_k":          defaultTopK,
		"tools":          a.toolDocuments,
		"tool_selection": "auto",
	}

	callCtx, cancelCall := context.WithTimeout(ctx, 5*time.Minute)
	defer cancelCall()

	ch, err := a.krn.ChatStreaming(callCtx, d)
	if err != nil {
		return "", nil, nil, fmt.Errorf("error chat streaming: %w", err)
	}

	var chunks []string
	var sentenceBuf string
	var lastResp model.ChatResponse
	firstChunk := true
	reasonThinking := false

	emitSentence := func(s string) {
		if a.outputFunc != nil {
			a.outputFunc(s)
		}
	}

	for resp := range ch {
		lastResp = resp

		if len(resp.Choices) == 0 {
			continue
		}

		if firstChunk {
			firstChunk = false
		}

		switch resp.Choices[0].FinishReason() {
		case model.FinishReasonError:
			return "", nil, lastResp.Usage, fmt.Errorf("error from model: %s", resp.Choices[0].Delta.Content)

		case model.FinishReasonStop:
			if remaining := strings.TrimSpace(sentenceBuf); remaining != "" {
				emitSentence(remaining)
			}
			text := strings.TrimLeft(strings.Join(chunks, ""), "\n")
			return text, nil, lastResp.Usage, nil

		case model.FinishReasonTool:
			return "", resp.Choices[0].Delta.ToolCalls, lastResp.Usage, nil

		default:
			delta := resp.Choices[0].Delta
			if delta.Content != "" {
				if reasonThinking {
					reasonThinking = false
				}

				chunks = append(chunks, delta.Content)
				sentenceBuf += delta.Content
				sentenceBuf = flushSentences(sentenceBuf, emitSentence)
			}
		}
	}

	// Stream ended without an explicit finish reason.
	if remaining := strings.TrimSpace(sentenceBuf); remaining != "" {
		emitSentence(remaining)
	}
	text := strings.TrimLeft(strings.Join(chunks, ""), "\n")
	return text, nil, lastResp.Usage, nil
}

// appendToolCalls adds the assistant's tool call request to the conversation.
func (a *Actor) appendToolCalls(conversation *[]model.D, toolCalls []model.ResponseToolCall) {
	var toolCallDocs []model.D
	for _, tc := range toolCalls {
		argsJSON, _ := json.Marshal(tc.Function.Arguments)
		toolCallDocs = append(toolCallDocs, model.D{
			"id":   tc.ID,
			"type": "function",
			"function": model.D{
				"name":      tc.Function.Name,
				"arguments": string(argsJSON),
			},
		})
	}

	*conversation = append(*conversation, model.D{
		"role":       "assistant",
		"tool_calls": toolCallDocs,
	})
}

// appendAssistant adds the actor's text response to the conversation.
func (a *Actor) appendAssistant(conversation *[]model.D, content string) {
	if content == "" {
		return
	}

	*conversation = append(*conversation, model.D{"role": "assistant", "content": content})
}

// callTools looks up requested tools by name and executes them.
func (a *Actor) callTools(ctx context.Context, toolCalls []model.ResponseToolCall) []model.D {
	resps := make([]model.D, 0, len(toolCalls))

	for _, toolCall := range toolCalls {
		tool, exists := a.tools[toolCall.Function.Name]
		if !exists {
			log.Printf("\u001b[91mUnknown tool: %s\u001b[0m\n", toolCall.Function.Name)
			continue
		}

		log.Printf("\u001b[92m%s(%v)\u001b[0m: ", toolCall.Function.Name, toolCall.Function.Arguments)

		resp := tool.Call(ctx, toolCall)

		content, _ := resp["content"].(string)
		if strings.Contains(content, `"FAILED"`) {
			log.Printf("\u001b[91m%s\u001b[0m\n", content)
		}

		resps = append(resps, resp)
	}

	return resps
}

// InstallSystem installs necessary system components like models and libraries, and returns the path to the installed model.
// InstallSystem downloads the model from the given URL and installs the
// required llama.cpp libraries. It returns the path to the installed model.
func InstallSystem(modelURL string) (models.Path, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancel()

	// Install llama.cpp libraries.
	libs, err := libs.New(libs.WithVersion(defaults.LibVersion("")))
	if err != nil {
		return models.Path{}, err
	}
	if _, err := libs.Download(ctx, kronk.FmtLogger); err != nil {
		return models.Path{}, fmt.Errorf("unable to install llama.cpp: %w", err)
	}

	// Download model.
	mdls, err := models.New()
	if err != nil {
		return models.Path{}, fmt.Errorf("unable to create models manager: %w", err)
	}

	mp, err := mdls.Download(ctx, kronk.FmtLogger, modelURL, "")
	if err != nil {
		return models.Path{}, fmt.Errorf("unable to install model: %w", err)
	}

	return mp, nil
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

func newKronk(mp models.Path) (*kronk.Kronk, error) {
	log.Println("loading model...")

	krn, err := kronk.New(
		model.WithModelFiles(mp.ModelFiles),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create inference model: %w", err)
	}

	return krn, nil
}
