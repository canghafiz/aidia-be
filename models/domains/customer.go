package domains

import "time"

type Customer struct {
	ID               int       `gorm:"column:id;primaryKey;autoIncrement"`
	Name             string    `gorm:"column:name;not null"`
	PhoneCountryCode string    `gorm:"column:phone_country_code;not null"`
	PhoneNumber      string    `gorm:"column:phone_number;not null"`
	AccountType      string    `gorm:"column:account_type;not null"`
	CreatedAt        time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt        time.Time `gorm:"column:updated_at;autoUpdateTime"`
}
