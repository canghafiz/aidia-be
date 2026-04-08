package services

import (
	subsRes "backend/models/responses/subs"
)

type SubsServ interface {
	GetCurrentSubs(accessToken string) (*subsRes.Response, error)
	GetTokenUsage(accessToken string) (*subsRes.TokenUsageResponse, error)
}
