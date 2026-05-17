package mocks

import (
	"backend/models/domains"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type KnowledgeBaseRepoMock struct {
	mock.Mock
}

func (m *KnowledgeBaseRepoMock) FindAll(db *gorm.DB, schema string) ([]domains.KnowledgeBase, error) {
	args := m.Called(db, schema)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domains.KnowledgeBase), args.Error(1)
}

func (m *KnowledgeBaseRepoMock) Create(db *gorm.DB, schema string, kb domains.KnowledgeBase) (domains.KnowledgeBase, error) {
	args := m.Called(db, schema, kb)
	return args.Get(0).(domains.KnowledgeBase), args.Error(1)
}

func (m *KnowledgeBaseRepoMock) FindByID(db *gorm.DB, schema string, id uuid.UUID) (*domains.KnowledgeBase, error) {
	args := m.Called(db, schema, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domains.KnowledgeBase), args.Error(1)
}

func (m *KnowledgeBaseRepoMock) Delete(db *gorm.DB, schema string, id uuid.UUID) error {
	args := m.Called(db, schema, id)
	return args.Error(0)
}
