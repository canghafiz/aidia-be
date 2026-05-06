package helpers

import (
	"fmt"
	"net/smtp"
	"os"
	"strings"
)

func SendEmail(to []string, subject, body string) error {
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	username := os.Getenv("SMTP_USERNAME")
	password := os.Getenv("SMTP_PASSWORD")
	from := os.Getenv("SMTP_FROM")

	if host == "" || username == "" || password == "" {
		return fmt.Errorf("SMTP_HOST, SMTP_USERNAME, SMTP_PASSWORD are required")
	}
	if port == "" {
		port = "587"
	}
	if from == "" {
		from = username
	}

	auth := smtp.PlainAuth("", username, password, host)

	// SMTP envelope sender must be plain email, not "Name <email>" format
	envelopeFrom := username

	msg := "From: " + from + "\r\n" +
		"To: " + strings.Join(to, ", ") + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=\"utf-8\"\r\n" +
		"\r\n" +
		body

	return smtp.SendMail(host+":"+port, auth, envelopeFrom, to, []byte(msg))
}
