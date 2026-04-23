package repositories

import (
	"backend/models/domains"

	"gorm.io/gorm"
)

type CustomerRepo interface {
	Create(db *gorm.DB, schema string, customer domains.Customer) (*domains.Customer, error)
	Update(db *gorm.DB, schema string, customer domains.Customer) (*domains.Customer, error)
	GetAll(db *gorm.DB, schema string, pagination domains.Pagination) ([]domains.Customer, int, error)
	GetByPhone(db *gorm.DB, schema string, phoneCountryCode, phoneNumber string) (*domains.Customer, error)
	GetByUsername(db *gorm.DB, schema string, username string) (*domains.Customer, error)
	GetByID(db *gorm.DB, schema string, id int) (*domains.Customer, error)
}
