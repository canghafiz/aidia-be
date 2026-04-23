package helpers

import (
	"backend/hub"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const TokenDuration = 24 * time.Hour

func GetJwtToken(context *gin.Context) string {
	header := context.GetHeader("Authorization")
	jwt := strings.TrimPrefix(header, "Bearer ")

	return jwt
}

func GetUserRoleFromToken(accessToken, jwtKey string, allowedRoles []string) (*string, bool, error) {
	result, errDecode := DecodeJWT(accessToken, jwtKey)
	if errDecode != nil {
		return nil, false, fmt.Errorf("invalid token format")
	}

	role, ok := result["role"].(string)
	if !ok || role == "" {
		return nil, false, fmt.Errorf("invalid token claims")
	}

	for _, allowed := range allowedRoles {
		if role == allowed {
			return &role, true, nil
		}
	}

	return nil, false, fmt.Errorf("user role not authorized")
}

func GetUserIdFromToken(accessToken, jwtKey string) (*string, error) {
	result, errDecode := DecodeJWT(accessToken, jwtKey)
	if errDecode != nil {
		return nil, fmt.Errorf("invalid token format")
	}

	userId, ok := result["user_id"].(string)
	if !ok || userId == "" {
		return nil, fmt.Errorf("invalid token claims")
	}

	return &userId, nil
}

func GetUsernameFromToken(accessToken, jwtKey string) (*string, error) {
	result, errDecode := DecodeJWT(accessToken, jwtKey)
	if errDecode != nil {
		return nil, fmt.Errorf("invalid token format")
	}

	userId, ok := result["username"].(string)
	if !ok || userId == "" {
		return nil, fmt.Errorf("invalid token claims")
	}

	return &userId, nil
}

func ParseUUID(context *gin.Context, param string) (uuid.UUID, error) {
	id, err := uuid.Parse(context.Param(param))
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid %s", param)
	}
	return id, nil
}

func GetChatHub() *hub.ChatHub {
	return hub.GetChatHub()
}

var telegramUsernameRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{3,30}[a-zA-Z0-9]$`)

// IsValidTelegramUsername validates a Telegram username (without the leading '@').
// Rules: 5–32 chars, letters/numbers/underscores, cannot start or end with underscore.
func IsValidTelegramUsername(username string) bool {
	if len(username) < 5 || len(username) > 32 {
		return false
	}
	return telegramUsernameRegex.MatchString(username)
}
