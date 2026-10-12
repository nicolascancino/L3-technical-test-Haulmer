package repository

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/domain"
)

func TestParseCursorAcceptsUUID(t *testing.T) {
	want := uuid.Must(uuid.NewV7())

	got, err := parseCursor(want.String())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestParseCursorRejectsMalformedInput(t *testing.T) {
	tests := map[string]string{
		"empty":     "",
		"not uuid":  "!!!",
		"truncated": "0190ab12-0000-7000-8000",
		"extra":     "0190ab12-0000-7000-8000-000000000000-00",
		"non hex":   "0190ab12-0000-7000-8000-00000000000z",
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := parseCursor(input)

			if !errors.Is(err, domain.ErrInvalidCursor) {
				t.Fatalf("got %v, want ErrInvalidCursor", err)
			}
		})
	}
}

func TestEscapeLike(t *testing.T) {
	tests := map[string]struct{ in, want string }{
		"plain":      {"Ada", "Ada"},
		"percent":    {"a%b", `a\%b`},
		"underscore": {"a_b", `a\_b`},
		"backslash":  {`a\b`, `a\\b`},
		"all":        {`%_\`, `\%\_\\`},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := escapeLike(tt.in); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
