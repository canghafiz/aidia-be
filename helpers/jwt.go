package helpers

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateJWT(jwtKey string, duration time.Duration, data interface{}) (string, error) {
	if jwtKey == "" {
		return "", fmt.Errorf("JWT key is empty")
	}

	claims := jwt.MapClaims{
		"data": data,
		"exp":  time.Now().Add(duration).Unix(),
		"iat":  time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(jwtKey))
}

func DecodeJWT(tokenString, jwtKey string) (map[string]interface{}, error) {
	if jwtKey == "" {
		return nil, fmt.Errorf("JWT key is empty")
	}

	// Parse token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(jwtKey), nil
	})

	if err != nil {
		return nil, err
	}

	// Get Claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Get field "data"
	data, ok := claims["data"].(map[string]interface{})
	if !ok {
		return map[string]interface{}{
			"data": claims["data"],
		}, nil
	}

	return data, nil
}
