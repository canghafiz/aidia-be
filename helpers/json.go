package helpers

import (
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
)

func ReadFromRequestBody(context *gin.Context, result interface{}) error {
	err := context.ShouldBindJSON(result)
	if err != nil {
		return fmt.Errorf("ReadFromRequestBody: %v", err)
	}

	return nil
}

func WriteToResponseBody(context *gin.Context, statusCode int, result interface{}) error {
	_, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("WriteToResponseBody: %v", err)
	}

	context.JSON(statusCode, result)
	return nil
}
