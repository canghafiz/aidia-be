package impl

import (
	"backend/models/domains"

	"gorm.io/gorm"
)

type OrderPaymentRepoImpl struct{}

func NewOrderPaymentRepoImpl() *OrderPaymentRepoImpl {
	return &OrderPaymentRepoImpl{}
}

func (repo *OrderPaymentRepoImpl) Create(db *gorm.DB, schema string, payment domains.OrderPayment) (*domains.OrderPayment, error) {
	if err := db.Table(schema + ".order_payments").Create(&payment).Error; err != nil {
		return nil, err
	}
	return &payment, nil
}

func (repo *OrderPaymentRepoImpl) GetAll(db *gorm.DB, schema string, pagination domains.Pagination) ([]domains.OrderPayment, int, error) {
	var payments []domains.OrderPayment
	var total int64

	if err := db.Table(schema + ".order_payments").Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := db.Raw(`
		SELECT * FROM `+schema+`.order_payments
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`, pagination.Limit, pagination.Offset()).
		Scan(&payments).Error; err != nil {
		return nil, 0, err
	}

	return payments, int(total), nil
}

func (repo *OrderPaymentRepoImpl) GetByOrderID(db *gorm.DB, schema string, orderID int) (*domains.OrderPayment, error) {
	var payment domains.OrderPayment
	if err := db.Raw(`SELECT * FROM `+schema+`.order_payments WHERE order_id = ?`, orderID).
		Scan(&payment).Error; err != nil {
		return nil, err
	}
	return &payment, nil
}

func (repo *OrderPaymentRepoImpl) UpdateStatus(db *gorm.DB, schema string, orderID int, status domains.PaymentStatus) error {
	return db.Table(schema+".order_payments").
		Where("order_id = ?", orderID).
		Update("payment_status", status).Error
}
