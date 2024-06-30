package llm

import (
	"context"
	"log"
	"regexp"

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

	// l.history.AddUserMessage(ctx, l.config.InitialQuestion)
	// l.history.AddAIMessage(ctx, l.config.InitialResponse)

	buf := NewFixedSizeBuffer(MaxTTSBufferSize)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case question := <-questions:
			historyMessages, _ := l.history.Messages(ctx)
			promptMsgs, _ := promptTmpl.FormatMessages(map[string]any{
				"historyMessages": historyMessages,
				"question":        question.GetContent(),
			})

			mc := chatMessagesToMessageContent(promptMsgs)

			response, err := l.model.GenerateContent(ctx, mc,
				llms.WithStreamingFunc(func(_ context.Context, chunk []byte) error {
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
						if len(chunk) == 0 {
							// send the full reply
							replyChunks <- removeEmoji(buf.String())
							buf.Reset()
							return nil
						}

						_, err := buf.Write(chunk)
						if err != nil {
							if err == ErrBufferFull {
								replyChunks <- removeEmoji(buf.String())
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
							replyChunks <- removeEmoji(buf.String())
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
				aiMsg := llms.AIChatMessage{Content: removeEmoji(r.Content)}
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
		mcs[i] = chatMessageToMessageContent(msg)
	}

	return mcs
}

func chatMessageToMessageContent(msg llms.ChatMessage) llms.MessageContent {
	role := msg.GetType()
	text := msg.GetContent()

	switch p := msg.(type) {
	case llms.ToolChatMessage:
		return llms.MessageContent{
			Role: role,
			Parts: []llms.ContentPart{llms.ToolCallResponse{
				ToolCallID: p.ID,
				Content:    p.Content,
			}},
		}

	case llms.AIChatMessage:
		return llms.MessageContent{
			Role: role,
			Parts: []llms.ContentPart{
				llms.TextContent{Text: text},
				llms.ToolCall{
					ID:           p.ToolCalls[0].ID,
					Type:         p.ToolCalls[0].Type,
					FunctionCall: p.ToolCalls[0].FunctionCall,
				},
			},
		}

	case llms.GenericChatMessage:
		return llms.MessageContent{
			Role:  role,
			Parts: []llms.ContentPart{llms.TextContent{Text: p.Name + ": " + text}},
		}

	default:
		return llms.MessageContent{
			Role:  role,
			Parts: []llms.ContentPart{llms.TextContent{Text: text}},
		}
	}
}

func removeEmoji(str string) string {
	// Regex pattern to match most emoji characters
	emojiPattern := "[\U0001F600-\U0001F64F\U0001F300-\U0001F5FF\U0001F680-\U0001F6FF\U0001F700-\U0001F77F\U0001F780-\U0001F7FF\U0001F800-\U0001F8FF\U0001F900-\U0001F9FF\U00002702-\U000027B0\U000024C2-\U0001F251]+"
	re := regexp.MustCompile(emojiPattern)
	// Replace matched emoji with an empty string to remove it
	return re.ReplaceAllString(str, "")
}
