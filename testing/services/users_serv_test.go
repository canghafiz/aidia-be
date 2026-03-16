package services_test

import (
	"backend/models/domains"
	"backend/models/requests/user"
	"backend/models/responses/pagination"
	implServ "backend/models/services/impl"
	"backend/testing/mocks"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ============================================================
// HELPERS
// ============================================================

func setupServ() (*implServ.UsersServImpl, *mocks.UsersRepoMock, sqlmock.Sqlmock) {
	repoMock := new(mocks.UsersRepoMock)
	validate := validator.New()

	db, sqlMock, _ := sqlmock.New()
	dialector := postgres.New(postgres.Config{Conn: db})
	gormDB, _ := gorm.Open(dialector, &gorm.Config{})

	serv := implServ.NewUsersServImpl(gormDB, validate, repoMock, "test-secret-key")
	return serv, repoMock, sqlMock
}

func dummyUserDomain() *domains.Users {
	id := uuid.New()
	return &domains.Users{
		UserID:   id,
		Username: "testuser",
		Name:     "Test User",
		Email:    "test@mail.com",
		Gender:   "Male",
		IsActive: true,
		Password: "hashedpassword",
	}
}

func dummyUserWithToken() *domains.Users {
	u := dummyUserDomain()
	token := uuid.New().String()
	expire := time.Now().UTC().Add(24 * time.Hour)
	u.Token = &token
	u.TokenExpire = &expire
	return u
}

func generateTestJWT(jwtKey, userID, username string) (string, error) {
	claims := jwt.MapClaims{
		"data": map[string]interface{}{
			"user_id":  userID,
			"username": username,
			"name":     "Test User",
			"email":    "test@mail.com",
			"gender":   "Male",
			"role":     "SuperAdmin",
		},
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtKey))
}

