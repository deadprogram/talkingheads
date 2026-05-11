package main

import "testing"

func TestStripSayPrefix(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		wantRest string
		wantOK   bool
	}{
		{"plain", "say hello world", "hello world", true},
		{"capitalized", "Say hello", "hello", true},
		{"colon suffix", "say: hello", "hello", true},
		{"comma suffix", "say, hello", "hello", true},
		{"with quotes", `say "hello world"`, `"hello world"`, true},
		{"leading space", "   say hello", "hello", true},
		{"only say", "say", "", true},
		{"not say", "tell me a joke", "tell me a joke", false},
		{"say embedded", "please say hi", "please say hi", false},
		{"empty", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rest, ok := stripSayPrefix(tc.in)
			if ok != tc.wantOK {
				t.Fatalf("ok: got %v, want %v", ok, tc.wantOK)
			}
			if rest != tc.wantRest {
				t.Errorf("rest: got %q, want %q", rest, tc.wantRest)
			}
		})
	}
}

func TestTrimSurroundingQuotes(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{`"hello"`, "hello"},
		{`  "hello world"  `, "hello world"},
		{`hello`, "hello"},
		{`"hello`, `"hello`},
		{`hello"`, `hello"`},
		{`""`, ""},
		{`"`, `"`},
		{``, ``},
		{`"a "quoted" b"`, `a "quoted" b`},
	}
	for _, tc := range cases {
		got := trimSurroundingQuotes(tc.in)
		if got != tc.want {
			t.Errorf("trimSurroundingQuotes(%q): got %q, want %q", tc.in, got, tc.want)
		}
	}
}
