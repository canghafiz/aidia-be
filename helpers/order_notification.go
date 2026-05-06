package helpers

import (
	"fmt"
	"log"
	"strings"

	"gorm.io/gorm"
)

type notificationSetting struct {
	waNumber1 string
	waNumber2 string
	email1    string
	email2    string
}

func readNotificationSettings(db *gorm.DB, schema string) notificationSetting {
	var ns notificationSetting

	type settingRow struct {
		SubGroupName string
		Name         string
		Value        string
	}

	var rows []settingRow
	db.Raw(
		fmt.Sprintf(`SELECT sub_group_name, name, value FROM %q.setting WHERE group_name = 'notification'`, schema),
	).Scan(&rows)

	for _, r := range rows {
		if r.SubGroupName != "Status New / Confirmed Order" {
			continue
		}
		switch r.Name {
		case "new-order-wa-number":
			ns.waNumber1 = strings.TrimSpace(r.Value)
		case "new-order-wa-number-2":
			ns.waNumber2 = strings.TrimSpace(r.Value)
		case "new-order-email":
			ns.email1 = strings.TrimSpace(r.Value)
		case "new-order-email-2":
			ns.email2 = strings.TrimSpace(r.Value)
		}
	}

	return ns
}

// SendOrderNotification fires off WA and email notifications for a new order.
// waClient is nil if no WA integration is configured; errors are logged, none returned.
// Designed to be called as a goroutine — all errors are logged, none are returned.
func SendOrderNotification(db *gorm.DB, schema, orderID, customerName string, totalPrice float64, waClient WhatsAppSender) {
	ns := readNotificationSettings(db, schema)

	waMsg := fmt.Sprintf(
		"🛎 *New Order!*\n\nOrder #%s\nCustomer: %s\nTotal: $ %.0f\n\nPlease check the dashboard to process the order.",
		orderID, customerName, totalPrice,
	)
	emailBody := fmt.Sprintf(
		"A new order has been placed.\n\nOrder #%s\nCustomer: %s\nTotal: Rp %.0f\n\nOpen the dashboard to process the order.",
		orderID, customerName, totalPrice,
	)

	// WhatsApp — Meta Cloud API only
	if waClient != nil {
		waNumbers := []string{}
		if ns.waNumber1 != "" {
			waNumbers = append(waNumbers, ns.waNumber1)
		}
		if ns.waNumber2 != "" {
			waNumbers = append(waNumbers, ns.waNumber2)
		}
		for _, num := range waNumbers {
			if err := waClient.SendMessage(num, waMsg); err != nil {
				log.Printf("[OrderNotification] WA send to %s failed: %v", num, err)
			} else {
				log.Printf("[OrderNotification] WA sent to %s", num)
			}
		}
	}

	// Email via Resend
	emailTo := []string{}
	if ns.email1 != "" {
		emailTo = append(emailTo, ns.email1)
	}
	if ns.email2 != "" {
		emailTo = append(emailTo, ns.email2)
	}
	if len(emailTo) > 0 {
		subject := fmt.Sprintf("New Order #%s - %s", orderID, customerName)
		if err := SendEmail(emailTo, subject, emailBody); err != nil {
			log.Printf("[OrderNotification] Email send failed: %v", err)
		} else {
			log.Printf("[OrderNotification] Email sent to %v", emailTo)
		}
	}
}
