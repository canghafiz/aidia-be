package domains

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

type Pagination struct {
	Page, Limit int
}

func (p *Pagination) Offset() int {
	return (p.Page - 1) * p.Limit
}

func ParsePagination(context *gin.Context) Pagination {
	page, _ := strconv.Atoi(context.Query("page"))
	limit, _ := strconv.Atoi(context.Query("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	return Pagination{Page: page, Limit: limit}
}
