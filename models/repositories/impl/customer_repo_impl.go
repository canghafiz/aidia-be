package impl

import (
	"backend/models/domains"

	"gorm.io/gorm"
)

type CustomerRepoImpl struct{}

func NewCustomerRepoImpl() *CustomerRepoImpl {
	return &CustomerRepoImpl{}
}

func (repo *CustomerRepoImpl) Create(db *gorm.DB, schema string, customer domains.Customer) (*domains.Customer, error) {
	if err := db.Table(schema + ".customer").Create(&customer).Error; err != nil {
		return nil, err
	}
	return &customer, nil
}

func (repo *CustomerRepoImpl) Update(db *gorm.DB, schema string, customer domains.Customer) (*domains.Customer, error) {
	if err := db.Table(schema + ".customer").Save(&customer).Error; err != nil {
		return nil, err
	}
	return &customer, nil
}

func (repo *CustomerRepoImpl) GetAll(db *gorm.DB, schema string, pagination domains.Pagination) ([]domains.Customer, int, error) {
	var customers []domains.Customer
	var total int64

	if err := db.Table(schema + ".customer").Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := db.Raw(`
		SELECT * FROM `+schema+`.customer
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`, pagination.Limit, pagination.Offset()).
		Scan(&customers).Error; err != nil {
		return nil, 0, err
	}

	return customers, int(total), nil
}

func (repo *CustomerRepoImpl) GetByPhone(db *gorm.DB, schema string, phoneCountryCode, phoneNumber string) (*domains.Customer, error) {
	var customer domains.Customer
	if err := db.Table(schema+".customer").
		Where("phone_country_code = ? AND phone_number = ?", phoneCountryCode, phoneNumber).
		First(&customer).Error; err != nil {
		return nil, err
	}
	return &customer, nil
}

func (repo *CustomerRepoImpl) GetByUsername(db *gorm.DB, schema string, username string) (*domains.Customer, error) {
	var customer domains.Customer
	if err := db.Table(schema+".customer").
		Where("username = ?", username).
		First(&customer).Error; err != nil {
		return nil, err
	}
	return &customer, nil
}

func (repo *CustomerRepoImpl) GetByID(db *gorm.DB, schema string, id int) (*domains.Customer, error) {
	var customer domains.Customer
	if err := db.Table(schema+".customer").
		Where("id = ?", id).
		First(&customer).Error; err != nil {
		return nil, err
	}
	return &customer, nil
}
