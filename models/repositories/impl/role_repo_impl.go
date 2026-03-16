package impl

import (
	"backend/models/domains"

	"gorm.io/gorm"
)

type RoleRepoImpl struct {
}

func NewRoleRepoImpl() *RoleRepoImpl {
	return &RoleRepoImpl{}
}

func (repo *RoleRepoImpl) GetRoles(db *gorm.DB) ([]domains.Roles, error) {
	var roles []domains.Roles
	err := db.Where("name != ?", "SuperAdmin").Find(&roles).Error
	if err != nil {
		return nil, err
	}
	return roles, nil
}
