package repository

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/adapters/repository/tables"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/app/query"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/domain"
)

func toTable(c *domain.Card) (tables.Card, error) {
	id, err := parseCardID(c.ID())
	if err != nil {
		return tables.Card{}, err
	}
	return tables.Card{
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

func toQueryCard(r tables.Card) query.Card {
	return query.Card{
		ID:             r.ID,
		Last4:          r.Last4,
		ExpMonth:       r.ExpMonth,
		ExpYear:        r.ExpYear,
		CardHolderName: r.CardHolderName,
		Brand:          r.Brand,
		CreatedAt:      r.CreatedAt.UTC(),
		UpdatedAt:      r.UpdatedAt.UTC(),
	}
}

// A malformed id cannot exist in a uuid column, so it is reported as not found (plan D8).
func parseCardID(id domain.CardID) (uuid.UUID, error) {
	parsed, err := uuid.Parse(string(id))
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: %q", domain.ErrCardNotFound, id)
	}
	return parsed, nil
}
