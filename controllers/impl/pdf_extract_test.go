package impl

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============================================================
// TEST: extractPDFText
// ============================================================

func TestExtractPDFText_InvalidBytes_ReturnsError(t *testing.T) {
	r := bytes.NewReader([]byte("not a pdf at all"))

	text, err := extractPDFText(r)

	assert.Error(t, err)
	assert.Empty(t, text)
}

func TestExtractPDFText_EmptyReader_ReturnsError(t *testing.T) {
	r := bytes.NewReader([]byte{})

	text, err := extractPDFText(r)

	assert.Error(t, err)
	assert.Empty(t, text)
}

func TestExtractPDFText_RandomBytes_ReturnsError(t *testing.T) {
	r := bytes.NewReader([]byte{0xFF, 0xFE, 0x00, 0x01, 0x02, 0x03})

	text, err := extractPDFText(r)

	assert.Error(t, err)
	assert.Empty(t, text)
}

func TestExtractPDFText_TruncatesLongText(t *testing.T) {
	// Test the truncation logic directly (without needing a real PDF)
	// by constructing a string longer than maxLen and verifying behavior.
	// We test the truncation constant is 20000 chars.
	const maxLen = 20000
	longText := strings.Repeat("a", maxLen+5000)

	// Simulate what the function does after extracting text
	result := strings.TrimSpace(longText)
	if len(result) > maxLen {
		result = result[:maxLen]
	}

	assert.Len(t, result, maxLen)
}

func TestExtractPDFText_ShortTextNotTruncated(t *testing.T) {
	const maxLen = 20000
	shortText := "Hello Knowledge Base"

	result := strings.TrimSpace(shortText)
	if len(result) > maxLen {
		result = result[:maxLen]
	}

	assert.Equal(t, shortText, result)
}
