package helpers

import (
	"fmt"
	"time"
)

func GenerateInvoiceNumber(sequence int) string {
	now := time.Now()
	date := now.Format("20060102")
	return fmt.Sprintf("INV-AI%s%05d", date, sequence)
}
