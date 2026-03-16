package controllers_test

import (
	contImpl "backend/controllers/impl"
	"backend/helpers"
	"backend/middlewares"
	"backend/models/domains"
	"backend/models/requests/user"
	"backend/models/responses/pagination"
	res "backend/models/responses/user"
	"backend/testing/mocks"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ============================================================
// HELPERS
// ============================================================

const testJwtKey = "test-secret-key"

func generateMiddlewareJWT(userID string) string {
	claims := map[string]interface{}{
		"user_id":  userID,
		"username": "testuser",
		"name":     "Test User",
		"email":    "test@mail.com",
		"gender":   "Male",
		"role":     "SuperAdmin",
	}
	token, _ := helpers.GenerateJWT(testJwtKey, 24*time.Hour, claims)
	return token
}

func setupRouter(cont *contImpl.UsersContImpl, repoMock *mocks.UsersRepoMock) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	mw := middlewares.Middleware(nil, repoMock, testJwtKey)

	r.POST("/api/v1/auth/login", cont.Login)
	r.POST("/api/v1/auth/create-superadmin", cont.CreateSuperAdmin)
	r.GET("/api/v1/auth/check-superadmin-exist", cont.CheckSuperAdminExist)

	protected := r.Group("/api/v1/auth")
	protected.Use(mw)
	{
		protected.GET("/me", cont.Me)
		protected.PATCH("/change-password", cont.ChangePw)
		protected.PATCH("/refresh-token", cont.RefreshToken)
		protected.DELETE("/logout", cont.Logout)
	}

	users := r.Group("/api/v1/users")
	users.GET("", cont.GetUsers)
	users.GET("/filter", cont.FilterUsers)
	users.GET("/:user_id", cont.GetByUserId)

	usersProtected := r.Group("/api/v1/users")
	usersProtected.Use(mw)
	usersProtected.POST("", cont.CreateUser)
	usersProtected.PUT("/:user_id/client", cont.UpdateProfileClient)
	usersProtected.PUT("/:user_id/non-client", cont.UpdateProfileNonClient)
	usersProtected.PUT("/:user_id/other", cont.EditUserData)
	usersProtected.PATCH("/:user_id/status", cont.UpdateUserStatusActive)
	usersProtected.DELETE("/:user_id", cont.DeleteByUserId)

	return r
}

func setupCont() (*contImpl.UsersContImpl, *mocks.UsersServMock, *mocks.UsersRepoMock) {
	servMock := new(mocks.UsersServMock)
	repoMock := new(mocks.UsersRepoMock)
	cont := contImpl.NewUsersContImpl(servMock)
	return cont, servMock, repoMock
}

func makeJSON(v interface{}) *bytes.Buffer {
	b, _ := json.Marshal(v)
	return bytes.NewBuffer(b)
}

