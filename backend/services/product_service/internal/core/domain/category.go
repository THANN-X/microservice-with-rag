package domain

import "time"

type CateGory struct {
	ID          uint
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Name        string
	Slug        string
	Description string
	IsActive    bool
}
