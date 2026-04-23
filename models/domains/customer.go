package domains

import "time"

type Customer struct {
	ID               int       `gorm:"column:id;primaryKey;autoIncrement"`
	Name             string    `gorm:"column:name;not null"`
	Username         *string   `gorm:"column:username"`
	PhoneCountryCode *string   `gorm:"column:phone_country_code"`
	PhoneNumber      *string   `gorm:"column:phone_number"`
	AccountType      string    `gorm:"column:account_type;not null"`
	CreatedAt        time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt        time.Time `gorm:"column:updated_at;autoUpdateTime"`
}
