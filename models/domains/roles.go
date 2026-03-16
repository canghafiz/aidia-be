package domains

import (
	"time"

	"github.com/google/uuid"
)

type Roles struct {
	ID          uuid.UUID `gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	Name        string    `gorm:"column:name;not null;uniqueIndex:uq_roles_name"`
	Description string    `gorm:"column:description"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relations
	UserRoles []UserRoles `gorm:"foreignKey:RoleID;references:ID"`
}

func (Roles) TableName() string {
	return "roles"
}
