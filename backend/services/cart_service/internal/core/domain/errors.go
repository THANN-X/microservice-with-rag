package domain

import "errors"

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrInvalidInput   = errors.New("invalid input")
	ErrItemNotFound   = errors.New("cart item not found")
	ErrInternal       = errors.New("internal server error")
)
