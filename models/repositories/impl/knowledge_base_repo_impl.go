package impl

import (
	"backend/models/domains"
	"backend/models/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type KnowledgeBaseRepoImpl struct{}

func NewKnowledgeBaseRepoImpl() repositories.KnowledgeBaseRepo {
	return &KnowledgeBaseRepoImpl{}
}

func (r *KnowledgeBaseRepoImpl) FindAll(db *gorm.DB, schema string) ([]domains.KnowledgeBase, error) {
	var items []domains.KnowledgeBase
	err := db.Raw(
		"SELECT id, file_name, file_url, extracted_text, created_at, updated_at FROM "+schema+".knowledge_base ORDER BY created_at DESC",
	).Scan(&items).Error
	if items == nil {
		items = []domains.KnowledgeBase{}
	}
	return items, err
}

func (r *KnowledgeBaseRepoImpl) Create(db *gorm.DB, schema string, kb domains.KnowledgeBase) (domains.KnowledgeBase, error) {
	var created domains.KnowledgeBase
	err := db.Raw(
		"INSERT INTO "+schema+".knowledge_base (file_name, file_url, extracted_text) VALUES (?, ?, ?) RETURNING id, file_name, file_url, extracted_text, created_at, updated_at",
		kb.FileName, kb.FileURL, kb.ExtractedText,
	).Scan(&created).Error
	return created, err
}

func (r *KnowledgeBaseRepoImpl) FindByID(db *gorm.DB, schema string, id uuid.UUID) (*domains.KnowledgeBase, error) {
	var kb domains.KnowledgeBase
	err := db.Raw(
		"SELECT id, file_name, file_url, extracted_text, created_at, updated_at FROM "+schema+".knowledge_base WHERE id = ?",
		id,
	).Scan(&kb).Error
	if err != nil {
		return nil, err
	}
	if kb.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}
	return &kb, nil
}

func (r *KnowledgeBaseRepoImpl) Delete(db *gorm.DB, schema string, id uuid.UUID) error {
	return db.Exec("DELETE FROM "+schema+".knowledge_base WHERE id = ?", id).Error
}
