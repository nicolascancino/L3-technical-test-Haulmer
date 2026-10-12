package domain

import (
	"strings"
	"time"
)

const last4Length = 4

type CardID string

func (id CardID) IsZero() bool { return id == "" }

type CardBrand string

const (
	Visa       CardBrand = "visa"
	Mastercard CardBrand = "mastercard"
	Amex       CardBrand = "amex"
	Discover   CardBrand = "discover"
	Diners     CardBrand = "diners"
	JCB        CardBrand = "jcb"
	Unknown    CardBrand = "unknown"
)

// The full PAN is deliberately not a field: its protected storage is still undecided.
type Card struct {
	id             CardID
	last4          string
	expMonth       int
	expYear        int
	cardHolderName string
	brand          CardBrand
	panHash        string
	createdAt      time.Time
	updatedAt      time.Time
}

type NewCardConfig struct {
	ID             CardID
	PAN            string
	ExpMonth       int
	ExpYear        int
	CardHolderName string
	PANHash        string    // opaque: the hashing algorithm and its secret live outside the domain
	CreatedAt      time.Time // injected so tests do not depend on the wall clock
}

func NewCard(cfg NewCardConfig) (*Card, error) {
	if cfg.ID.IsZero() {
		return nil, ErrIDIsRequired
	}
	if cfg.CreatedAt.IsZero() {
		return nil, ErrCreatedAtIsRequired
	}
	holderName := strings.TrimSpace(cfg.CardHolderName)
	if holderName == "" {
		return nil, ErrCardHolderNameRequired
	}
	if cfg.PANHash == "" {
		return nil, ErrPANHashRequired
	}
	if err := validatePAN(cfg.PAN); err != nil {
		return nil, err
	}
	if err := validateExpiry(cfg.ExpMonth, cfg.ExpYear, cfg.CreatedAt); err != nil {
		return nil, err
	}

	return &Card{
		id:             cfg.ID,
		last4:          cfg.PAN[len(cfg.PAN)-last4Length:],
		expMonth:       cfg.ExpMonth,
		expYear:        cfg.ExpYear,
		cardHolderName: holderName,
		brand:          detectBrand(cfg.PAN),
		panHash:        cfg.PANHash,
		createdAt:      cfg.CreatedAt,
		updatedAt:      cfg.CreatedAt,
	}, nil
}

// UnmarshalCardFromDatabase skips the expiry rule on purpose: a card valid when
// created may legitimately be expired today.
func UnmarshalCardFromDatabase(
	id CardID,
	last4 string,
	expMonth, expYear int,
	cardHolderName string,
	brand CardBrand,
	panHash string,
	createdAt, updatedAt time.Time,
) (*Card, error) {
	if id.IsZero() {
		return nil, ErrIDIsRequired
	}
	if len(last4) != last4Length {
		return nil, ErrInvalidLast4
	}
	if cardHolderName == "" {
		return nil, ErrCardHolderNameRequired
	}
	if panHash == "" {
		return nil, ErrPANHashRequired
	}
	return &Card{
		id:             id,
		last4:          last4,
		expMonth:       expMonth,
		expYear:        expYear,
		cardHolderName: cardHolderName,
		brand:          brand,
		panHash:        panHash,
		createdAt:      createdAt,
		updatedAt:      updatedAt,
	}, nil
}

// NormalizePAN must run before hashing, so the same card always yields the same hash.
func NormalizePAN(raw string) string {
	return strings.NewReplacer(" ", "", "-", "").Replace(raw)
}

func validatePAN(pan string) error {
	if len(pan) < 12 || len(pan) > 19 {
		return ErrInvalidPAN
	}
	sum := 0
	double := false
	for i := len(pan) - 1; i >= 0; i-- {
		if pan[i] < '0' || pan[i] > '9' {
			return ErrInvalidPAN
		}
		digit := int(pan[i] - '0')
		if double {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
		double = !double
	}
	if sum%10 != 0 {
		return ErrInvalidPAN
	}
	return nil
}

// A card stays valid through the last day of its expiry month.
func validateExpiry(month, year int, now time.Time) error {
	if month < 1 || month > 12 || year < 1000 || year > 9999 {
		return ErrInvalidExpiry
	}
	now = now.UTC()
	if year < now.Year() || (year == now.Year() && month < int(now.Month())) {
		return ErrCardExpired
	}
	return nil
}

// TODO: prefix-based detection; signature only, currently returns Unknown.
func detectBrand(pan string) CardBrand {
	return Unknown
}

func (c *Card) ID() CardID             { return c.id }
func (c *Card) Last4() string          { return c.last4 }
func (c *Card) ExpMonth() int          { return c.expMonth }
func (c *Card) ExpYear() int           { return c.expYear }
func (c *Card) CardHolderName() string { return c.cardHolderName }
func (c *Card) Brand() CardBrand       { return c.brand }
func (c *Card) PANHash() string        { return c.panHash }
func (c *Card) CreatedAt() time.Time   { return c.createdAt }
func (c *Card) UpdatedAt() time.Time   { return c.updatedAt }
