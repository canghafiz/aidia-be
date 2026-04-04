package services

import (
	"backend/models/domains"

	"github.com/google/uuid"
)

type ChatServ interface {
	GetConversations(accessToken string, clientID uuid.UUID, pagination domains.Pagination) (interface{}, error)
	GetConversationDetail(accessToken string, clientID, guestID uuid.UUID, beforeID *uuid.UUID, limit int) (interface{}, error)
	MarkAsRead(accessToken string, clientID, guestID uuid.UUID) error
	SendManualReply(accessToken string, clientID, guestID uuid.UUID, message string) error
}
