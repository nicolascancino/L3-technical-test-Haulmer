package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/domain"
)

func TestNewCard(t *testing.T) {
	now := time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)
	valid := domain.NewCardConfig{ID: "card-1", PAN: "4111111111111111", ExpMonth: 12, ExpYear: 2030, CreatedAt: now}

	tests := []struct {
		name    string
		mutate  func(*domain.NewCardConfig)
		wantErr error
	}{
		{name: "valid card", mutate: func(*domain.NewCardConfig) {}},
		{name: "missing id", mutate: func(c *domain.NewCardConfig) { c.ID = "" }, wantErr: domain.ErrIDIsRequired},
		{name: "missing created at", mutate: func(c *domain.NewCardConfig) { c.CreatedAt = time.Time{} }, wantErr: domain.ErrCreatedAtIsRequired},
		{name: "pan too short", mutate: func(c *domain.NewCardConfig) { c.PAN = "123" }, wantErr: domain.ErrInvalidPAN},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := valid
			tt.mutate(&cfg)

			card, err := domain.NewCard(cfg)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && card.Last4() != "1111" {
				t.Fatalf("last4 = %q, want %q", card.Last4(), "1111")
			}
		})
	}
}

func TestUnmarshalCardFromDatabase(t *testing.T) {
	now := time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		id      domain.CardID
		last4   string
		wantErr error
	}{
		{name: "valid", id: "card-1", last4: "1111"},
		{name: "missing id", id: "", last4: "1111", wantErr: domain.ErrIDIsRequired},
		{name: "bad last4", id: "card-1", last4: "11", wantErr: domain.ErrInvalidLast4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.UnmarshalCardFromDatabase(tt.id, tt.last4, 1, 2020, now)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
