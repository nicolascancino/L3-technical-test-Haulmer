package repository

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/domain"
)

func TestCursorRoundTrip(t *testing.T) {
	want := cursor{
		CreatedAt: time.Date(2026, 10, 10, 12, 30, 45, 123456000, time.UTC),
		ID:        uuid.Must(uuid.NewV7()),
	}

	got, err := decodeCursor(encodeCursor(want))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) || got.ID != want.ID {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestDecodeCursorRejectsMalformedInput(t *testing.T) {
	tests := map[string]string{
		"empty":          "",
		"not base64":     "!!!",
		"no separator":   "MTIzNA",
		"bad timestamp":  "YWJjfDAxOTBhYjEyLTAwMDAtNzAwMC04MDAwLTAwMDAwMDAwMDAwMA",
		"bad uuid":       "MTIzNHxub3QtYS11dWlk",
		"empty id":       "MTIzNHw",
		"extra segments": "MTIzfDQ1NnxhYmM",
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := decodeCursor(input)

			if !errors.Is(err, domain.ErrInvalidCursor) {
				t.Fatalf("got %v, want ErrInvalidCursor", err)
			}
		})
	}
}
