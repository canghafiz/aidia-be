package impl

import (
	"backend/models/repositories"
	"backend/models/responses/role"
	"fmt"
	"log"

	"gorm.io/gorm"
)

type RoleServImpl struct {
	Db       *gorm.DB
	RoleRepo repositories.RoleRepo
}

func NewRoleServImpl(db *gorm.DB, roleRepo repositories.RoleRepo) *RoleServImpl {
	return &RoleServImpl{Db: db, RoleRepo: roleRepo}
}

func (serv *RoleServImpl) GetRoles() ([]role.Response, error) {
	// Call repo
	result, err := serv.RoleRepo.GetRoles(serv.Db)
	if err != nil {
		log.Printf("[RoleRepo.GetRoles] error: %v", err)
		return nil, fmt.Errorf("failed to get roles, please try again later")
	}

	return role.ToResponses(result), nil
}
