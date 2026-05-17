package knowledge_base_test

import (
	contImpl "backend/controllers/impl"
	"backend/models/domains"
	"backend/testing/mocks"
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================
// HELPERS
// ============================================================

const kbContTestSchema = "kb_test_schema"

func setupKBCont() (*contImpl.SettingContImpl, *mocks.KnowledgeBaseRepoMock, *mocks.UsersRepoMock, *mocks.FileServMock) {
	kbRepo := new(mocks.KnowledgeBaseRepoMock)
	userRepo := new(mocks.UsersRepoMock)
	fileServ := new(mocks.FileServMock)
	cont := contImpl.NewSettingContImpl(nil, userRepo, nil, fileServ, kbRepo)
	return cont, kbRepo, userRepo, fileServ
}

func setupKBRouter(cont *contImpl.SettingContImpl) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/client/:client_id/knowledge-base", cont.UploadKnowledgeBase)
	r.GET("/api/v1/client/:client_id/knowledge-base", cont.ListKnowledgeBase)
	r.DELETE("/api/v1/client/:client_id/knowledge-base/:kb_id", cont.DeleteKnowledgeBase)
	return r
}

func mockSchema(userRepo *mocks.UsersRepoMock, clientID uuid.UUID) {
	schema := kbContTestSchema
	user := domains.Users{UserID: clientID, TenantSchema: &schema}
	userRepo.On("GetByUserId", mock.Anything, clientID).Return(&user, nil)
}

func mockSchemaError(userRepo *mocks.UsersRepoMock, clientID uuid.UUID) {
	userRepo.On("GetByUserId", mock.Anything, clientID).Return(nil, errors.New("not found"))
}

func buildMultipartRequest(t *testing.T, url, fieldName, filename string, content []byte) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(fieldName, filename)
	if err != nil {
		t.Fatal(err)
	}
	_, err = part.Write(content)
	if err != nil {
		t.Fatal(err)
	}
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, url, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func parseKBResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	return result
}

var invalidPDFBytes = []byte("not a real pdf content")

// ============================================================
// TEST: UploadKnowledgeBase
// ============================================================

func TestKBUpload_Success_ExtractionFails_StillSaves(t *testing.T) {
	cont, kbRepo, userRepo, fileServ := setupKBCont()
	r := setupKBRouter(cont)
	clientID := uuid.New()
	mockSchema(userRepo, clientID)

	fileServ.On("UploadFiles", mock.Anything).Return([]domains.File{{FileName: "test.pdf", FileURL: "https://s3.example.com/test.pdf"}}, nil)

	kbRepo.On("Create", mock.Anything, kbContTestSchema, mock.MatchedBy(func(kb domains.KnowledgeBase) bool {
		return kb.FileName == "test.pdf" && kb.FileURL == "https://s3.example.com/test.pdf"
	})).Return(domains.KnowledgeBase{
		ID: uuid.New(), FileName: "test.pdf", FileURL: "https://s3.example.com/test.pdf", ExtractedText: "",
	}, nil)

	url := "/api/v1/client/" + clientID.String() + "/knowledge-base"
	req := buildMultipartRequest(t, url, "file", "test.pdf", invalidPDFBytes)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseKBResponse(w)
	assert.True(t, resp["success"].(bool))
	kbRepo.AssertExpectations(t)
	fileServ.AssertExpectations(t)
}

