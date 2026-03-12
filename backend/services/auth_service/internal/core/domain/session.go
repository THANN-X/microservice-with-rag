package domain

import "time"

type Session struct {
	ID           uint
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
	UserID       *uint
	User         *User
	AdminID      *uint
	Admin        *Admin
	RefreshToken string
	DeviceInfo   string
	IPAddress    string
	ExpiredAt    time.Time
	IsRevoked    bool
}
