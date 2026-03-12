package domain

import "errors"

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrIncorrectPassword = errors.New("incorrect old password")
	ErrSessionNotFound   = errors.New("session not found")
)
