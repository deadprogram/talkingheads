package dialogue

import "testing"

func TestSayAnythingEmptyString(t *testing.T) {
	v := &Voice{Name: "test"}
	if err := v.SayAnything(""); err != nil {
		t.Errorf("SayAnything(\"\") returned unexpected error: %v", err)
	}
}
