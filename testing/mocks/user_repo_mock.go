package mocks

import (
	"backend/models/domains"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type UsersRepoMock struct {
	mock.Mock
}

func (m *UsersRepoMock) Create(db *gorm.DB, user domains.Users) error {
	args := m.Called(db, user)
	return args.Error(0)
}

func (m *UsersRepoMock) AssignRole(db *gorm.DB, userID uuid.UUID, roleName string) error {
	args := m.Called(db, userID, roleName)
	return args.Error(0)
}

func (m *UsersRepoMock) GenerateToken(db *gorm.DB, userID uuid.UUID, duration time.Duration) (string, error) {
	args := m.Called(db, userID, duration)
	return args.String(0), args.Error(1)
}

func (m *UsersRepoMock) RefreshToken(db *gorm.DB, refreshToken string, duration time.Duration) (*domains.Users, error) {
	args := m.Called(db, refreshToken, duration)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domains.Users), args.Error(1)
}

func (m *UsersRepoMock) CheckTokenValid(db *gorm.DB, user domains.Users) bool {
	args := m.Called(db, user)
	return args.Bool(0)
}

func (m *UsersRepoMock) ChangePassword(db *gorm.DB, user domains.Users) error {
	args := m.Called(db, user)
	return args.Error(0)
}

func (m *UsersRepoMock) GetUserRole(db *gorm.DB, userID uuid.UUID) (string, error) {
	args := m.Called(db, userID)
	return args.String(0), args.Error(1)
}

func (m *UsersRepoMock) FindByToken(db *gorm.DB, token string) (*domains.Users, error) {
	args := m.Called(db, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domains.Users), args.Error(1)
}

func (m *UsersRepoMock) FindByUsernameOrEmail(db *gorm.DB, usernameOrEmail string, preloads ...string) (*domains.Users, error) {
	args := m.Called(db, usernameOrEmail)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domains.Users), args.Error(1)
}

func (m *UsersRepoMock) CheckPasswordValid(db *gorm.DB, usernameOrEmail, password string) (bool, error) {
	args := m.Called(db, usernameOrEmail, password)
	return args.Bool(0), args.Error(1)
}

func (m *UsersRepoMock) CheckSuperAdminExist(db *gorm.DB) (bool, error) {
	args := m.Called(db)
	return args.Bool(0), args.Error(1)
}

func (m *UsersRepoMock) ResetToken(db *gorm.DB, user domains.Users) error {
	args := m.Called(db, user)
	return args.Error(0)
}

func (m *UsersRepoMock) GetByUserId(db *gorm.DB, userID uuid.UUID) (*domains.Users, error) {
	args := m.Called(db, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domains.Users), args.Error(1)
}

func (m *UsersRepoMock) UpdateTenantSchema(db *gorm.DB, user domains.Users) error {
	args := m.Called(db, user)
	return args.Error(0)
}

func (m *UsersRepoMock) Update(db *gorm.DB, user domains.Users) error {
	args := m.Called(db, user)
	return args.Error(0)
}

func (m *UsersRepoMock) GetUsers(db *gorm.DB, pagination domains.Pagination) ([]domains.Users, int, error) {
	args := m.Called(db, pagination)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]domains.Users), args.Int(1), args.Error(2)
}

func (m *UsersRepoMock) FilterUsers(db *gorm.DB, name, email, role string, pagination domains.Pagination) ([]domains.Users, int, error) {
	args := m.Called(db, name, email, role, pagination)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]domains.Users), args.Int(1), args.Error(2)
}

func (m *UsersRepoMock) UpdateUserStatusActive(db *gorm.DB, users domains.Users) error {
	args := m.Called(db, users)
	return args.Error(0)
}

func (m *UsersRepoMock) DeleteByUserId(db *gorm.DB, userID uuid.UUID) error {
	args := m.Called(db, userID)
	return args.Error(0)
}
