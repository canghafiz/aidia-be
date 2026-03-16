package user

import "backend/models/domains"

type LoginResponse struct {
	AccessToken string   `json:"access_token"`
	UserData    Response `json:"user_data"`
}

func ToLoginResponse(token, role string, user domains.Users) *LoginResponse {
	return &LoginResponse{
		AccessToken: token,
		UserData:    ToResponse(user, role),
	}
}
