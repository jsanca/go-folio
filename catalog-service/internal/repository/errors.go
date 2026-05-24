package repository

import "errors"

var (
	ErrProductNotFound     = errors.New("product not found")
	ErrVariantNotFound     = errors.New("variant not found")
	ErrImageNotFound       = errors.New("image not found")
	ErrDuplicateSKU        = errors.New("duplicate SKU")
	ErrDuplicateSlug       = errors.New("duplicate slug")
	ErrDuplicateProductCode = errors.New("duplicate product code")
)
