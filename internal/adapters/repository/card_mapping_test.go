package repository

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/adapters/repository/tables"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/domain"
)

func TestToRow(t *testing.T) {
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	card, err := domain.NewCard(domain.NewCardConfig{
		ID:             domain.CardID(uuid.Must(uuid.NewV7()).String()),
		PAN:            "4111111111111111",
		ExpMonth:       12,
		ExpYear:        2030,
		CardHolderName: "Ada Lovelace",
		PANHash:        "hash",
		CreatedAt:      now,
	})
	if err != nil {
		t.Fatal(err)
	}

	row, err := toRow(card)
	if err != nil {
		t.Fatalf("toRow: %v", err)
	}

	want := tables.CardRow{
		ID:             string(card.ID()),
		CardHolderName: "Ada Lovelace",
		Brand:          string(card.Brand()),
		PANHash:        "hash",
		Last4:          "1111",
		ExpMonth:       12,
		ExpYear:        2030,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if row != want {
		t.Fatalf("got %+v, want %+v", row, want)
	}
}

func TestParseCardIDMapsNonUUIDToNotFound(t *testing.T) {
	_, err := parseCardID("not-a-uuid")

	if !errors.Is(err, domain.ErrCardNotFound) {
		t.Fatalf("got %v, want ErrCardNotFound", err)
	}
}

func TestToQueryCardReturnsUTC(t *testing.T) {
	santiago := time.FixedZone("UTC-3", -3*60*60)
	local := time.Date(2026, 10, 10, 9, 0, 0, 0, santiago)

	got := toQueryCard(tables.CardRow{CreatedAt: local, UpdatedAt: local})

	for name, ts := range map[string]time.Time{"created": got.CreatedAt, "updated": got.UpdatedAt} {
		if ts.Location() != time.UTC || !ts.Equal(local) {
			t.Errorf("%s = %v, want the same instant in UTC", name, ts)
		}
	}
}
