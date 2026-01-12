package entity

import "gorm.io/gorm"

type AdminEntity struct {
	gorm.Model
	FirstName string `gorm:"not null;"`
	LastName  string `gorm:"not null;"`
	Username  string `gorm:"unique:not null;"`
	Password  string `gorm:"not null;"`
	Phone     string
	Address   string
	Role      string `gorm:"default:admin;"`
}
