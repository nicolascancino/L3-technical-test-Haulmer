package query

import (
	"context"
	"time"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/domain"
)

type Card struct {
	ID             string
	Last4          string
	ExpMonth       int
	ExpYear        int
	CardHolderName string
	Brand          string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type CardPage struct {
	Items      []Card
	NextCursor string
}

type CardFilter struct {
	Last4          string
	Brand          string
	CardHolderName string
	ExpMonth       int
	ExpYear        int
}

type CardReader interface {
	GetCard(ctx context.Context, id domain.CardID) (Card, error)
	ListCards(ctx context.Context, limit int, cursor string) (CardPage, error)
	SearchCards(ctx context.Context, filter CardFilter, limit int, cursor string) (CardPage, error)
}
