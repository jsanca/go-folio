package service

import "errors"

var (
	// Validation sentinels — returned when input fails domain rules.
	ErrInvalidProduct = errors.New("invalid product")
	ErrInvalidVariant = errors.New("invalid variant")
	ErrInvalidImage   = errors.New("invalid image")
	ErrInvalidCursor  = errors.New("invalid cursor")

	// Not-found sentinels — returned when a requested resource does not exist.
	ErrProductNotFound = errors.New("product not found")
	ErrVariantNotFound = errors.New("variant not found")

	// Conflict sentinels — returned when a uniqueness constraint would be violated.
	ErrProductConflict = errors.New("product already exists")
	ErrVariantConflict = errors.New("variant already exists")
)
