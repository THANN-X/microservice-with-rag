package errs

import "net/http"

// AppError struct ที่เก็บ Code และ Message
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Implement interface error
func (e AppError) Error() string {
	return e.Message
}

// Helper function เพื่อสร้าง AppError
func NewAppError(code int, message string) error {
	return AppError{
		Code:    code,
		Message: message,
	}
}

// --- Domain/Business Errors (ตั้งชื่อตาม Business Case) ---

func NewNotFoundError(msg string) error {
	return NewAppError(http.StatusNotFound, msg)
}

func NewUnexpectedError() error {
	return NewAppError(http.StatusInternalServerError, "An unexpected error occurred")
}

func NewValidationError(msg string) error {
	return NewAppError(http.StatusBadRequest, msg)
}

func NewConflictError(msg string) error { // เช่น อีเมลซ้ำ
	return NewAppError(http.StatusConflict, msg)
}

func NewUnauthorizedError(msg string) error { // เช่น รหัสผ่านผิด
	return NewAppError(http.StatusUnauthorized, msg)
}
