package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/domain"
)

func TestNewCard(t *testing.T) {
	now := time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)
	valid := domain.NewCardConfig{
		ID: "card-1", PAN: "4111111111111111", ExpMonth: 12, ExpYear: 2030, CreatedAt: now,
		CardHolderName: "  Ada Lovelace ", PANHash: "hash-1",
	}

	tests := []struct {
		name    string
		mutate  func(*domain.NewCardConfig)
		wantErr error
	}{
		{name: "valid card", mutate: func(*domain.NewCardConfig) {}},
		{name: "missing id", mutate: func(c *domain.NewCardConfig) { c.ID = "" }, wantErr: domain.ErrIDIsRequired},
		{name: "missing created at", mutate: func(c *domain.NewCardConfig) { c.CreatedAt = time.Time{} }, wantErr: domain.ErrCreatedAtIsRequired},
		{name: "pan too short", mutate: func(c *domain.NewCardConfig) { c.PAN = "123" }, wantErr: domain.ErrInvalidPAN},
		{name: "blank holder name", mutate: func(c *domain.NewCardConfig) { c.CardHolderName = "   " }, wantErr: domain.ErrCardHolderNameRequired},
		{name: "missing pan hash", mutate: func(c *domain.NewCardConfig) { c.PANHash = "" }, wantErr: domain.ErrPANHashRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := valid
			tt.mutate(&cfg)

			card, err := domain.NewCard(cfg)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if card.Last4() != "1111" {
				t.Fatalf("last4 = %q, want %q", card.Last4(), "1111")
			}
			if card.CardHolderName() != "Ada Lovelace" {
				t.Errorf("holder name = %q, want trimmed name", card.CardHolderName())
			}
			if card.PANHash() != "hash-1" {
				t.Errorf("pan hash = %q, want hash-1", card.PANHash())
			}
			if card.Brand() != domain.Unknown {
				t.Errorf("brand = %q, want %q", card.Brand(), domain.Unknown)
			}
			if !card.UpdatedAt().Equal(now) {
				t.Errorf("updatedAt = %v, want %v", card.UpdatedAt(), now)
			}
		})
	}
}

func TestUnmarshalCardFromDatabase(t *testing.T) {
	now := time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)
	later := now.Add(time.Hour)

	tests := []struct {
		name    string
		id      domain.CardID
		last4   string
		holder  string
		hash    string
		wantErr error
	}{
		{name: "valid", id: "card-1", last4: "1111", holder: "Ada", hash: "h"},
		{name: "missing id", id: "", last4: "1111", holder: "Ada", hash: "h", wantErr: domain.ErrIDIsRequired},
		{name: "bad last4", id: "card-1", last4: "11", holder: "Ada", hash: "h", wantErr: domain.ErrInvalidLast4},
		{name: "missing holder name", id: "card-1", last4: "1111", holder: "", hash: "h", wantErr: domain.ErrCardHolderNameRequired},
		{name: "missing pan hash", id: "card-1", last4: "1111", holder: "Ada", hash: "", wantErr: domain.ErrPANHashRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			card, err := domain.UnmarshalCardFromDatabase(tt.id, tt.last4, 1, 2020, tt.holder, domain.Visa, tt.hash, now, later)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if card.CardHolderName() != "Ada" || card.Brand() != domain.Visa || card.PANHash() != "h" || !card.UpdatedAt().Equal(later) {
				t.Errorf("unexpected card: %+v", card)
			}
		})
	}
}
