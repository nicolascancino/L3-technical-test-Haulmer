package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/domain"
)

func TestNormalizePAN(t *testing.T) {
	tests := map[string]string{
		"4111111111111111":        "4111111111111111",
		"4111 1111 1111 1111":     "4111111111111111",
		"4111-1111-1111-1111":     "4111111111111111",
		" 4111 - 1111 1111-1111 ": "4111111111111111",
		"41111111111a1111":        "41111111111a1111",
	}
	for raw, want := range tests {
		if got := domain.NormalizePAN(raw); got != want {
			t.Errorf("NormalizePAN(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestNewCard_AcceptsPANLengthBounds(t *testing.T) {
	now := time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)
	for pan, last4 := range map[string]string{"400000000002": "0002", "4030000000000000003": "0003"} {
		card, err := domain.NewCard(domain.NewCardConfig{
			ID: "card-1", PAN: pan, ExpMonth: 12, ExpYear: 2030, CreatedAt: now,
			CardHolderName: "Ada Lovelace", PANHash: "hash-1",
		})
		if err != nil {
			t.Fatalf("%d-digit pan: unexpected error: %v", len(pan), err)
		}
		if card.Last4() != last4 {
			t.Errorf("%d-digit pan: last4 = %q, want %q", len(pan), card.Last4(), last4)
		}
	}
}

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
		{name: "month zero", mutate: func(c *domain.NewCardConfig) { c.ExpMonth = 0 }, wantErr: domain.ErrInvalidExpiry},
		{name: "month thirteen", mutate: func(c *domain.NewCardConfig) { c.ExpMonth = 13 }, wantErr: domain.ErrInvalidExpiry},
		{name: "two digit year", mutate: func(c *domain.NewCardConfig) { c.ExpYear = 30 }, wantErr: domain.ErrInvalidExpiry},
		{name: "five digit year", mutate: func(c *domain.NewCardConfig) { c.ExpYear = 10000 }, wantErr: domain.ErrInvalidExpiry},
		{name: "expires this month", mutate: func(c *domain.NewCardConfig) { c.ExpMonth, c.ExpYear = 10, 2026 }},
		{name: "expired last month", mutate: func(c *domain.NewCardConfig) { c.ExpMonth, c.ExpYear = 9, 2026 }, wantErr: domain.ErrCardExpired},
		{name: "expired last year", mutate: func(c *domain.NewCardConfig) { c.ExpMonth, c.ExpYear = 12, 2025 }, wantErr: domain.ErrCardExpired},
		{name: "earlier month next year", mutate: func(c *domain.NewCardConfig) { c.ExpMonth, c.ExpYear = 1, 2027 }},
		{name: "pan fails luhn", mutate: func(c *domain.NewCardConfig) { c.PAN = "4111111111111112" }, wantErr: domain.ErrInvalidPAN},
		{name: "pan with letters", mutate: func(c *domain.NewCardConfig) { c.PAN = "41111111111a1111" }, wantErr: domain.ErrInvalidPAN},
		{name: "pan with spaces is not normalized here", mutate: func(c *domain.NewCardConfig) { c.PAN = "4111 1111 1111 1111" }, wantErr: domain.ErrInvalidPAN},
		{name: "pan 11 digits", mutate: func(c *domain.NewCardConfig) { c.PAN = "40000000006" }, wantErr: domain.ErrInvalidPAN},
		{name: "pan 20 digits", mutate: func(c *domain.NewCardConfig) { c.PAN = "40300000000000000003" }, wantErr: domain.ErrInvalidPAN},
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
			if card.Brand() != domain.Visa {
				t.Errorf("brand = %q, want %q", card.Brand(), domain.Visa)
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

func TestNewCard_DetectsBrand(t *testing.T) {
	now := time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)
	tests := map[string]domain.CardBrand{
		"4111111111111111":    domain.Visa,
		"4222222222222":       domain.Visa,
		"5555555555554444":    domain.Mastercard,
		"5105105105105100":    domain.Mastercard,
		"2223003122003222":    domain.Mastercard,
		"378282246310005":     domain.Amex,
		"371449635398431":     domain.Amex,
		"6011111111111117":    domain.Unknown,
		"400000000002":        domain.Unknown,
		"36227206271667":      domain.Unknown,
		"6200000000000005":    domain.Unknown,
		"3530111333300000":    domain.Unknown,
		"4030000000000000003": domain.Visa,
	}
	for pan, want := range tests {
		card, err := domain.NewCard(domain.NewCardConfig{
			ID: "card-1", PAN: pan, CardHolderName: "Ada", PANHash: "h", ExpMonth: 12, ExpYear: 2030, CreatedAt: now,
		})
		if err != nil {
			t.Errorf("NewCard(%q): %v", pan, err)
			continue
		}
		if card.Brand() != want {
			t.Errorf("brand of %q = %q, want %q", pan, card.Brand(), want)
		}
	}
}
