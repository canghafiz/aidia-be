package helpers

// WhatsAppSender abstracts sending WhatsApp messages.
// Both *WhatsAppClient (Meta Cloud API) and WhatsmeowSender implement this interface,
// allowing the controller to be agnostic about which backend is active per tenant.
type WhatsAppSender interface {
	SendMessage(to, text string) error
	SendTemplateMessage(to, templateName, languageCode string, bodyParams []string) error
	SendImageMessage(to, imageURL, caption string) error
}