func withBearer(req *http.Request, token string) *http.Request {
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func parseResponse(w *httptest.ResponseRecorder) map[string]interface{} {
	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	return result
}

func mockTokenValid(repoMock *mocks.UsersRepoMock, userID uuid.UUID) {
	repoMock.On("CheckTokenValid", mock.Anything, mock.MatchedBy(func(u domains.Users) bool {
		return u.UserID == userID
	})).Return(true)
}

// ============================================================
// TEST: Middleware
// ============================================================

func TestMiddleware_ShouldFail_NoAuthHeader(t *testing.T) {
	cont, _, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMiddleware_ShouldFail_InvalidBearerFormat(t *testing.T) {
	cont, _, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "InvalidFormat token")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMiddleware_ShouldFail_InvalidToken(t *testing.T) {
	cont, _, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	withBearer(req, "invalid.jwt.token")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMiddleware_ShouldFail_TokenRevokedInDB(t *testing.T) {
	cont, _, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	userID := uuid.New()
	token := generateMiddlewareJWT(userID.String())

	repoMock.On("CheckTokenValid", mock.Anything, mock.MatchedBy(func(u domains.Users) bool {
		return u.UserID == userID
	})).Return(false)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	withBearer(req, token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	repoMock.AssertExpectations(t)
}

func TestMiddleware_ShouldPass_ValidToken(t *testing.T) {
	cont, servMock, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	userID := uuid.New()
	token := generateMiddlewareJWT(userID.String())

	mockTokenValid(repoMock, userID)

	expected := &res.Response{UserID: userID.String(), Username: "testuser"}
	servMock.On("Me", token).Return(expected, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	withBearer(req, token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	repoMock.AssertExpectations(t)
	servMock.AssertExpectations(t)
}

// ============================================================
// TEST: Login
// ============================================================

func TestLogin_Success(t *testing.T) {
	cont, servMock, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	loginResp := &res.LoginResponse{
		AccessToken: "generated-access-token",
		UserData: res.Response{
			UserID:   "some-uuid",
			Username: "testuser",
			Name:     "Test User",
			Email:    "test@mail.com",
			Gender:   "Male",
			Role:     "SuperAdmin",
		},
	}
	servMock.On("Login", user.LoginRequest{
		UsernameOrEmail: "test@mail.com",
		Password:        "password123",
	}).Return(loginResp, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", makeJSON(map[string]string{
		"username_or_email": "test@mail.com",
		"password":          "password123",
	}))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseResponse(w)
	assert.Equal(t, true, body["success"])
	servMock.AssertExpectations(t)
}

func TestLogin_ShouldFail_ServiceError(t *testing.T) {
	cont, servMock, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	servMock.On("Login", mock.AnythingOfType("user.LoginRequest")).
		Return(nil, errors.New("user not found"))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", makeJSON(map[string]string{
		"username_or_email": "ghost@mail.com",
		"password":          "password123",
	}))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	body := parseResponse(w)
	assert.Equal(t, false, body["success"])
	servMock.AssertExpectations(t)
}

func TestLogin_ShouldFail_InvalidBody(t *testing.T) {
	cont, _, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================
// TEST: Logout
// ============================================================

func TestLogout_Success(t *testing.T) {
	cont, servMock, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	userID := uuid.New()
	token := generateMiddlewareJWT(userID.String())

	mockTokenValid(repoMock, userID)
	servMock.On("Logout", token).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/logout", nil)
	withBearer(req, token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseResponse(w)
	assert.Equal(t, true, body["success"])
	servMock.AssertExpectations(t)
	repoMock.AssertExpectations(t)
}

func TestLogout_ShouldFail_NoToken(t *testing.T) {
	cont, _, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/logout", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ============================================================
// TEST: Me
// ============================================================

func TestMe_Success(t *testing.T) {
	cont, servMock, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	userID := uuid.New()
	token := generateMiddlewareJWT(userID.String())

	mockTokenValid(repoMock, userID)
	expected := &res.Response{UserID: userID.String(), Username: "testuser", Name: "Test User", Email: "test@mail.com", Gender: "Male", Role: "SuperAdmin"}
	servMock.On("Me", token).Return(expected, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	withBearer(req, token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseResponse(w)
	assert.Equal(t, true, body["success"])
	servMock.AssertExpectations(t)
	repoMock.AssertExpectations(t)
}

func TestMe_ShouldFail_NoToken(t *testing.T) {
	cont, _, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMe_ShouldFail_ServiceError(t *testing.T) {
	cont, servMock, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	userID := uuid.New()
	token := generateMiddlewareJWT(userID.String())

	mockTokenValid(repoMock, userID)
	servMock.On("Me", token).Return(nil, errors.New("user not found"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	withBearer(req, token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	body := parseResponse(w)
	assert.Equal(t, false, body["success"])
	servMock.AssertExpectations(t)
	repoMock.AssertExpectations(t)
}

// ============================================================
// TEST: ChangePw
// ============================================================

func TestChangePw_Success(t *testing.T) {
	cont, servMock, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	userID := uuid.New()
	token := generateMiddlewareJWT(userID.String())

	mockTokenValid(repoMock, userID)
	servMock.On("ChangePw", token, user.ChangePwRequest{
		CurrentPassword: "oldpassword123",
		NewPassword:     "newpassword123",
		ConfirmPassword: "newpassword123",
	}).Return(nil)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/change-password", makeJSON(map[string]string{
		"current_password": "oldpassword123",
		"new_password":     "newpassword123",
		"confirm_password": "newpassword123",
	}))
	req.Header.Set("Content-Type", "application/json")
	withBearer(req, token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseResponse(w)
	assert.Equal(t, true, body["success"])
	servMock.AssertExpectations(t)
	repoMock.AssertExpectations(t)
}

func TestChangePw_ShouldFail_NoToken(t *testing.T) {
	cont, _, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/change-password", makeJSON(map[string]string{
		"current_password": "oldpassword123",
		"new_password":     "newpassword123",
		"confirm_password": "newpassword123",
	}))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestChangePw_ShouldFail_InvalidBody(t *testing.T) {
	cont, _, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	userID := uuid.New()
	token := generateMiddlewareJWT(userID.String())

	mockTokenValid(repoMock, userID)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/change-password", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	withBearer(req, token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ============================================================
// TEST: CheckSuperAdminExist
// ============================================================

func TestCheckSuperAdminExist_True(t *testing.T) {
	cont, servMock, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	servMock.On("CheckSuperAdminExist").Return(true, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/check-superadmin-exist", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseResponse(w)
	assert.Equal(t, true, body["success"])
	assert.Equal(t, true, body["data"])
	servMock.AssertExpectations(t)
}

func TestCheckSuperAdminExist_False(t *testing.T) {
	cont, servMock, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	servMock.On("CheckSuperAdminExist").Return(false, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/check-superadmin-exist", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseResponse(w)
	assert.Equal(t, true, body["success"])
	assert.Equal(t, false, body["data"])
	servMock.AssertExpectations(t)
}

func TestCheckSuperAdminExist_ShouldFail_ServiceError(t *testing.T) {
	cont, servMock, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	servMock.On("CheckSuperAdminExist").Return(false, errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/check-superadmin-exist", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	body := parseResponse(w)
	assert.Equal(t, false, body["success"])
	servMock.AssertExpectations(t)
}

// ============================================================
// TEST: CreateSuperAdmin
// ============================================================

func TestCreateSuperAdmin_Success(t *testing.T) {
	cont, servMock, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	servMock.On("CreateSuperAdmin", user.CreateSuperAdminRequest{
		Username: "superadmin",
		Email:    "admin@mail.com",
		Gender:   "Male",
		Password: "password123",
	}).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/create-superadmin", makeJSON(map[string]string{
		"username": "superadmin",
		"email":    "admin@mail.com",
		"gender":   "Male",
		"password": "password123",
	}))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseResponse(w)
	assert.Equal(t, true, body["success"])
	servMock.AssertExpectations(t)
}

func TestCreateSuperAdmin_ShouldFail_InvalidBody(t *testing.T) {
	cont, _, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/create-superadmin", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateSuperAdmin_ShouldFail_ServiceError(t *testing.T) {
	cont, servMock, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	servMock.On("CreateSuperAdmin", mock.AnythingOfType("user.CreateSuperAdminRequest")).
		Return(errors.New("super admin already exists"))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/create-superadmin", makeJSON(map[string]string{
		"username": "superadmin",
		"email":    "admin@mail.com",
		"gender":   "Male",
		"password": "password123",
	}))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	body := parseResponse(w)
	assert.Equal(t, false, body["success"])
	servMock.AssertExpectations(t)
}

// ============================================================
// TEST: RefreshToken
// ============================================================

func TestRefreshToken_Success(t *testing.T) {
	cont, servMock, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	userID := uuid.New()
	token := generateMiddlewareJWT(userID.String())

	mockTokenValid(repoMock, userID)
	newToken := "new-access-token"
	servMock.On("RefreshToken", token).Return(&newToken, nil)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/refresh-token", nil)
	withBearer(req, token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseResponse(w)
	assert.Equal(t, true, body["success"])
	servMock.AssertExpectations(t)
	repoMock.AssertExpectations(t)
}

func TestRefreshToken_ShouldFail_NoToken(t *testing.T) {
	cont, _, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/refresh-token", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRefreshToken_ShouldFail_ServiceError(t *testing.T) {
	cont, servMock, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	userID := uuid.New()
	token := generateMiddlewareJWT(userID.String())

	mockTokenValid(repoMock, userID)
	servMock.On("RefreshToken", token).Return(nil, errors.New("refresh token not found"))

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/auth/refresh-token", nil)
	withBearer(req, token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	body := parseResponse(w)
	assert.Equal(t, false, body["success"])
	servMock.AssertExpectations(t)
	repoMock.AssertExpectations(t)
}

// ============================================================
// TEST: CreateUser
// ============================================================

func TestCreateUser_Success(t *testing.T) {
	cont, servMock, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	userID := uuid.New()
	token := generateMiddlewareJWT(userID.String())

	mockTokenValid(repoMock, userID)
	servMock.On("CreateUser", token, mock.AnythingOfType("user.CreateUserRequest")).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", makeJSON(map[string]string{
		"username": "newuser",
		"name":     "New User",
		"email":    "new@mail.com",
		"gender":   "Male",
		"password": "password123",
		"role_id":  uuid.New().String(),
	}))
	req.Header.Set("Content-Type", "application/json")
	withBearer(req, token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseResponse(w)
	assert.Equal(t, true, body["success"])
	servMock.AssertExpectations(t)
	repoMock.AssertExpectations(t)
}

func TestCreateUser_ShouldFail_NoToken(t *testing.T) {
	cont, _, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", makeJSON(map[string]string{
		"username": "newuser",
		"name":     "New User",
		"email":    "new@mail.com",
		"gender":   "Male",
		"password": "password123",
		"role_id":  uuid.New().String(),
	}))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateUser_ShouldFail_ServiceError(t *testing.T) {
	cont, servMock, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	userID := uuid.New()
	token := generateMiddlewareJWT(userID.String())

	mockTokenValid(repoMock, userID)
	servMock.On("CreateUser", token, mock.AnythingOfType("user.CreateUserRequest")).
		Return(errors.New("user already exists"))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", makeJSON(map[string]string{
		"username": "newuser",
		"name":     "New User",
		"email":    "new@mail.com",
		"gender":   "Male",
		"password": "password123",
		"role_id":  uuid.New().String(),
	}))
	req.Header.Set("Content-Type", "application/json")
	withBearer(req, token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusConflict, w.Code) // 409
	body := parseResponse(w)
	assert.Equal(t, false, body["success"])
	servMock.AssertExpectations(t)
	repoMock.AssertExpectations(t)
}

