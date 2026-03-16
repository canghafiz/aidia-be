package mocks

import (
	"backend/models/domains"
	"backend/models/requests/user"
	"backend/models/responses/pagination"
	res "backend/models/responses/user"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type UsersServMock struct {
	mock.Mock
}

func (m *UsersServMock) Login(request user.LoginRequest) (*res.LoginResponse, error) {
	args := m.Called(request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*res.LoginResponse), args.Error(1)
}

func (m *UsersServMock) Logout(accessToken string) error {
	args := m.Called(accessToken)
	return args.Error(0)
}

func (m *UsersServMock) Me(accessToken string) (*res.Response, error) {
	args := m.Called(accessToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*res.Response), args.Error(1)
}

func (m *UsersServMock) ChangePw(accessToken string, request user.ChangePwRequest) error {
	args := m.Called(accessToken, request)
	return args.Error(0)
}

func (m *UsersServMock) CheckSuperAdminExist() (bool, error) {
	args := m.Called()
	return args.Bool(0), args.Error(1)
}

func (m *UsersServMock) CreateSuperAdmin(request user.CreateSuperAdminRequest) error {
	args := m.Called(request)
	return args.Error(0)
}

func (m *UsersServMock) RefreshToken(accessToken string) (*string, error) {
	args := m.Called(accessToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*string), args.Error(1)
}

func (m *UsersServMock) CreateUser(accessToken string, request user.CreateUserRequest) error {
	args := m.Called(accessToken, request)
	return args.Error(0)
}

func (m *UsersServMock) UpdateProfileClient(accessToken string, userID uuid.UUID, request user.UpdateProfileRequest) error {
	args := m.Called(accessToken, userID, request)
	return args.Error(0)
}

func (m *UsersServMock) UpdateProfileNonClient(accessToken string, userID uuid.UUID, request user.UpdateProfileNonClientRequest) error {
	args := m.Called(accessToken, userID, request)
	return args.Error(0)
}

func (m *UsersServMock) EditUserData(accessToken string, userID uuid.UUID, request user.EditUserDataRequest) error {
	args := m.Called(accessToken, userID, request)
	return args.Error(0)
}

func (m *UsersServMock) GetByUserId(userID uuid.UUID) (*res.SingleResponse, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*res.SingleResponse), args.Error(1)
}

func (m *UsersServMock) GetUsers(pg domains.Pagination) (pagination.Response, error) {
	args := m.Called(pg)
	return args.Get(0).(pagination.Response), args.Error(1)
}

func (m *UsersServMock) FilterUsers(name, email, role string, pg domains.Pagination) (pagination.Response, error) {
	args := m.Called(name, email, role, pg)
	return args.Get(0).(pagination.Response), args.Error(1)
}

func (m *UsersServMock) UpdateUserStatusActive(accessToken string, userID uuid.UUID) error {
	args := m.Called(accessToken, userID)
	return args.Error(0)
}

func (m *UsersServMock) DeleteByUserId(accessToken string, userID uuid.UUID) error {
	args := m.Called(accessToken, userID)
	return args.Error(0)
}
