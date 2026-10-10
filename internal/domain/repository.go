package domain

import "context"

// CardRepository is the write-side contract for cards. A card has no state
// transitions, so there is no Update method. Listing and searching by last4
// are read models and live next to the query handlers in app/query.
type CardRepository interface {
	// AddCard persists a new card. It returns ErrCardAlreadyExists when the id is taken.
	AddCard(ctx context.Context, c *Card) error
	// GetCard returns ErrCardNotFound when the id does not exist.
	GetCard(ctx context.Context, id CardID) (*Card, error)
}