// ============================================================
// TEST: UpdateProfileClient
// ============================================================

func TestUpdateProfileClient_Success(t *testing.T) {
	cont, servMock, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	userID := uuid.New()
	token := generateMiddlewareJWT(userID.String())

	mockTokenValid(repoMock, userID)
	servMock.On("UpdateProfileClient", token, userID, mock.AnythingOfType("user.UpdateProfileRequest")).Return(nil)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+userID.String()+"/client", makeJSON(map[string]string{
		"name":   "Updated Name",
		"gender": "Male",
	}))
	req.Header.Set("Content-Type", "application/json")
	withBearer(req, token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseResponse(w)
	assert.Equal(t, true, body["success"])
	servMock.AssertExpectations(t)
	repoMock.AssertExpectations(t)
}

func TestUpdateProfileClient_ShouldFail_InvalidUUID(t *testing.T) {
	cont, _, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	userID := uuid.New()
	token := generateMiddlewareJWT(userID.String())

	mockTokenValid(repoMock, userID)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/invalid-uuid/client", makeJSON(map[string]string{
		"name": "Updated Name",
	}))
	req.Header.Set("Content-Type", "application/json")
	withBearer(req, token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateProfileClient_ShouldFail_ServiceError(t *testing.T) {
	cont, servMock, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	userID := uuid.New()
	token := generateMiddlewareJWT(userID.String())

	mockTokenValid(repoMock, userID)
	servMock.On("UpdateProfileClient", token, userID, mock.AnythingOfType("user.UpdateProfileRequest")).
		Return(errors.New("user not found"))

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+userID.String()+"/client", makeJSON(map[string]string{
		"name": "Updated Name",
	}))
	req.Header.Set("Content-Type", "application/json")
	withBearer(req, token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	body := parseResponse(w)
	assert.Equal(t, false, body["success"])
	servMock.AssertExpectations(t)
	repoMock.AssertExpectations(t)
}

// ============================================================
// TEST: UpdateProfileNonClient
// ============================================================

func TestUpdateProfileNonClient_Success(t *testing.T) {
	cont, servMock, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	userID := uuid.New()
	token := generateMiddlewareJWT(userID.String())

	mockTokenValid(repoMock, userID)
	servMock.On("UpdateProfileNonClient", token, userID, mock.AnythingOfType("user.UpdateProfileNonClientRequest")).Return(nil)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+userID.String()+"/non-client", makeJSON(map[string]string{
		"name":   "Updated Name",
		"gender": "Male",
	}))
	req.Header.Set("Content-Type", "application/json")
	withBearer(req, token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseResponse(w)
	assert.Equal(t, true, body["success"])
	servMock.AssertExpectations(t)
	repoMock.AssertExpectations(t)
}

func TestUpdateProfileNonClient_ShouldFail_InvalidUUID(t *testing.T) {
	cont, _, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	userID := uuid.New()
	token := generateMiddlewareJWT(userID.String())

	mockTokenValid(repoMock, userID)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/invalid-uuid/non-client", makeJSON(map[string]string{
		"name": "Updated Name",
	}))
	req.Header.Set("Content-Type", "application/json")
	withBearer(req, token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateProfileNonClient_ShouldFail_ServiceError(t *testing.T) {
	cont, servMock, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	userID := uuid.New()
	token := generateMiddlewareJWT(userID.String())

	mockTokenValid(repoMock, userID)
	servMock.On("UpdateProfileNonClient", token, userID, mock.AnythingOfType("user.UpdateProfileNonClientRequest")).
		Return(errors.New("user not found"))

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+userID.String()+"/non-client", makeJSON(map[string]string{
		"name": "Updated Name",
	}))
	req.Header.Set("Content-Type", "application/json")
	withBearer(req, token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	body := parseResponse(w)
	assert.Equal(t, false, body["success"])
	servMock.AssertExpectations(t)
	repoMock.AssertExpectations(t)
}

// ============================================================
// TEST: EditUserData
// ============================================================

func TestEditUserData_Success(t *testing.T) {
	cont, servMock, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	userID := uuid.New()
	token := generateMiddlewareJWT(userID.String())

	mockTokenValid(repoMock, userID)
	servMock.On("EditUserData", token, userID, mock.AnythingOfType("user.EditUserDataRequest")).Return(nil)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+userID.String()+"/other", makeJSON(map[string]string{
		"name":   "Updated Name",
		"gender": "Male",
	}))
	req.Header.Set("Content-Type", "application/json")
	withBearer(req, token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseResponse(w)
	assert.Equal(t, true, body["success"])
	servMock.AssertExpectations(t)
	repoMock.AssertExpectations(t)
}

func TestEditUserData_ShouldFail_InvalidUUID(t *testing.T) {
	cont, _, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	userID := uuid.New()
	token := generateMiddlewareJWT(userID.String())

	mockTokenValid(repoMock, userID)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/invalid-uuid/other", makeJSON(map[string]string{
		"name": "Updated Name",
	}))
	req.Header.Set("Content-Type", "application/json")
	withBearer(req, token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEditUserData_ShouldFail_ServiceError(t *testing.T) {
	cont, servMock, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	userID := uuid.New()
	token := generateMiddlewareJWT(userID.String())

	mockTokenValid(repoMock, userID)
	servMock.On("EditUserData", token, userID, mock.AnythingOfType("user.EditUserDataRequest")).
		Return(errors.New("user not found"))

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/"+userID.String()+"/other", makeJSON(map[string]string{
		"name": "Updated Name",
	}))
	req.Header.Set("Content-Type", "application/json")
	withBearer(req, token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	body := parseResponse(w)
	assert.Equal(t, false, body["success"])
	servMock.AssertExpectations(t)
	repoMock.AssertExpectations(t)
}

