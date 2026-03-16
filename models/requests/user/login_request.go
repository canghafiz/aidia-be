package user

type LoginRequest struct {
	UsernameOrEmail string `json:"username_or_email" validate:"required,min=3,max=150"`
	Password        string `json:"password"          validate:"required,min=6,max=100"`
}
