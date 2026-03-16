package services

import "backend/models/responses/role"

type RoleServ interface {
	GetRoles() ([]role.Response, error)
}
