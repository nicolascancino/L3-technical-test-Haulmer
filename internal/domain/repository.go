package domain

import "context"

// No Update or Get methods: a card has no state transitions and reads are
// served by app/query.
type CardRepository interface {
	AddCard(ctx context.Context, c *Card) error
}
