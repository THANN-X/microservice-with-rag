package service

import "time"

type CategoryRes struct {
	ID          uint          `json:"id"`
	Name        string        `json:"name"`
	Slug        string        `json:"slug"`
	Description string        `json:"description"`
	IsActive    bool          `json:"is_active"`
	ParentID    *uint         `json:"parent_id,omitempty"`
	Children    []CategoryRes `json:"children,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}