func TestKBUpload_RejectsNonPDF(t *testing.T) {
	cont, _, userRepo, _ := setupKBCont()
	r := setupKBRouter(cont)
	clientID := uuid.New()
	mockSchema(userRepo, clientID)

	url := "/api/v1/client/" + clientID.String() + "/knowledge-base"
	req := buildMultipartRequest(t, url, "file", "document.docx", []byte("docx content"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseKBResponse(w)
	assert.False(t, resp["success"].(bool))
}

func TestKBUpload_RejectsFileTooLarge(t *testing.T) {
	cont, _, userRepo, _ := setupKBCont()
	r := setupKBRouter(cont)
	clientID := uuid.New()
	mockSchema(userRepo, clientID)

	largeContent := make([]byte, 11*1024*1024)

	url := "/api/v1/client/" + clientID.String() + "/knowledge-base"
	req := buildMultipartRequest(t, url, "file", "big.pdf", largeContent)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseKBResponse(w)
	assert.False(t, resp["success"].(bool))
}

func TestKBUpload_RejectsNoFile(t *testing.T) {
	cont, _, userRepo, _ := setupKBCont()
	r := setupKBRouter(cont)
	clientID := uuid.New()
	mockSchema(userRepo, clientID)

	url := "/api/v1/client/" + clientID.String() + "/knowledge-base"
	req := httptest.NewRequest(http.MethodPost, url, strings.NewReader(""))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseKBResponse(w)
	assert.False(t, resp["success"].(bool))
}

func TestKBUpload_FailsWhenFileServErrors(t *testing.T) {
	cont, _, userRepo, fileServ := setupKBCont()
	r := setupKBRouter(cont)
	clientID := uuid.New()
	mockSchema(userRepo, clientID)

	fileServ.On("UploadFiles", mock.Anything).Return(nil, errors.New("storage error"))

	url := "/api/v1/client/" + clientID.String() + "/knowledge-base"
	req := buildMultipartRequest(t, url, "file", "test.pdf", invalidPDFBytes)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	resp := parseKBResponse(w)
	assert.False(t, resp["success"].(bool))
}

func TestKBUpload_FailsWhenDBErrors(t *testing.T) {
	cont, kbRepo, userRepo, fileServ := setupKBCont()
	r := setupKBRouter(cont)
	clientID := uuid.New()
	mockSchema(userRepo, clientID)

	fileServ.On("UploadFiles", mock.Anything).Return([]domains.File{{FileName: "test.pdf", FileURL: "https://s3.example.com/test.pdf"}}, nil)
	kbRepo.On("Create", mock.Anything, kbContTestSchema, mock.Anything).Return(domains.KnowledgeBase{}, errors.New("db error"))

	url := "/api/v1/client/" + clientID.String() + "/knowledge-base"
	req := buildMultipartRequest(t, url, "file", "test.pdf", invalidPDFBytes)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	resp := parseKBResponse(w)
	assert.False(t, resp["success"].(bool))
}

func TestKBUpload_FailsWhenClientIDInvalid(t *testing.T) {
	cont, _, _, _ := setupKBCont()
	r := setupKBRouter(cont)

	req := buildMultipartRequest(t, "/api/v1/client/not-a-uuid/knowledge-base", "file", "test.pdf", invalidPDFBytes)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestKBUpload_FailsWhenUserNotFound(t *testing.T) {
	cont, _, userRepo, _ := setupKBCont()
	r := setupKBRouter(cont)
	clientID := uuid.New()
	mockSchemaError(userRepo, clientID)

	url := "/api/v1/client/" + clientID.String() + "/knowledge-base"
	req := buildMultipartRequest(t, url, "file", "test.pdf", invalidPDFBytes)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestKBUpload_AcceptsPDFWithUppercaseExtension(t *testing.T) {
	cont, kbRepo, userRepo, fileServ := setupKBCont()
	r := setupKBRouter(cont)
	clientID := uuid.New()
	mockSchema(userRepo, clientID)

	fileServ.On("UploadFiles", mock.Anything).Return([]domains.File{{FileName: "DOC.PDF", FileURL: "https://s3.example.com/DOC.PDF"}}, nil)
	kbRepo.On("Create", mock.Anything, kbContTestSchema, mock.Anything).Return(domains.KnowledgeBase{
		ID: uuid.New(), FileName: "DOC.PDF", FileURL: "https://s3.example.com/DOC.PDF",
	}, nil)

	url := "/api/v1/client/" + clientID.String() + "/knowledge-base"
	req := buildMultipartRequest(t, url, "file", "DOC.PDF", invalidPDFBytes)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ============================================================
// TEST: ListKnowledgeBase
// ============================================================

func TestKBList_Success_ReturnsItems(t *testing.T) {
	cont, kbRepo, userRepo, _ := setupKBCont()
	r := setupKBRouter(cont)
	clientID := uuid.New()
	mockSchema(userRepo, clientID)

	items := []domains.KnowledgeBase{
		{ID: uuid.New(), FileName: "a.pdf", FileURL: "https://s3.example.com/a.pdf", ExtractedText: "text a"},
		{ID: uuid.New(), FileName: "b.pdf", FileURL: "https://s3.example.com/b.pdf", ExtractedText: "text b"},
	}
	kbRepo.On("FindAll", mock.Anything, kbContTestSchema).Return(items, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/client/"+clientID.String()+"/knowledge-base", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseKBResponse(w)
	assert.True(t, resp["success"].(bool))
	data := resp["data"].([]interface{})
	assert.Len(t, data, 2)
}

func TestKBList_Success_EmptyList(t *testing.T) {
	cont, kbRepo, userRepo, _ := setupKBCont()
	r := setupKBRouter(cont)
	clientID := uuid.New()
	mockSchema(userRepo, clientID)

	kbRepo.On("FindAll", mock.Anything, kbContTestSchema).Return([]domains.KnowledgeBase{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/client/"+clientID.String()+"/knowledge-base", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseKBResponse(w)
	assert.True(t, resp["success"].(bool))
}

func TestKBList_FailsWhenDBErrors(t *testing.T) {
	cont, kbRepo, userRepo, _ := setupKBCont()
	r := setupKBRouter(cont)
	clientID := uuid.New()
	mockSchema(userRepo, clientID)

	kbRepo.On("FindAll", mock.Anything, kbContTestSchema).Return(nil, errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/client/"+clientID.String()+"/knowledge-base", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	resp := parseKBResponse(w)
	assert.False(t, resp["success"].(bool))
}

func TestKBList_FailsWhenClientIDInvalid(t *testing.T) {
	cont, _, _, _ := setupKBCont()
	r := setupKBRouter(cont)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/client/bad-uuid/knowledge-base", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestKBList_FailsWhenUserNotFound(t *testing.T) {
	cont, _, userRepo, _ := setupKBCont()
	r := setupKBRouter(cont)
	clientID := uuid.New()
	mockSchemaError(userRepo, clientID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/client/"+clientID.String()+"/knowledge-base", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusOK, w.Code)
}

// ============================================================
// TEST: DeleteKnowledgeBase
// ============================================================

func TestKBContDelete_Success(t *testing.T) {
	cont, kbRepo, userRepo, _ := setupKBCont()
	r := setupKBRouter(cont)
	clientID := uuid.New()
	kbID := uuid.New()
	mockSchema(userRepo, clientID)

	kbRepo.On("Delete", mock.Anything, kbContTestSchema, kbID).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/client/"+clientID.String()+"/knowledge-base/"+kbID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseKBResponse(w)
	assert.True(t, resp["success"].(bool))
	kbRepo.AssertExpectations(t)
}

func TestKBDelete_FailsOnInvalidKBID(t *testing.T) {
	cont, _, userRepo, _ := setupKBCont()
	r := setupKBRouter(cont)
	clientID := uuid.New()
	mockSchema(userRepo, clientID)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/client/"+clientID.String()+"/knowledge-base/not-a-uuid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseKBResponse(w)
	assert.False(t, resp["success"].(bool))
}

func TestKBDelete_FailsOnInvalidClientID(t *testing.T) {
	cont, _, _, _ := setupKBCont()
	r := setupKBRouter(cont)
	kbID := uuid.New()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/client/bad-uuid/knowledge-base/"+kbID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestKBDelete_FailsWhenDBErrors(t *testing.T) {
	cont, kbRepo, userRepo, _ := setupKBCont()
	r := setupKBRouter(cont)
	clientID := uuid.New()
	kbID := uuid.New()
	mockSchema(userRepo, clientID)

	kbRepo.On("Delete", mock.Anything, kbContTestSchema, kbID).Return(errors.New("db error"))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/client/"+clientID.String()+"/knowledge-base/"+kbID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	resp := parseKBResponse(w)
	assert.False(t, resp["success"].(bool))
}

func TestKBDelete_FailsWhenUserNotFound(t *testing.T) {
	cont, _, userRepo, _ := setupKBCont()
	r := setupKBRouter(cont)
	clientID := uuid.New()
	kbID := uuid.New()
	mockSchemaError(userRepo, clientID)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/client/"+clientID.String()+"/knowledge-base/"+kbID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusOK, w.Code)
}
