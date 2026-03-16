package pagination

type Response struct {
	Data       interface{} `json:"data"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	Limit      int         `json:"limit"`
	TotalPages int         `json:"total_pages"`
}

func ToResponse(data interface{}, total, page, limit int) Response {
	totalPages := total / limit
	if total%limit != 0 {
		totalPages++
	}

	return Response{
		Data:       data,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}
}
