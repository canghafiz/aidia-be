package repositories

import (
	"backend/models/domains"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UsersRepo interface {
	Create(db *gorm.DB, user domains.Users) error
	AssignRole(db *gorm.DB, userID uuid.UUID, roleName string) error
	ChangePassword(db *gorm.DB, user domains.Users) error
	GetUserRole(db *gorm.DB, userID uuid.UUID) (string, error)
	FindByUsernameOrEmail(db *gorm.DB, usernameOrEmail string, preloads ...string) (*domains.Users, error)
	CheckPasswordValid(db *gorm.DB, usernameOrEmail, password string) (bool, error)
	CheckSuperAdminExist(db *gorm.DB) (bool, error)
	GetByUserId(db *gorm.DB, userID uuid.UUID) (*domains.Users, error)
	UpdateTenantSchema(db *gorm.DB, user domains.Users) error
	Update(db *gorm.DB, user domains.Users) error
	GetUsers(db *gorm.DB, exceptId string, pagination domains.Pagination) ([]domains.Users, int, error)
	GetUsersRoleClient(db *gorm.DB, pagination domains.Pagination) ([]domains.Users, int, error)
	FilterUsers(db *gorm.DB, name, email, role string, pagination domains.Pagination) ([]domains.Users, int, error)
	UpdateUserStatusActive(db *gorm.DB, users domains.Users) error
	CreateApprovalLogs(db *gorm.DB, model domains.TenantApprovalLogs) error
	DeleteByUserId(db *gorm.DB, userID uuid.UUID) error
}
