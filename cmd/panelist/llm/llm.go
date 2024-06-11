package llm

import (
	"context"
	"log"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/prompts"
)

type Config struct {
	ModelName       string
	HistSize        uint
	SeedPrompt      string
	InitialQuestion string
	InitialResponse string
}

type LLM struct {
	config     Config
	model      *ollama.LLM
	seedPrompt string
	histSize   uint
	history    *memory.ChatMessageHistory
}

func New(c Config) (*LLM, error) {
	model, err := ollama.New(ollama.WithModel(c.ModelName))
	if err != nil {
		return nil, err
	}

	return &LLM{
		config:     c,
		model:      model,
		seedPrompt: c.SeedPrompt,
		histSize:   c.HistSize,
		history:    memory.NewChatMessageHistory(),
	}, nil
}

func (l *LLM) Stream(ctx context.Context, questions chan llms.HumanChatMessage, replyChunks chan string, replies chan llms.AIChatMessage, others chan llms.GenericChatMessage) error {
	log.Println("launching LLM stream")
	defer log.Println("done streaming LLM")

	promptTmpl := prompts.NewChatPromptTemplate([]prompts.MessageFormatter{
		prompts.NewSystemMessagePromptTemplate(
			l.seedPrompt,
			nil,
		),
		prompts.NewGenericMessagePromptTemplate(
			"history",
			"{{range .historyMessages}}{{.GetContent}}\n{{end}}",
			[]string{"history"},
		),
		prompts.NewHumanMessagePromptTemplate(
			`{{.question}}`,
			[]string{"question"},
		),
	})

	l.history.AddUserMessage(ctx, l.config.InitialQuestion)
	l.history.AddAIMessage(ctx, l.config.InitialResponse)

	buf := NewFixedSizeBuffer(MaxTTSBufferSize)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case question := <-questions:
			historyMessages, _ := l.history.Messages(ctx)
			prompt, _ := promptTmpl.FormatPrompt(map[string]any{
				"historyMessages": historyMessages,
				"question":        question.GetContent(),
			})

			mc := chatMessagesToMessageContent(prompt.Messages())

			response, err := l.model.GenerateContent(ctx, mc,
				llms.WithStreamingFunc(func(_ context.Context, chunk []byte) error {
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
						if len(chunk) == 0 {
							// send the full reply
							replyChunks <- buf.String()
							buf.Reset()
							return nil
						}

						_, err := buf.Write(chunk)
						if err != nil {
							if err == ErrBufferFull {
								replyChunks <- buf.String()
								buf.Reset()
								// NOTE: flush resets the buffer and we need
								// to write the remaining chunks to it which
								// have not fitted into the buffer on the last write.
								if _, err := buf.Write(chunk); err != nil {
									return err
								}
								return nil
							}
							return err
						}

						// does the last token end in a period? if so, send the sentence.
						if chunk[len(chunk)-1] == '.' {
							// send the full reply
							replyChunks <- buf.String()
							buf.Reset()
						}

						return nil
					}
				}))
			if err != nil {
				return err
			}

			l.history.AddMessage(ctx, question)
			for _, r := range response.Choices {
				aiMsg := llms.AIChatMessage{Content: r.Content}
				l.history.AddMessage(ctx, aiMsg)
				if replies != nil {
					replies <- aiMsg
				}
			}

			// only used for mqtt
		case other := <-others:
			l.history.AddMessage(ctx, other)
		}
	}
}

func chatMessagesToMessageContent(chat []llms.ChatMessage) []llms.MessageContent {
	mcs := make([]llms.MessageContent, len(chat))
	for i, msg := range chat {
		role := msg.GetType()
		text := msg.GetContent()

		var mc llms.MessageContent

		switch p := msg.(type) {
		case llms.ToolChatMessage:
			mc = llms.MessageContent{
				Role: role,
				Parts: []llms.ContentPart{llms.ToolCallResponse{
					ToolCallID: p.ID,
					Content:    p.Content,
				}},
			}

		case llms.AIChatMessage:
			mc = llms.MessageContent{
				Role: role,
				Parts: []llms.ContentPart{
					llms.ToolCall{
						ID:           p.ToolCalls[0].ID,
						Type:         p.ToolCalls[0].Type,
						FunctionCall: p.ToolCalls[0].FunctionCall,
					},
				},
			}
		default:
			mc = llms.MessageContent{
				Role:  role,
				Parts: []llms.ContentPart{llms.TextContent{Text: text}},
			}
		}
		mcs[i] = mc
	}

	return mcs
}