func generateTokenWithRole(jwtKey, userID, username, role string) string {
	claims := jwt.MapClaims{
		"data": map[string]interface{}{
			"user_id":  userID,
			"username": username,
			"role":     role,
		},
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t, _ := token.SignedString([]byte(jwtKey))
	return t
}

// ============================================================
// TEST: Login
// ============================================================

func TestLogin_Success(t *testing.T) {
	serv, repoMock, _ := setupServ()
	u := dummyUserDomain()

	repoMock.On("FindByUsernameOrEmail", mock.Anything, "test@mail.com").Return(u, nil)
	repoMock.On("GetUserRole", mock.Anything, u.UserID).Return("SuperAdmin", nil)
	repoMock.On("CheckPasswordValid", mock.Anything, "test@mail.com", "password123").Return(true, nil)
	repoMock.On("GenerateToken", mock.Anything, u.UserID, mock.AnythingOfType("time.Duration")).Return("refresh-token-uuid", nil)

	req := user.LoginRequest{UsernameOrEmail: "test@mail.com", Password: "password123"}
	result, err := serv.Login(req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.AccessToken)
	repoMock.AssertExpectations(t)
}

func TestLogin_ShouldFail_ValidationError(t *testing.T) {
	serv, _, _ := setupServ()

	req := user.LoginRequest{UsernameOrEmail: "", Password: ""}
	result, err := serv.Login(req)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestLogin_ShouldFail_UserNotFound(t *testing.T) {
	serv, repoMock, _ := setupServ()

	repoMock.On("FindByUsernameOrEmail", mock.Anything, "ghost@mail.com").Return(nil, nil)

	req := user.LoginRequest{UsernameOrEmail: "ghost@mail.com", Password: "password123"}
	result, err := serv.Login(req)

	assert.Error(t, err)
	assert.Nil(t, result)
	repoMock.AssertExpectations(t)
}

func TestLogin_ShouldFail_UserNotActive(t *testing.T) {
	serv, repoMock, _ := setupServ()
	u := dummyUserDomain()
	u.IsActive = false

	repoMock.On("FindByUsernameOrEmail", mock.Anything, "test@mail.com").Return(u, nil)
	repoMock.On("GetUserRole", mock.Anything, u.UserID).Return("Client", nil)

	req := user.LoginRequest{UsernameOrEmail: "test@mail.com", Password: "password123"}
	result, err := serv.Login(req)

	assert.Error(t, err)
	assert.EqualError(t, err, "user is not active")
	assert.Nil(t, result)
	repoMock.AssertExpectations(t)
}

func TestLogin_ShouldFail_InvalidPassword(t *testing.T) {
	serv, repoMock, _ := setupServ()
	u := dummyUserDomain()

	repoMock.On("FindByUsernameOrEmail", mock.Anything, "test@mail.com").Return(u, nil)
	repoMock.On("GetUserRole", mock.Anything, u.UserID).Return("SuperAdmin", nil)
	repoMock.On("CheckPasswordValid", mock.Anything, "test@mail.com", "wrongpassword").Return(false, nil)

	req := user.LoginRequest{UsernameOrEmail: "test@mail.com", Password: "wrongpassword"}
	result, err := serv.Login(req)

	assert.Error(t, err)
	assert.EqualError(t, err, "password invalid")
	assert.Nil(t, result)
	repoMock.AssertExpectations(t)
}

// ============================================================
// TEST: Logout
// ============================================================

func TestLogout_Success(t *testing.T) {
	serv, repoMock, _ := setupServ()
	u := dummyUserDomain()

	accessToken, err := generateTestJWT(serv.JwtKey, u.UserID.String(), u.Username)
	assert.NoError(t, err)

	repoMock.On("ResetToken", mock.Anything, mock.MatchedBy(func(u domains.Users) bool {
		return u.UserID != uuid.Nil
	})).Return(nil)

	err = serv.Logout(accessToken)
	assert.NoError(t, err)
	repoMock.AssertExpectations(t)
}

func TestLogout_ShouldFail_InvalidToken(t *testing.T) {
	serv, _, _ := setupServ()

	err := serv.Logout("invalid.token.here")
	assert.Error(t, err)
}

// ============================================================
// TEST: Me
// ============================================================

func TestMe_Success(t *testing.T) {
	serv, repoMock, _ := setupServ()
	u := dummyUserDomain()

	accessToken, err := generateTestJWT(serv.JwtKey, u.UserID.String(), u.Username)
	assert.NoError(t, err)

	repoMock.On("FindByUsernameOrEmail", mock.Anything, u.Username).Return(u, nil)
	repoMock.On("GetUserRole", mock.Anything, u.UserID).Return("SuperAdmin", nil)

	result, err := serv.Me(accessToken)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, u.Email, result.Email)
	repoMock.AssertExpectations(t)
}

func TestMe_ShouldFail_InvalidToken(t *testing.T) {
	serv, _, _ := setupServ()

	result, err := serv.Me("invalid.token.here")
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestMe_ShouldFail_UserNotFound(t *testing.T) {
	serv, repoMock, _ := setupServ()
	u := dummyUserDomain()

	accessToken, err := generateTestJWT(serv.JwtKey, u.UserID.String(), u.Username)
	assert.NoError(t, err)

	repoMock.On("FindByUsernameOrEmail", mock.Anything, u.Username).Return(nil, nil)

	result, err := serv.Me(accessToken)
	assert.Error(t, err)
	assert.Nil(t, result)
	repoMock.AssertExpectations(t)
}

// ============================================================
// TEST: CheckSuperAdminExist
// ============================================================

func TestCheckSuperAdminExist_True(t *testing.T) {
	serv, repoMock, _ := setupServ()

	repoMock.On("CheckSuperAdminExist", mock.Anything).Return(true, nil)

	exists, err := serv.CheckSuperAdminExist()
	assert.NoError(t, err)
	assert.True(t, exists)
	repoMock.AssertExpectations(t)
}

func TestCheckSuperAdminExist_False(t *testing.T) {
	serv, repoMock, _ := setupServ()

	repoMock.On("CheckSuperAdminExist", mock.Anything).Return(false, nil)

	exists, err := serv.CheckSuperAdminExist()
	assert.NoError(t, err)
	assert.False(t, exists)
	repoMock.AssertExpectations(t)
}

func TestCheckSuperAdminExist_RepoError(t *testing.T) {
	serv, repoMock, _ := setupServ()

	repoMock.On("CheckSuperAdminExist", mock.Anything).Return(false, errors.New("db error"))

	exists, err := serv.CheckSuperAdminExist()
	assert.Error(t, err)
	assert.False(t, exists)
	repoMock.AssertExpectations(t)
}

// ============================================================
// TEST: CreateSuperAdmin
// ============================================================

func TestCreateSuperAdmin_Success(t *testing.T) {
	serv, repoMock, sqlMock := setupServ()
	u := dummyUserDomain()

	sqlMock.ExpectBegin()
	sqlMock.ExpectCommit()

	repoMock.On("CheckSuperAdminExist", mock.Anything).Return(false, nil)
	repoMock.On("Create", mock.Anything, mock.AnythingOfType("domains.Users")).Return(nil)
	repoMock.On("FindByUsernameOrEmail", mock.Anything, u.Email).Return(u, nil)
	repoMock.On("AssignRole", mock.Anything, u.UserID, "SuperAdmin").Return(nil)

	req := user.CreateSuperAdminRequest{
		Username: "superadmin",
		Email:    u.Email,
		Gender:   "Male",
		Password: "password123",
	}

	err := serv.CreateSuperAdmin(req)
	assert.NoError(t, err)
	repoMock.AssertExpectations(t)
}

func TestCreateSuperAdmin_ShouldFail_ValidationError(t *testing.T) {
	serv, _, _ := setupServ()

	req := user.CreateSuperAdminRequest{
		Username: "",
		Email:    "invalid-email",
		Gender:   "Unknown",
		Password: "123",
	}

	err := serv.CreateSuperAdmin(req)
	assert.Error(t, err)
}

func TestCreateSuperAdmin_ShouldFail_AlreadyExists(t *testing.T) {
	serv, repoMock, _ := setupServ()

	repoMock.On("CheckSuperAdminExist", mock.Anything).Return(true, nil)

	req := user.CreateSuperAdminRequest{
		Username: "superadmin",
		Email:    "admin@mail.com",
		Gender:   "Male",
		Password: "password123",
	}

	err := serv.CreateSuperAdmin(req)
	assert.Error(t, err)
	assert.EqualError(t, err, "super admin already exists")
	repoMock.AssertExpectations(t)
}

// ============================================================
// TEST: RefreshToken
// ============================================================

func TestRefreshToken_Success(t *testing.T) {
	serv, repoMock, _ := setupServ()
	u := dummyUserWithToken()

	repoMock.On("RefreshToken", mock.Anything, *u.Token, mock.AnythingOfType("time.Duration")).Return(u, nil)
	repoMock.On("GetUserRole", mock.Anything, u.UserID).Return("SuperAdmin", nil)

	result, err := serv.RefreshToken(*u.Token)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, *result)
	repoMock.AssertExpectations(t)
}

func TestRefreshToken_ShouldFail_TokenNotFound(t *testing.T) {
	serv, repoMock, _ := setupServ()

	repoMock.On("RefreshToken", mock.Anything, "bad-token", mock.AnythingOfType("time.Duration")).
		Return(nil, errors.New("refresh token not found"))

	result, err := serv.RefreshToken("bad-token")
	assert.Error(t, err)
	assert.Nil(t, result)
	repoMock.AssertExpectations(t)
}

// ============================================================
// TEST: CreateUser
// ============================================================
func TestCreateUser_Success_ClientRole_TenantSchemaCreated(t *testing.T) {
	serv, repoMock, sqlMock := setupServ()
	u := dummyUserDomain()

	accessToken := generateTokenWithRole(serv.JwtKey, u.UserID.String(), u.Username, "SuperAdmin")

	roleID := uuid.New()
	sqlMock.ExpectQuery(`SELECT \* FROM "roles" WHERE id = \$1`).
		WithArgs(roleID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(roleID, "Client"))

	repoMock.On("Create", mock.Anything, mock.AnythingOfType("domains.Users")).Return(nil)
	repoMock.On("FindByUsernameOrEmail", mock.Anything, mock.Anything).Return(u, nil)
	repoMock.On("AssignRole", mock.Anything, u.UserID, "Client").Return(nil)
	repoMock.On("UpdateTenantSchema", mock.Anything, mock.MatchedBy(func(updated domains.Users) bool {
		return updated.TenantSchema != nil && *updated.TenantSchema == updated.Username
	})).Return(nil)

	req := user.CreateUserRequest{
		Username: u.Username,
		Name:     u.Name,
		Email:    u.Email,
		Gender:   u.Gender,
		Password: "password123",
		RoleID:   roleID.String(),
	}

	serv.CreateUser(accessToken, req)

	repoMock.AssertCalled(t, "UpdateTenantSchema", mock.Anything, mock.MatchedBy(func(updated domains.Users) bool {
		return updated.TenantSchema != nil && *updated.TenantSchema == updated.Username
	}))
}

func TestCreateUser_Success_AdminRole_NoTenantSchema(t *testing.T) {
	serv, repoMock, sqlMock := setupServ()
	u := dummyUserDomain()

	accessToken := generateTokenWithRole(serv.JwtKey, u.UserID.String(), u.Username, "SuperAdmin")

	roleID := uuid.New()
	sqlMock.ExpectQuery(`SELECT \* FROM "roles" WHERE id = \$1`).
		WithArgs(roleID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(roleID, "Admin"))

	repoMock.On("Create", mock.Anything, mock.AnythingOfType("domains.Users")).Return(nil)
	repoMock.On("FindByUsernameOrEmail", mock.Anything, mock.Anything).Return(u, nil)
	repoMock.On("AssignRole", mock.Anything, u.UserID, "Admin").Return(nil)

	req := user.CreateUserRequest{
		Username: u.Username,
		Name:     u.Name,
		Email:    u.Email,
		Gender:   u.Gender,
		Password: "password123",
		RoleID:   roleID.String(),
	}

	serv.CreateUser(accessToken, req)

	repoMock.AssertNotCalled(t, "UpdateTenantSchema")
}

func TestCreateUser_Success_CallerIsAdmin(t *testing.T) {
	serv, repoMock, sqlMock := setupServ()
	u := dummyUserDomain()

	accessToken := generateTokenWithRole(serv.JwtKey, u.UserID.String(), u.Username, "Admin")

	roleID := uuid.New()
	sqlMock.ExpectQuery(`SELECT \* FROM "roles" WHERE id = \$1`).
		WithArgs(roleID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(roleID, "Admin"))

	repoMock.On("Create", mock.Anything, mock.AnythingOfType("domains.Users")).Return(nil)
	repoMock.On("FindByUsernameOrEmail", mock.Anything, mock.Anything).Return(u, nil)
	repoMock.On("AssignRole", mock.Anything, u.UserID, "Admin").Return(nil)

	req := user.CreateUserRequest{
		Username: u.Username,
		Name:     u.Name,
		Email:    u.Email,
		Gender:   u.Gender,
		Password: "password123",
		RoleID:   roleID.String(),
	}

	serv.CreateUser(accessToken, req)

	repoMock.AssertCalled(t, "Create", mock.Anything, mock.AnythingOfType("domains.Users"))
}

func TestCreateUser_ShouldFail_InvalidToken(t *testing.T) {
	serv, _, _ := setupServ()

	req := user.CreateUserRequest{
		Username: "newuser",
		Name:     "New User",
		Email:    "new@mail.com",
		Gender:   "Male",
		Password: "password123",
		RoleID:   uuid.New().String(),
	}

	err := serv.CreateUser("invalid.token.here", req)
	assert.Error(t, err)
}

func TestCreateUser_ShouldFail_UnauthorizedRole_Client(t *testing.T) {
	serv, _, _ := setupServ()
	u := dummyUserDomain()

	accessToken := generateTokenWithRole(serv.JwtKey, u.UserID.String(), u.Username, "Client")

	req := user.CreateUserRequest{
		Username: "newuser",
		Name:     "New User",
		Email:    "new@mail.com",
		Gender:   "Male",
		Password: "password123",
		RoleID:   uuid.New().String(),
	}

	err := serv.CreateUser(accessToken, req)
	assert.Error(t, err)
}

func TestCreateUser_ShouldFail_ValidationError(t *testing.T) {
	serv, _, _ := setupServ()
	u := dummyUserDomain()

	accessToken, _ := generateTestJWT(serv.JwtKey, u.UserID.String(), u.Username)

	req := user.CreateUserRequest{
		Username: "",
		Email:    "invalid",
		Gender:   "Unknown",
		Password: "123",
		RoleID:   "not-a-uuid",
	}

	err := serv.CreateUser(accessToken, req)
	assert.Error(t, err)
}

func TestCreateUser_ShouldFail_DuplicateUser(t *testing.T) {
	serv, repoMock, _ := setupServ()
	u := dummyUserDomain()

	accessToken, _ := generateTestJWT(serv.JwtKey, u.UserID.String(), u.Username)

	repoMock.On("Create", mock.Anything, mock.AnythingOfType("domains.Users")).
		Return(errors.New("duplicate key value violates unique constraint"))

	req := user.CreateUserRequest{
		Username: "newuser",
		Name:     "New User",
		Email:    "new@mail.com",
		Gender:   "Male",
		Password: "password123",
		RoleID:   uuid.New().String(),
	}

	err := serv.CreateUser(accessToken, req)
	assert.Error(t, err)
	assert.EqualError(t, err, "user already exists")
	repoMock.AssertExpectations(t)
}

func TestCreateUser_ShouldFail_RepoError(t *testing.T) {
	serv, repoMock, _ := setupServ()
	u := dummyUserDomain()

	accessToken, _ := generateTestJWT(serv.JwtKey, u.UserID.String(), u.Username)

	repoMock.On("Create", mock.Anything, mock.AnythingOfType("domains.Users")).
		Return(errors.New("db error"))

	req := user.CreateUserRequest{
		Username: "newuser",
		Name:     "New User",
		Email:    "new@mail.com",
		Gender:   "Male",
		Password: "password123",
		RoleID:   uuid.New().String(),
	}

	err := serv.CreateUser(accessToken, req)
	assert.Error(t, err)
	assert.EqualError(t, err, "failed to create user")
	repoMock.AssertExpectations(t)
}

// ============================================================
// TEST: EditUserData
// ============================================================

func TestEditUserData_Success(t *testing.T) {
	serv, repoMock, _ := setupServ()
	u := dummyUserDomain()
	u.Tenant = &domains.Tenant{TenantID: uuid.New(), UserID: u.UserID}

	accessToken := generateTokenWithRole(serv.JwtKey, u.UserID.String(), u.Username, "Client")

	repoMock.On("GetByUserId", mock.Anything, u.UserID).Return(u, nil)
	repoMock.On("Update", mock.Anything, mock.AnythingOfType("domains.Users")).Return(nil)

	req := user.UpdateProfileRequest{
		Name:   "Updated Name",
		Gender: "Male",
	}

	err := serv.UpdateProfileClient(accessToken, u.UserID, req)
	assert.NoError(t, err)
	repoMock.AssertExpectations(t)
}

func TestEditUserData_ShouldFail_InvalidToken(t *testing.T) {
	serv, _, _ := setupServ()

	req := user.UpdateProfileRequest{Name: "Updated Name"}
	err := serv.UpdateProfileClient("invalid.token.here", uuid.New(), req)
	assert.Error(t, err)
}

func TestEditUserData_ShouldFail_UserNotFound(t *testing.T) {
	serv, repoMock, _ := setupServ()
	u := dummyUserDomain()

	accessToken := generateTokenWithRole(serv.JwtKey, u.UserID.String(), u.Username, "Client")

	repoMock.On("GetByUserId", mock.Anything, mock.AnythingOfType("uuid.UUID")).
		Return(nil, errors.New("not found"))

	req := user.UpdateProfileRequest{Name: "Updated Name"}
	err := serv.UpdateProfileClient(accessToken, uuid.New(), req)
	assert.Error(t, err)
	assert.EqualError(t, err, "user not found")
	repoMock.AssertExpectations(t)
}

func TestEditUserData_ShouldFail_RepoError(t *testing.T) {
	serv, repoMock, _ := setupServ()
	u := dummyUserDomain()

	accessToken := generateTokenWithRole(serv.JwtKey, u.UserID.String(), u.Username, "Client")

	repoMock.On("GetByUserId", mock.Anything, u.UserID).Return(u, nil)
	repoMock.On("Update", mock.Anything, mock.AnythingOfType("domains.Users")).
		Return(errors.New("db error"))

	req := user.UpdateProfileRequest{Name: "Updated Name"}
	err := serv.UpdateProfileClient(accessToken, u.UserID, req)
	assert.Error(t, err)
	assert.EqualError(t, err, "failed to update user")
	repoMock.AssertExpectations(t)
}

// ============================================================
// TEST: GetByUserId
// ============================================================

func TestGetByUserId_Success(t *testing.T) {
	serv, repoMock, _ := setupServ()
	u := dummyUserDomain()

	repoMock.On("GetByUserId", mock.Anything, u.UserID).Return(u, nil)
	repoMock.On("GetUserRole", mock.Anything, u.UserID).Return("SuperAdmin", nil)

	result, err := serv.GetByUserId(u.UserID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, u.UserID.String(), result.UserID)
	repoMock.AssertExpectations(t)
}

func TestGetByUserId_ShouldFail_NotFound(t *testing.T) {
	serv, repoMock, _ := setupServ()

	repoMock.On("GetByUserId", mock.Anything, mock.AnythingOfType("uuid.UUID")).
		Return(nil, errors.New("not found"))

	result, err := serv.GetByUserId(uuid.New())
	assert.Error(t, err)
	assert.Nil(t, result)
	repoMock.AssertExpectations(t)
}

// ============================================================
// TEST: GetUsers
// ============================================================

func TestGetUsers_Success(t *testing.T) {
	serv, repoMock, _ := setupServ()
	users := []domains.Users{*dummyUserDomain(), *dummyUserDomain()}

	repoMock.On("GetUsers", mock.Anything, domains.Pagination{Page: 1, Limit: 10}).
		Return(users, 2, nil)

	result, err := serv.GetUsers(domains.Pagination{Page: 1, Limit: 10})
	assert.NoError(t, err)
	assert.Equal(t, 2, result.Total)
	repoMock.AssertExpectations(t)
}

func TestGetUsers_ShouldFail_RepoError(t *testing.T) {
	serv, repoMock, _ := setupServ()

	repoMock.On("GetUsers", mock.Anything, domains.Pagination{Page: 1, Limit: 10}).
		Return(nil, 0, errors.New("db error"))

	result, err := serv.GetUsers(domains.Pagination{Page: 1, Limit: 10})
	assert.Error(t, err)
	assert.Equal(t, pagination.Response{}, result)
	repoMock.AssertExpectations(t)
}

// ============================================================
// TEST: FilterUsers
// ============================================================

func TestFilterUsers_Success(t *testing.T) {
	serv, repoMock, _ := setupServ()
	users := []domains.Users{*dummyUserDomain()}

	repoMock.On("FilterUsers", mock.Anything, "Test", "", "", domains.Pagination{Page: 1, Limit: 10}).
		Return(users, 1, nil)

	result, err := serv.FilterUsers("Test", "", "", domains.Pagination{Page: 1, Limit: 10})
	assert.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	repoMock.AssertExpectations(t)
}

func TestFilterUsers_ShouldFail_RepoError(t *testing.T) {
	serv, repoMock, _ := setupServ()

	repoMock.On("FilterUsers", mock.Anything, "", "", "", domains.Pagination{Page: 1, Limit: 10}).
		Return(nil, 0, errors.New("db error"))

	result, err := serv.FilterUsers("", "", "", domains.Pagination{Page: 1, Limit: 10})
	assert.Error(t, err)
	assert.Equal(t, pagination.Response{}, result)
	repoMock.AssertExpectations(t)
}

// ============================================================
// TEST: UpdateUserStatusActive
// ============================================================

func TestUpdateUserStatusActive_Success_Toggle(t *testing.T) {
	serv, repoMock, _ := setupServ()
	u := dummyUserDomain() // IsActive = true

	accessToken, _ := generateTestJWT(serv.JwtKey, u.UserID.String(), u.Username)

	repoMock.On("GetByUserId", mock.Anything, u.UserID).Return(u, nil)
	repoMock.On("UpdateUserStatusActive", mock.Anything, mock.MatchedBy(func(updated domains.Users) bool {
		return updated.UserID == u.UserID && updated.IsActive == false // toggled
	})).Return(nil)

	err := serv.UpdateUserStatusActive(accessToken, u.UserID)
	assert.NoError(t, err)
	repoMock.AssertExpectations(t)
}

func TestUpdateUserStatusActive_ShouldFail_InvalidToken(t *testing.T) {
	serv, _, _ := setupServ()

	err := serv.UpdateUserStatusActive("invalid.token.here", uuid.New())
	assert.Error(t, err)
}

func TestUpdateUserStatusActive_ShouldFail_UnauthorizedRole(t *testing.T) {
	serv, _, _ := setupServ()
	u := dummyUserDomain()

	// Generate token with Admin role (UpdateUserStatusActive only allows SuperAdmin)
	claims := jwt.MapClaims{
		"data": map[string]interface{}{
			"user_id":  u.UserID.String(),
			"username": u.Username,
			"role":     "Admin",
		},
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, _ := token.SignedString([]byte(serv.JwtKey))

	err := serv.UpdateUserStatusActive(accessToken, u.UserID)
	assert.Error(t, err)
}

func TestUpdateUserStatusActive_ShouldFail_UserNotFound(t *testing.T) {
	serv, repoMock, _ := setupServ()
	u := dummyUserDomain()

	accessToken, _ := generateTestJWT(serv.JwtKey, u.UserID.String(), u.Username)

	repoMock.On("GetByUserId", mock.Anything, mock.AnythingOfType("uuid.UUID")).
		Return(nil, errors.New("not found"))

	err := serv.UpdateUserStatusActive(accessToken, uuid.New())
	assert.Error(t, err)
	assert.EqualError(t, err, "user not found")
	repoMock.AssertExpectations(t)
}

func TestUpdateUserStatusActive_ShouldFail_RepoError(t *testing.T) {
	serv, repoMock, _ := setupServ()
	u := dummyUserDomain()

	accessToken, _ := generateTestJWT(serv.JwtKey, u.UserID.String(), u.Username)

	repoMock.On("GetByUserId", mock.Anything, u.UserID).Return(u, nil)
	repoMock.On("UpdateUserStatusActive", mock.Anything, mock.AnythingOfType("domains.Users")).
		Return(errors.New("db error"))

	err := serv.UpdateUserStatusActive(accessToken, u.UserID)
	assert.Error(t, err)
	assert.EqualError(t, err, "failed to update user status")
	repoMock.AssertExpectations(t)
}

// ============================================================
// TEST: DeleteByUserId
// ============================================================

func TestDeleteByUserId_Success(t *testing.T) {
	serv, repoMock, _ := setupServ()
	u := dummyUserDomain()

	accessToken, _ := generateTestJWT(serv.JwtKey, u.UserID.String(), u.Username)

	repoMock.On("GetByUserId", mock.Anything, u.UserID).Return(u, nil)
	repoMock.On("DeleteByUserId", mock.Anything, u.UserID).Return(nil)

	err := serv.DeleteByUserId(accessToken, u.UserID)
	assert.NoError(t, err)
	repoMock.AssertExpectations(t)
}

func TestDeleteByUserId_ShouldFail_InvalidToken(t *testing.T) {
	serv, _, _ := setupServ()

	err := serv.DeleteByUserId("invalid.token.here", uuid.New())
	assert.Error(t, err)
}

func TestDeleteByUserId_ShouldFail_UserNotFound(t *testing.T) {
	serv, repoMock, _ := setupServ()
	u := dummyUserDomain()

	accessToken, _ := generateTestJWT(serv.JwtKey, u.UserID.String(), u.Username)

	repoMock.On("GetByUserId", mock.Anything, mock.AnythingOfType("uuid.UUID")).
		Return(nil, errors.New("not found"))

	err := serv.DeleteByUserId(accessToken, uuid.New())
	assert.Error(t, err)
	assert.EqualError(t, err, "user not found")
	repoMock.AssertExpectations(t)
}

func TestDeleteByUserId_ShouldFail_RepoError(t *testing.T) {
	serv, repoMock, _ := setupServ()
	u := dummyUserDomain()

	accessToken, _ := generateTestJWT(serv.JwtKey, u.UserID.String(), u.Username)

	repoMock.On("GetByUserId", mock.Anything, u.UserID).Return(u, nil)
	repoMock.On("DeleteByUserId", mock.Anything, u.UserID).Return(errors.New("db error"))

	err := serv.DeleteByUserId(accessToken, u.UserID)
	assert.Error(t, err)
	assert.EqualError(t, err, "failed to delete user")
	repoMock.AssertExpectations(t)
}
