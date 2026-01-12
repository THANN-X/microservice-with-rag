package domain

import "errors"

// ประกาศตัวแปร Error ไว้เทียบค่า
var ErrUserNotFound = errors.New("user not found")

var ErrIncorrectPassword = errors.New("incorrect old password")

var ErrSessionNotFound = errors.New("session not found")
