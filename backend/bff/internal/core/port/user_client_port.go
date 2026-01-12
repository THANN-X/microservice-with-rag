package port

import (
	"bff/internal/core/dto"
)

type UserClientPort interface {
	// Define methods for user client interactions
	FetchUser(id string) (*dto.UserRawdata, error)
}
