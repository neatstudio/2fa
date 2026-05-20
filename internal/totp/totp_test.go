package totp

import (
	"testing"
	"time"
)

func TestGenerateMatchesRFC6238SHA1Vectors(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	tests := []struct {
		at   int64
		want string
	}{
		{59, "94287082"},
		{1111111109, "07081804"},
		{1111111111, "14050471"},
		{1234567890, "89005924"},
		{2000000000, "69279037"},
		{20000000000, "65353130"},
	}

	for _, tt := range tests {
		code, err := Generate(secret, time.Unix(tt.at, 0).UTC(), Options{Period: 30, Digits: 8})
		if err != nil {
			t.Fatalf("Generate() returned error: %v", err)
		}
		if code.Value != tt.want {
			t.Fatalf("Generate(%d) = %q, want %q", tt.at, code.Value, tt.want)
		}
	}
}

func TestGenerateAcceptsSpacedLowercaseBase32(t *testing.T) {
	at := time.Unix(59, 0).UTC()
	clean, err := Generate("GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ", at, Options{Period: 30, Digits: 6})
	if err != nil {
		t.Fatalf("clean secret failed: %v", err)
	}

	spaced, err := Generate(" ge zdgnbv gy3tqojq gezdgnbvgy3tqojq ", at, Options{Period: 30, Digits: 6})
	if err != nil {
		t.Fatalf("spaced secret failed: %v", err)
	}

	if spaced.Value != clean.Value {
		t.Fatalf("spaced lowercase code = %q, want %q", spaced.Value, clean.Value)
	}
}

func TestGenerateDefaultCodeHasSixDigitsAndRemainingSeconds(t *testing.T) {
	code, err := GenerateDefault("GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ", time.Unix(59, 0).UTC())
	if err != nil {
		t.Fatalf("GenerateDefault() returned error: %v", err)
	}
	if len(code.Value) != 6 {
		t.Fatalf("code length = %d, want 6", len(code.Value))
	}
	if code.Remaining != 1 {
		t.Fatalf("remaining = %d, want 1", code.Remaining)
	}
}
