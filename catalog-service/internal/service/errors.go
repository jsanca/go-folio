package service

import "errors"

var (
	ErrInvalidProduct = errors.New("invalid product")
	ErrInvalidVariant = errors.New("invalid variant")
	ErrInvalidImage   = errors.New("invalid image")
	ErrInvalidCursor  = errors.New("invalid cursor")
)
