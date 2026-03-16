package repositories

import (
	"backend/models/domains"

	"gorm.io/gorm"
)

type RoleRepo interface {
	GetRoles(db *gorm.DB) ([]domains.Roles, error)
}
