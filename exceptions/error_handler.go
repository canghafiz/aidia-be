package exceptions

import (
	"backend/helpers"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func ErrorHandler(c *gin.Context, err interface{}) {
	if newError(c, err) {
		return
	}
	internalServerError(c, err)
}

func newError(c *gin.Context, err interface{}) bool {
	var errMsg string

	switch e := err.(type) {
	case error:
		errMsg = e.Error()
	case string:
		errMsg = e
	default:
		errMsg = fmt.Sprintf("%v", err)
	}

	lowered := strings.ToLower(errMsg)

	errorResponses := map[string]int{
		"user already exists":      http.StatusConflict,
		"user not found":           http.StatusUnauthorized,
		"invalid password":         http.StatusUnauthorized,
		"user role not authorized": http.StatusUnauthorized,
	}

	statusCode, exists := errorResponses[lowered]
	if !exists {
		statusCode = http.StatusBadRequest
	}

	webResponse := helpers.ApiResponse{
		Success: false,
		Code:    statusCode,
		Data:    errMsg,
	}

	_ = helpers.WriteToResponseBody(c, statusCode, webResponse)
	return true
}

// internalServerError handles all uncaught errors as HTTP 500.
func internalServerError(c *gin.Context, err interface{}) {
	webResponse := helpers.ApiResponse{
		Success: false,
		Code:    http.StatusInternalServerError,
		Data:    fmt.Sprintf("%v", err),
	}
	_ = helpers.WriteToResponseBody(c, http.StatusInternalServerError, webResponse)
}
