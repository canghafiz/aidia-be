package domains

import (
	"github.com/google/uuid"
	"time"
)

type KnowledgeBase struct {
	ID            uuid.UUID `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	FileName      string    `gorm:"column:file_name"                               json:"file_name"`
	FileURL       string    `gorm:"column:file_url"                                json:"file_url"`
	ExtractedText string    `gorm:"column:extracted_text"                          json:"extracted_text,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (KnowledgeBase) TableName() string {
	return "knowledge_base"
}
