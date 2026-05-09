package actor

import "github.com/hybridgroup/yzma/pkg/message"

var defaultPauseWords = []string{
	"let me think...", "let me see...", "let me consider that...", "let me ponder that...",
	"let me figure that out...", "let me work that out...",
	"let's see...", "let's think...", "let's see now...", "let's think about that...",
	"that is interesting...", "well, that is interesting...", "ok, that is interesting...",
	"now then...", "well then...", "well now...", "right then...",
	"I see...", "I see now...",
	"one moment...", "one moment please...",
	"give me a second...", "give me a moment...",
	"hold on a moment...", "hold on a second...",
	"I'm thinking...", "I'm working on that...",
	"let me consider that briefly...",
	"well let me see...", "now let me think...",
}

const (
	DefaultTemperature     = float32(0.6)
	DefaultTopP            = float32(0.95)
	DefaultMinP            = float32(0.05)
	DefaultTopK            = int32(20)
	DefaultMaxTokens       = 2048
	DefaultContextSize     = 4096
	DefaultRepeatPenalty   = float32(1.0) // 1.0 = disabled
	DefaultFreqPenalty     = float32(0.0) // 0.0 = disabled
	DefaultPresencePenalty = float32(0.0) // 0.0 = disabled
	DefaultDryMultiplier   = float32(0.0) // 0.0 = disabled
	DefaultBatchSize       = uint32(0)    // 0 = use llama.cpp default
	DefaultUBatchSize      = uint32(0)    // 0 = use llama.cpp default
	DefaultPauseInterval   = 3            // seconds between repeated pause words
)

// Config holds the tunable parameters for an Actor.
type Config struct {
	Temperature float32
	TopP        float32
	MinP        float32
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
	// PauseWords is a list of filler words (e.g. "well", "hmm") that the actor
	// speaks immediately after receiving a new request, before the model has
	// produced its first token, to mask the inference startup latency.
	// Set to nil or an empty slice to disable.
	PauseWords []string
	// PauseInterval is the number of seconds between repeated pause words while
	// waiting for the model to produce its first token. Defaults to DefaultPauseInterval.
	PauseInterval int
}

// DefaultConfig returns a Config populated with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Temperature:     DefaultTemperature,
		TopP:            DefaultTopP,
		MinP:            DefaultMinP,
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
		PauseWords:      defaultPauseWords,
		PauseInterval:   DefaultPauseInterval,
	}
}
