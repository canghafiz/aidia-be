package services

import (
	"backend/models/domains"
	"backend/models/requests/user"
	"backend/models/responses/pagination"

	"github.com/google/uuid"
)
import res "backend/models/responses/user"

type UsersServ interface {
	Login(request user.LoginRequest) (*res.LoginResponse, error)
	ChangePw(accessToken string, request user.ChangePwRequest) error
	Me(accessToken string) (*res.Response, error)
	CheckSuperAdminExist() (bool, error)
	CreateSuperAdmin(request user.CreateSuperAdminRequest) error
	CreateUser(accessToken string, request user.CreateUserRequest) error
	UpdateProfileClient(accessToken string, userID uuid.UUID, request user.UpdateProfileRequest) error
	UpdateProfileNonClient(accessToken string, userID uuid.UUID, request user.UpdateProfileNonClientRequest) error
	EditUserData(accessToken string, userID uuid.UUID, request user.EditUserDataRequest) error
	GetByUserId(userID uuid.UUID) (*res.SingleResponse, error)
	GetUsers(accessToken string, pagination domains.Pagination) (pagination.Response, error)
	GetClients(pagination domains.Pagination) (pagination.Response, error)
	FilterUsers(name, email, role string, pagination domains.Pagination) (pagination.Response, error)
	DeleteByUserId(accessToken string, userID uuid.UUID) error
}
