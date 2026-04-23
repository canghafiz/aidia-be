package services

import (
	"backend/models/domains"

	"github.com/google/uuid"
)

type ChatServ interface {
	GetConversations(accessToken string, clientID uuid.UUID, platform string, pagination domains.Pagination) (interface{}, error)
	GetConversationDetail(accessToken string, clientID, guestID uuid.UUID, platform string, beforeID *uuid.UUID, limit int) (interface{}, error)
	MarkAsRead(accessToken string, clientID, guestID uuid.UUID) error
	SendManualReply(accessToken string, clientID, guestID uuid.UUID, message, platform string) error
	SendTemplateMessage(accessToken string, clientID, guestID uuid.UUID, templateName, languageCode string, bodyParams []string) error
	InitTelegramChat(accessToken string, clientID uuid.UUID, customerID int) (string, error)
}
