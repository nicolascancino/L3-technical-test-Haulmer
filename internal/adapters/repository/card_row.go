package repository

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/adapters/repository/tables"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/domain"
)

type cardRow tables.CardRow

func toRow(c *domain.Card) (cardRow, error) {
	id, err := parseCardID(c.ID())
	if err != nil {
		return cardRow{}, err
	}
	return cardRow{
		ID:             id.String(),
		CardHolderName: c.CardHolderName(),
		Brand:          string(c.Brand()),
		PANHash:        c.PANHash(),
		Last4:          c.Last4(),
		ExpMonth:       c.ExpMonth(),
		ExpYear:        c.ExpYear(),
		CreatedAt:      c.CreatedAt(),
		UpdatedAt:      c.UpdatedAt(),
	}, nil
}

func (r cardRow) toDomain() (*domain.Card, error) {
	return domain.UnmarshalCardFromDatabase(
		domain.CardID(r.ID),
		r.Last4,
		r.ExpMonth, r.ExpYear,
		r.CardHolderName,
		domain.CardBrand(r.Brand),
		r.PANHash,
		r.CreatedAt, r.UpdatedAt,
	)
}

// A malformed id cannot exist in a uuid column, so it is reported as not found (plan D8).
func parseCardID(id domain.CardID) (uuid.UUID, error) {
	parsed, err := uuid.Parse(string(id))
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: %q", domain.ErrCardNotFound, id)
	}
	return parsed, nil
}