// ============================================================
// TEST: GetByUserId
// ============================================================

func TestGetByUserId_Success(t *testing.T) {
	cont, servMock, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	userID := uuid.New()
	expected := &res.SingleResponse{UserID: userID.String(), Username: "testuser"}
	servMock.On("GetByUserId", userID).Return(expected, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID.String(), nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseResponse(w)
	assert.Equal(t, true, body["success"])
	servMock.AssertExpectations(t)
	repoMock.AssertExpectations(t)
}

func TestGetByUserId_ShouldFail_InvalidUUID(t *testing.T) {
	cont, _, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/invalid-uuid", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetByUserId_ShouldFail_ServiceError(t *testing.T) {
	cont, servMock, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	userID := uuid.New()
	servMock.On("GetByUserId", userID).Return(nil, errors.New("user not found"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID.String(), nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	body := parseResponse(w)
	assert.Equal(t, false, body["success"])
	servMock.AssertExpectations(t)
	repoMock.AssertExpectations(t)
}

// ============================================================
// TEST: GetUsers
// ============================================================

func TestGetUsers_Success(t *testing.T) {
	cont, servMock, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	servMock.On("GetUsers", domains.Pagination{Page: 1, Limit: 10}).
		Return(pagination.Response{Total: 2, Page: 1, Limit: 10}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users?page=1&limit=10", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseResponse(w)
	assert.Equal(t, true, body["success"])
	servMock.AssertExpectations(t)
	repoMock.AssertExpectations(t)
}

func TestGetUsers_ShouldFail_ServiceError(t *testing.T) {
	cont, servMock, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	servMock.On("GetUsers", domains.Pagination{Page: 1, Limit: 10}).
		Return(pagination.Response{}, errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users?page=1&limit=10", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	body := parseResponse(w)
	assert.Equal(t, false, body["success"])
	servMock.AssertExpectations(t)
	repoMock.AssertExpectations(t)
}

// ============================================================
// TEST: FilterUsers
// ============================================================

func TestFilterUsers_Success(t *testing.T) {
	cont, servMock, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	servMock.On("FilterUsers", "Test", "", "SuperAdmin", domains.Pagination{Page: 1, Limit: 10}).
		Return(pagination.Response{Total: 1, Page: 1, Limit: 10}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/filter?name=Test&role=SuperAdmin&page=1&limit=10", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseResponse(w)
	assert.Equal(t, true, body["success"])
	servMock.AssertExpectations(t)
	repoMock.AssertExpectations(t)
}

func TestFilterUsers_ShouldFail_ServiceError(t *testing.T) {
	cont, servMock, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	servMock.On("FilterUsers", "", "", "", domains.Pagination{Page: 1, Limit: 10}).
		Return(pagination.Response{}, errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/filter?page=1&limit=10", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	body := parseResponse(w)
	assert.Equal(t, false, body["success"])
	servMock.AssertExpectations(t)
	repoMock.AssertExpectations(t)
}

// ============================================================
// TEST: UpdateUserStatusActive
// ============================================================

func TestUpdateUserStatusActive_Success(t *testing.T) {
	cont, servMock, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	userID := uuid.New()
	token := generateMiddlewareJWT(userID.String())

	mockTokenValid(repoMock, userID)
	servMock.On("UpdateUserStatusActive", token, userID).Return(nil)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/"+userID.String()+"/status", nil)
	withBearer(req, token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseResponse(w)
	assert.Equal(t, true, body["success"])
	servMock.AssertExpectations(t)
	repoMock.AssertExpectations(t)
}

func TestUpdateUserStatusActive_ShouldFail_InvalidUUID(t *testing.T) {
	cont, _, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	userID := uuid.New()
	token := generateMiddlewareJWT(userID.String())

	mockTokenValid(repoMock, userID)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/invalid-uuid/status", nil)
	withBearer(req, token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateUserStatusActive_ShouldFail_ServiceError(t *testing.T) {
	cont, servMock, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	userID := uuid.New()
	token := generateMiddlewareJWT(userID.String())

	mockTokenValid(repoMock, userID)
	servMock.On("UpdateUserStatusActive", token, userID).Return(errors.New("user not found"))

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/users/"+userID.String()+"/status", nil)
	withBearer(req, token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	body := parseResponse(w)
	assert.Equal(t, false, body["success"])
	servMock.AssertExpectations(t)
	repoMock.AssertExpectations(t)
}

// ============================================================
// TEST: DeleteByUserId
// ============================================================

func TestDeleteByUserId_Success(t *testing.T) {
	cont, servMock, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	userID := uuid.New()
	token := generateMiddlewareJWT(userID.String())

	mockTokenValid(repoMock, userID)
	servMock.On("DeleteByUserId", token, userID).Return(nil)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+userID.String(), nil)
	withBearer(req, token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	body := parseResponse(w)
	assert.Equal(t, true, body["success"])
	servMock.AssertExpectations(t)
	repoMock.AssertExpectations(t)
}

func TestDeleteByUserId_ShouldFail_InvalidUUID(t *testing.T) {
	cont, _, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	userID := uuid.New()
	token := generateMiddlewareJWT(userID.String())

	mockTokenValid(repoMock, userID)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/invalid-uuid", nil)
	withBearer(req, token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteByUserId_ShouldFail_ServiceError(t *testing.T) {
	cont, servMock, repoMock := setupCont()
	r := setupRouter(cont, repoMock)

	userID := uuid.New()
	token := generateMiddlewareJWT(userID.String())

	mockTokenValid(repoMock, userID)
	servMock.On("DeleteByUserId", token, userID).Return(errors.New("user not found"))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+userID.String(), nil)
	withBearer(req, token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	body := parseResponse(w)
	assert.Equal(t, false, body["success"])
	servMock.AssertExpectations(t)
	repoMock.AssertExpectations(t)
}
