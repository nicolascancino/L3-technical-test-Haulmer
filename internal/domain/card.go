package domain

import "time"

const last4Length = 4

type CardID string

func (id CardID) IsZero() bool { return id == "" }

// The full PAN is deliberately not a field: its protected storage is still undecided.
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

// UnmarshalCardFromDatabase skips the expiry rule on purpose: a card valid when
// created may legitimately be expired today.
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

// TODO: Luhn check; signature only, currently accepts any value.
func validatePAN(pan string) error {
	return nil
}

// TODO: month range and past-expiry check; signature only, currently accepts any value.
func validateExpiry(month, year int, now time.Time) error {
	return nil
}

func (c *Card) ID() CardID           { return c.id }
func (c *Card) Last4() string        { return c.last4 }
func (c *Card) ExpMonth() int        { return c.expMonth }
func (c *Card) ExpYear() int         { return c.expYear }
func (c *Card) CreatedAt() time.Time { return c.createdAt }
