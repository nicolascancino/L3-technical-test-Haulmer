package domain

import "context"

// No Update or Get methods: a card has no state transitions and reads are
// served by app/query.
//
// AddCard returns the stored card: c itself for a new idempotency key, or the
// card the key was first used with.
type CardRepository interface {
	AddCard(ctx context.Context, idempotencyKey string, c *Card) (*Card, error)
}
