package repository

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/adapters/repository/tables"
	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/domain"
)

func toRow(c *domain.Card) (tables.CardRow, error) {
	id, err := parseCardID(c.ID())
	if err != nil {
		return tables.CardRow{}, err
	}
	return tables.CardRow{
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

// A malformed id cannot exist in a uuid column, so it is reported as not found (plan D8).
func parseCardID(id domain.CardID) (uuid.UUID, error) {
	parsed, err := uuid.Parse(string(id))
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: %q", domain.ErrCardNotFound, id)
	}
	return parsed, nil
}
