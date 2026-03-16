package helpers

type ApiResponse struct {
	Success bool        `json:"success"`
	Code    int         `json:"code"`
	Data    interface{} `json:"data,omitempty"`
}
