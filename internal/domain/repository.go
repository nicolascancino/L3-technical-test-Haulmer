package domain

import "context"

// No Update method: a card has no state transitions. Listing and last4 search
// are read models and live in app/query.
type CardRepository interface {
	AddCard(ctx context.Context, c *Card) error
	GetCard(ctx context.Context, id CardID) (*Card, error)
}
