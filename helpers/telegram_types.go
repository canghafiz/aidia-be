package helpers

// TelegramKeyboard represents Telegram ReplyKeyboardMarkup
type TelegramKeyboard struct {
	Keyboard        [][]TelegramKeyboardButton `json:"keyboard"`
	ResizeKeyboard  bool                       `json:"resize_keyboard"`
	OneTimeKeyboard bool                       `json:"one_time_keyboard"`
	Selective       bool                       `json:"selective"`
}

// TelegramKeyboardButton represents a button in the keyboard
type TelegramKeyboardButton struct {
	Text           string `json:"text"`
	RequestContact bool   `json:"request_contact"`
}

// TelegramInlineKeyboard represents Telegram InlineKeyboardMarkup
type TelegramInlineKeyboard struct {
	InlineKeyboard [][]TelegramInlineButton `json:"inline_keyboard"`
}

// TelegramInlineButton represents a button in the inline keyboard
type TelegramInlineButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
	URL          string `json:"url,omitempty"`
}

// TelegramMessageWithKeyboard represents the request body for sendMessage with keyboard
type TelegramMessageWithKeyboard struct {
	ChatID      string            `json:"chat_id"`
	Text        string            `json:"text"`
	ReplyMarkup *TelegramKeyboard `json:"reply_markup,omitempty"`
}

// TelegramMessageWithInlineKeyboard represents the request body for sendMessage with inline keyboard
type TelegramMessageWithInlineKeyboard struct {
	ChatID      string                 `json:"chat_id"`
	Text        string                 `json:"text"`
	ReplyMarkup *TelegramInlineKeyboard `json:"reply_markup,omitempty"`
}
