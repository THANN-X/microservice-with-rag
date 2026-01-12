package client

import (
	"bff/internal/core/dto"
	"bff/internal/core/port"
	"encoding/json"
	"fmt"
	"net/http"
)

type UserHTTPClient struct {
	// Define fields for HTTP client, e.g., base URL, HTTP client instance
	baseURL string
	client  *http.Client
}

func NewUserHTTPClient(baseURL string) port.UserClientPort {
	return &UserHTTPClient{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

func (c *UserHTTPClient) FetchUser(id string) (*dto.UserRawdata, error) {
	// Implement the logic to make an HTTP request to fetch user data
	resp, err := c.client.Get("http://" + c.baseURL + "/users/" + id)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse the response and return the user data
	// This is a placeholder; actual implementation would involve decoding JSON response
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("service error: %d", resp.StatusCode)
	}

	var data dto.UserRawdata
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}
