package domain

import "errors"

var (
	ErrIDIsRequired        = errors.New("card id is required")
	ErrCreatedAtIsRequired = errors.New("card creation time is required")
	ErrInvalidPAN          = errors.New("invalid card number")
	ErrInvalidExpiry       = errors.New("invalid card expiry")
	ErrCardExpired         = errors.New("card is expired")
	ErrInvalidLast4        = errors.New("last4 must be exactly 4 digits")
	ErrCardNotFound        = errors.New("card not found")
	ErrCardAlreadyExists   = errors.New("card already exists")
	ErrInvalidCursor       = errors.New("invalid cursor")
)
