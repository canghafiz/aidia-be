package repositories

import (
	"backend/models/domains"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type KnowledgeBaseRepo interface {
	FindAll(db *gorm.DB, schema string) ([]domains.KnowledgeBase, error)
	Create(db *gorm.DB, schema string, kb domains.KnowledgeBase) (domains.KnowledgeBase, error)
	FindByID(db *gorm.DB, schema string, id uuid.UUID) (*domains.KnowledgeBase, error)
	Delete(db *gorm.DB, schema string, id uuid.UUID) error
}
