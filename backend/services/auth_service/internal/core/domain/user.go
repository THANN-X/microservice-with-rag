package domain

import (
	"time"
)

// User represents a user entity in the system.
type User struct {
	ID        uint
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
	FirstName string
	LastName  string
	Email     string
	Password  Password
	Phone     string
	Address   string
	Role      string
}

func (u *User) SetPassword(rawPassword string) error {

	err := u.Password.GeneratePassword(rawPassword)

	if err != nil {
		return err
	}

	return nil
}

func (u *User) CheckPassword(rawPassword string) error {

	err := u.Password.ComparePassword(rawPassword)

	if err != nil {
		return err
	}

	return nil
}

func (u *User) ChangePassword(oldPassword, newPassword string) error {

	err := u.Password.ComparePassword(oldPassword)

	if err != nil {
		return err
	}

	err = u.Password.GeneratePassword(newPassword)

	if err != nil {
		return err
	}

	return nil
}

func (u *User) UpdateUserProfile(req *User) *User {
	if req.FirstName != "" {
		u.FirstName = req.FirstName
	}

	if req.LastName != "" {
		u.LastName = req.LastName
	}

	if req.Address != "" {
		u.Address = req.Address
	}

	if req.Phone != "" {
		u.Phone = req.Phone
	}

	return u
}
