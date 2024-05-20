package llm

import (
	"context"
	"fmt"
	"log"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

type Config struct {
	ModelName  string
	HistSize   uint
	SeedPrompt string
}

type LLM struct {
	model      *ollama.LLM
	seedPrompt string
	histSize   uint
}

func New(c Config) (*LLM, error) {
	model, err := ollama.New(ollama.WithModel(c.ModelName))
	if err != nil {
		return nil, err
	}
	return &LLM{
		model:      model,
		seedPrompt: c.SeedPrompt,
		histSize:   c.HistSize,
	}, nil
}

func sendChunk(ctx context.Context, chunks chan string, chunk []byte) error {
	select {
	case <-ctx.Done():
	case chunks <- string(chunk):
	}
	return nil
}

func (l *LLM) Stream(ctx context.Context, prompts chan string, replyChunks chan string) error {
	log.Println("launching LLM stream")
	defer log.Println("done streaming LLM")
	chat := NewHistory(int(l.histSize))
	chat.Add(l.seedPrompt)

	fmt.Println("Seed prompt: ", l.seedPrompt)

	buf := NewFixedSizeBuffer(MaxTTSBufferSize)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case prompt := <-prompts:
			chat.Add(prompt)
			_, err := llms.GenerateFromSinglePrompt(ctx, l.model, chat.String(),
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
		}
	}
}
