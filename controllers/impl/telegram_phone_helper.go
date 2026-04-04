package impl

import (
	"backend/helpers"
	"log"
)

// sendPhoneRequest sends a phone number request to a new guest
func (cont *TelegramContImpl) sendPhoneRequest(schema, chatID, botToken string) {
	if botToken == "" {
		log.Printf("[PhoneRequest] bot token empty for schema: %s", schema)
		return
	}

	tgClient := helpers.NewTelegramClient(botToken)

	// Create custom keyboard with contact button
	keyboard := &helpers.ReplyKeyboardMarkup{
		Keyboard: [][]helpers.KeyboardButton{
			{
				{
					Text:           "📱 Share Phone Number",
					RequestContact: true,
				},
			},
		},
		ResizeKeyboard:  true,
		OneTimeKeyboard: true,
	}

	message := "👋 Welcome! To complete your registration, please share your phone number with us.\n\nClick the button below to share:"

	err := tgClient.SendMessageWithKeyboard(chatID, message, keyboard)
	if err != nil {
		log.Printf("[PhoneRequest] failed to send to %s: %v", chatID, err)
		return
	}

	log.Printf("[PhoneRequest] sent to %s", chatID)
}
