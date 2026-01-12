package domain

import "time"

type Admin struct {
	ID        uint
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
	FirstName string
	LastName  string
	Username  string
	Password  Password
	Phone     string
	Address   string
	Role      string
}
