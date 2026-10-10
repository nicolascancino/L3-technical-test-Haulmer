package query

import (
	"context"
	"time"

	"github.com/nicolascancino/L3-technical-test-Haulmer-/internal/domain"
)

type Card struct {
	ID        string
	Last4     string
	ExpMonth  int
	ExpYear   int
	CreatedAt time.Time
}

type CardPage struct {
	Items      []Card
	NextCursor string
}

type CardReader interface {
	GetCard(ctx context.Context, id domain.CardID) (Card, error)
	ListCards(ctx context.Context, limit int, cursor string) (CardPage, error)
	SearchByLast4(ctx context.Context, last4 string) ([]Card, error)
}
