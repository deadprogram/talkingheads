package actor

import "testing"

func TestDefaultConfig_SamplingDefaults(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Temperature != DefaultTemperature {
		t.Errorf("Temperature: got %v, want %v", cfg.Temperature, DefaultTemperature)
	}
	if cfg.TopP != DefaultTopP {
		t.Errorf("TopP: got %v, want %v", cfg.TopP, DefaultTopP)
	}
	if cfg.MinP != DefaultMinP {
		t.Errorf("MinP: got %v, want %v", cfg.MinP, DefaultMinP)
	}
	if cfg.TopK != DefaultTopK {
		t.Errorf("TopK: got %v, want %v", cfg.TopK, DefaultTopK)
	}
	if cfg.MaxSentences != DefaultMaxSentences {
		t.Errorf("MaxSentences: got %v, want %v", cfg.MaxSentences, DefaultMaxSentences)
	}
}
