package service

import (
	"bff/internal/core/dto"
	"bff/internal/core/port"
	"fmt"
	"sync"
)

type ProfileService struct {
	// Service fields and methods
	userClient port.UserClientPort
}

func NewProfileService(userClient port.UserClientPort) *ProfileService {
	return &ProfileService{userClient: userClient}
}

func (s *ProfileService) GetUserProfile(id string) (*dto.UserProfileResponse, error) {
	var wg sync.WaitGroup
	var userData *dto.UserRawdata
	// Fetch user data concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()
		userData, _ = s.userClient.FetchUser(id)
	}()

	//wait for all goroutines to finish
	wg.Wait()

	// Transform UserRawdata to UserProfileResponse
	if userData == nil {
		return nil, fmt.Errorf("user not found")
	}

	// aggregate data into UserProfileResponse
	return &dto.UserProfileResponse{
		ID: userData.ID,
		// Name:  userData.Name,
		Email: userData.Email,
	}, nil
}
