package domain

import "time"

const last4Length = 4

type CardID string

func (id CardID) IsZero() bool { return id == "" }

// Card is a credit card. The full PAN is deliberately not a field: how it is
// protected and stored is still undecided, so the entity only keeps what the
// API needs to show and search by (last4) plus the expiry.
type Card struct {
	id        CardID
	last4     string
	expMonth  int
	expYear   int
	createdAt time.Time
}

type NewCardConfig struct {
	ID        CardID
	PAN       string
	ExpMonth  int
	ExpYear   int
	CreatedAt time.Time // injected so tests do not depend on the wall clock
}

// NewCard creates a brand-new card. It enforces the business rules: the PAN
// must pass the Luhn check and the expiry must not be in the past.
func NewCard(cfg NewCardConfig) (*Card, error) {
	if cfg.ID.IsZero() {
		return nil, ErrIDIsRequired
	}
	if cfg.CreatedAt.IsZero() {
		return nil, ErrCreatedAtIsRequired
	}
	if err := validatePAN(cfg.PAN); err != nil {
		return nil, err
	}
	// Guards the slice below while validatePAN is still a stub.
	if len(cfg.PAN) < last4Length {
		return nil, ErrInvalidPAN
	}
	if err := validateExpiry(cfg.ExpMonth, cfg.ExpYear, cfg.CreatedAt); err != nil {
		return nil, err
	}

	return &Card{
		id:        cfg.ID,
		last4:     cfg.PAN[len(cfg.PAN)-last4Length:],
		expMonth:  cfg.ExpMonth,
		expYear:   cfg.ExpYear,
		createdAt: cfg.CreatedAt,
	}, nil
}

// UnmarshalCardFromDatabase rebuilds a persisted card. It validates the shape
// of the stored data but does not re-run the expiry rule: a card that was valid
// when created may legitimately be expired today. Only adapters should call it.
func UnmarshalCardFromDatabase(
	id CardID,
	last4 string,
	expMonth, expYear int,
	createdAt time.Time,
) (*Card, error) {
	if id.IsZero() {
		return nil, ErrIDIsRequired
	}
	if len(last4) != last4Length {
		return nil, ErrInvalidLast4
	}
	return &Card{
		id:        id,
		last4:     last4,
		expMonth:  expMonth,
		expYear:   expYear,
		createdAt: createdAt,
	}, nil
}

// validatePAN checks that pan is a well-formed card number using the Luhn
// algorithm. It returns ErrInvalidPAN when it is not.
//
// TODO: not implemented yet (signature only); it currently accepts any value.
func validatePAN(pan string) error {
	return nil
}

// validateExpiry checks that month is in 1..12 and that the card has not
// expired relative to now. It returns ErrInvalidExpiry for a malformed month
// or year and ErrCardExpired for a past expiry.
//
// TODO: not implemented yet (signature only); it currently accepts any value.
func validateExpiry(month, year int, now time.Time) error {
	return nil
}

func (c *Card) ID() CardID           { return c.id }
func (c *Card) Last4() string        { return c.last4 }
func (c *Card) ExpMonth() int        { return c.expMonth }
func (c *Card) ExpYear() int         { return c.expYear }
func (c *Card) CreatedAt() time.Time { return c.createdAt }
