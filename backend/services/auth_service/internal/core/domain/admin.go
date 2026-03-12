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

func (a *Admin) SetPassword(rawPassword string) error {

	err := a.Password.GeneratePassword(rawPassword)

	if err != nil {
		return err
	}

	return nil
}

func (a *Admin) CheckPassword(rawPassword string) error {
	err := a.Password.ComparePassword(rawPassword)

	if err != nil {
		return err
	}

	return nil
}
