package knowledge_base_test

import (
	"backend/helpers"
	"backend/models/domains"
	"backend/models/repositories/impl"
	testhelper "backend/testing"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ============================================================
// HELPERS
// ============================================================

const kbTestSchema = "kb_test_schema"

func setupKBSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	err := helpers.CreateTenantSchema(db, kbTestSchema)
	require.NoError(t, err, "failed to create test schema")
}

func teardownKBSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	err := helpers.DropTenantSchema(db, kbTestSchema)
	require.NoError(t, err, "failed to drop test schema")
}

func dummyKB() domains.KnowledgeBase {
	return domains.KnowledgeBase{
		FileName:      "test_" + uuid.New().String()[:8] + ".pdf",
		FileURL:       "https://storage.example.com/test.pdf",
		ExtractedText: "This is extracted text for testing purposes.",
	}
}

// ============================================================
// TEST: FindAll
// ============================================================

func TestKBFindAll_ReturnsEmpty_WhenNoItems(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewKnowledgeBaseRepoImpl()
	setupKBSchema(t, db)
	defer teardownKBSchema(t, db)

	items, err := repo.FindAll(db, kbTestSchema)

	assert.NoError(t, err)
	assert.Empty(t, items)
}

func TestKBFindAll_ReturnsItems_AfterInsert(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewKnowledgeBaseRepoImpl()
	setupKBSchema(t, db)
	defer teardownKBSchema(t, db)

	kb1 := dummyKB()
	kb2 := dummyKB()
	_, err := repo.Create(db, kbTestSchema, kb1)
	require.NoError(t, err)
	_, err = repo.Create(db, kbTestSchema, kb2)
	require.NoError(t, err)

	items, err := repo.FindAll(db, kbTestSchema)

	assert.NoError(t, err)
	assert.Len(t, items, 2)
}

func TestKBFindAll_OrderedByCreatedAtDesc(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewKnowledgeBaseRepoImpl()
	setupKBSchema(t, db)
	defer teardownKBSchema(t, db)

	first := dummyKB()
	first.FileName = "first.pdf"
	second := dummyKB()
	second.FileName = "second.pdf"

	_, err := repo.Create(db, kbTestSchema, first)
	require.NoError(t, err)
	_, err = repo.Create(db, kbTestSchema, second)
	require.NoError(t, err)

	items, err := repo.FindAll(db, kbTestSchema)

	assert.NoError(t, err)
	assert.Len(t, items, 2)
	// Most recent first
	assert.Equal(t, "second.pdf", items[0].FileName)
	assert.Equal(t, "first.pdf", items[1].FileName)
}

// ============================================================
// TEST: Create
// ============================================================

func TestKBCreate_Success(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewKnowledgeBaseRepoImpl()
	setupKBSchema(t, db)
	defer teardownKBSchema(t, db)

	kb := dummyKB()
	created, err := repo.Create(db, kbTestSchema, kb)

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, created.ID)
	assert.Equal(t, kb.FileName, created.FileName)
	assert.Equal(t, kb.FileURL, created.FileURL)
	assert.Equal(t, kb.ExtractedText, created.ExtractedText)
	assert.False(t, created.CreatedAt.IsZero())
}

func TestKBCreate_WithEmptyExtractedText(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewKnowledgeBaseRepoImpl()
	setupKBSchema(t, db)
	defer teardownKBSchema(t, db)

	kb := dummyKB()
	kb.ExtractedText = ""

	created, err := repo.Create(db, kbTestSchema, kb)

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, created.ID)
	assert.Empty(t, created.ExtractedText)
}

func TestKBCreate_PersistsToDatabase(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewKnowledgeBaseRepoImpl()
	setupKBSchema(t, db)
	defer teardownKBSchema(t, db)

	kb := dummyKB()
	created, err := repo.Create(db, kbTestSchema, kb)
	require.NoError(t, err)

	found, err := repo.FindByID(db, kbTestSchema, created.ID)

	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, kb.FileName, found.FileName)
}

// ============================================================
// TEST: FindByID
// ============================================================

func TestKBFindByID_Success(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewKnowledgeBaseRepoImpl()
	setupKBSchema(t, db)
	defer teardownKBSchema(t, db)

	kb := dummyKB()
	created, err := repo.Create(db, kbTestSchema, kb)
	require.NoError(t, err)

	found, err := repo.FindByID(db, kbTestSchema, created.ID)

	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, kb.FileName, found.FileName)
	assert.Equal(t, kb.FileURL, found.FileURL)
	assert.Equal(t, kb.ExtractedText, found.ExtractedText)
}

func TestKBFindByID_NotFound_ReturnsError(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewKnowledgeBaseRepoImpl()
	setupKBSchema(t, db)
	defer teardownKBSchema(t, db)

	found, err := repo.FindByID(db, kbTestSchema, uuid.New())

	assert.Error(t, err)
	assert.Nil(t, found)
}

// ============================================================
// TEST: Delete
// ============================================================

func TestKBDelete_Success(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewKnowledgeBaseRepoImpl()
	setupKBSchema(t, db)
	defer teardownKBSchema(t, db)

	kb := dummyKB()
	created, err := repo.Create(db, kbTestSchema, kb)
	require.NoError(t, err)

	err = repo.Delete(db, kbTestSchema, created.ID)
	assert.NoError(t, err)

	found, err := repo.FindByID(db, kbTestSchema, created.ID)
	assert.Error(t, err)
	assert.Nil(t, found)
}

func TestKBDelete_NonExistentID_NoError(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewKnowledgeBaseRepoImpl()
	setupKBSchema(t, db)
	defer teardownKBSchema(t, db)

	// Deleting non-existent ID should not error (GORM soft-delete pattern)
	err := repo.Delete(db, kbTestSchema, uuid.New())

	assert.NoError(t, err)
}

func TestKBDelete_DoesNotAffectOtherRows(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewKnowledgeBaseRepoImpl()
	setupKBSchema(t, db)
	defer teardownKBSchema(t, db)

	kb1 := dummyKB()
	kb2 := dummyKB()
	created1, err := repo.Create(db, kbTestSchema, kb1)
	require.NoError(t, err)
	created2, err := repo.Create(db, kbTestSchema, kb2)
	require.NoError(t, err)

	err = repo.Delete(db, kbTestSchema, created1.ID)
	assert.NoError(t, err)

	// kb2 should still exist
	found, err := repo.FindByID(db, kbTestSchema, created2.ID)
	assert.NoError(t, err)
	assert.NotNil(t, found)

	// Full list should have only 1 item
	items, err := repo.FindAll(db, kbTestSchema)
	assert.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, created2.ID, items[0].ID)
}
