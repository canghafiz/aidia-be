package role

import (
	"backend/models/domains"

	"github.com/google/uuid"
)

type Response struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func ToResponse(role domains.Roles) Response {
	var id uuid.UUID
	id = role.ID

	return Response{
		Id:          id.String(),
		Name:        role.Name,
		Description: role.Description,
	}
}

func ToResponses(roles []domains.Roles) []Response {
	var responses []Response
	for _, role := range roles {
		responses = append(responses, ToResponse(role))
	}
	return responses
}
