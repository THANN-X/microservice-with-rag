package errs

import "net/http"

// AppError struct to represent application-specific errors
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Implement interface error
func (e AppError) Error() string {
	return e.Message
}

// Constructor for AppError
func NewAppError(code int, message string) error {
	return AppError{
		Code:    code,
		Message: message,
	}
}

// --- Generic Errors ---
func NewNotFoundError(msg string) error {
	return NewAppError(http.StatusNotFound, msg)
}

func NewUnexpectedError() error {
	return NewAppError(http.StatusInternalServerError, "An unexpected error occurred")
}

func NewValidationError(msg string) error {
	return NewAppError(http.StatusBadRequest, msg)
}

func NewConflictError(msg string) error {
	return NewAppError(http.StatusConflict, msg)
}

func NewUnauthorizedError(msg string) error {
	return NewAppError(http.StatusUnauthorized, msg)
}

// --- Product Specific Errors ---
// Insufficient stock error
func NewInsufficientStockError(msg string) error {
	//
	return NewAppError(http.StatusUnprocessableEntity, msg)
}

// Product inactive error
func NewProductInactiveError(msg string) error {
	return NewAppError(http.StatusGone, msg) // 410 Gone
}

// Order cannot be modified error
func NewForbiddenError(msg string) error {
	return NewAppError(http.StatusForbidden, msg)
}
