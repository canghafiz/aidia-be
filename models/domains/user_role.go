package domains

import "github.com/google/uuid"

type UserRoles struct {
	UserID uuid.UUID `gorm:"column:user_id;primaryKey;type:uuid"`
	RoleID uuid.UUID `gorm:"column:role_id;primaryKey;type:uuid"`

	// Relations
	User Users `gorm:"foreignKey:UserID;references:UserID"`
	Role Roles `gorm:"foreignKey:RoleID;references:ID"`
}

func (UserRoles) TableName() string {
	return "user_roles"
}
