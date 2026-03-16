package repositories_test

import (
	"backend/helpers"
	"backend/models/domains"
	"backend/models/repositories/impl"
	testhelper "backend/testing"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// ============================================================
// HELPERS
// ============================================================

func cleanupUsers(db *gorm.DB, userID uuid.UUID) {
	db.Unscoped().Where("user_id = ?", userID).Delete(&domains.Users{})
}

func cleanupUserRoles(db *gorm.DB, userID uuid.UUID) {
	db.Unscoped().Where("user_id = ?", userID).Delete(&domains.UserRoles{})
}

func dummyUser() domains.Users {
	return domains.Users{
		Username: "testuser_" + uuid.New().String()[:8],
		Name:     "Test User",
		Email:    "test_" + uuid.New().String()[:8] + "@mail.com",
		Password: "password123",
		IsActive: true,
	}
}

func createAndGetUser(t *testing.T, db *gorm.DB, repo *impl.UserRepoImpl) domains.Users {
	t.Helper()
	user := dummyUser()

	err := repo.Create(db, user)
	assert.NoError(t, err)

	var saved domains.Users
	db.Where("username = ?", user.Username).First(&saved)
	return saved
}

func createAndGetUserWithToken(t *testing.T, db *gorm.DB, repo *impl.UserRepoImpl) domains.Users {
	t.Helper()
	saved := createAndGetUser(t, db, repo)

	_, err := repo.GenerateToken(db, saved.UserID, 24*time.Hour)
	assert.NoError(t, err)

	db.Where("user_id = ?", saved.UserID).First(&saved)
	return saved
}

func createUserWithPlainPassword(t *testing.T, db *gorm.DB, repo *impl.UserRepoImpl) (domains.Users, string) {
	t.Helper()
	user := dummyUser()
	plainPassword := user.Password

	err := repo.Create(db, user)
	assert.NoError(t, err)

	var saved domains.Users
	db.Where("username = ?", user.Username).First(&saved)
	return saved, plainPassword
}

func createUserWithRole(t *testing.T, db *gorm.DB, repo *impl.UserRepoImpl, roleName string) domains.Users {
	t.Helper()
	saved := createAndGetUser(t, db, repo)

	err := repo.AssignRole(db, saved.UserID, roleName)
	assert.NoError(t, err)

	return saved
}

// ============================================================
// TEST: Create
// ============================================================

func TestCreate_Success(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	user := dummyUser()

	err := repo.Create(db, user)
	assert.NoError(t, err)

	var saved domains.Users
	db.Where("username = ?", user.Username).First(&saved)
	assert.Equal(t, user.Username, saved.Username)
	assert.NotEqual(t, user.Password, saved.Password) // password must be hashed

	cleanupUsers(db, saved.UserID)
}

func TestCreate_DuplicateEmail_ShouldFail(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	user := dummyUser()

	err := repo.Create(db, user)
	assert.NoError(t, err)

	err = repo.Create(db, user)
	assert.Error(t, err)

	var saved domains.Users
	db.Where("username = ?", user.Username).First(&saved)
	cleanupUsers(db, saved.UserID)
}

// ============================================================
// TEST: AssignRole
// ============================================================

func TestAssignRole_Success(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved := createAndGetUser(t, db, repo)

	err := repo.AssignRole(db, saved.UserID, "SuperAdmin")
	assert.NoError(t, err)

	var userRole domains.UserRoles
	db.Where("user_id = ?", saved.UserID).First(&userRole)
	assert.Equal(t, saved.UserID, userRole.UserID)

	cleanupUserRoles(db, saved.UserID)
	cleanupUsers(db, saved.UserID)
}

func TestAssignRole_InvalidRole_ShouldFail(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved := createAndGetUser(t, db, repo)

	err := repo.AssignRole(db, saved.UserID, "InvalidRole")
	assert.Error(t, err)

	cleanupUsers(db, saved.UserID)
}

// ============================================================
// TEST: GetUserRole
// ============================================================

func TestGetUserRole_Success(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved := createUserWithRole(t, db, repo, "SuperAdmin")

	roleName, err := repo.GetUserRole(db, saved.UserID)
	assert.NoError(t, err)
	assert.Equal(t, "SuperAdmin", roleName)

	cleanupUserRoles(db, saved.UserID)
	cleanupUsers(db, saved.UserID)
}

func TestGetUserRole_ShouldFail_WhenNoRole(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved := createAndGetUser(t, db, repo)

	roleName, err := repo.GetUserRole(db, saved.UserID)
	assert.Error(t, err)
	assert.Empty(t, roleName)

	cleanupUsers(db, saved.UserID)
}

// ============================================================
// TEST: GenerateToken
// ============================================================

func TestGenerateToken_Success(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved := createAndGetUser(t, db, repo)

	token, err := repo.GenerateToken(db, saved.UserID, 24*time.Hour)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	var updated domains.Users
	db.Where("user_id = ?", saved.UserID).First(&updated)
	assert.NotNil(t, updated.Token)
	assert.NotNil(t, updated.TokenExpire)
	assert.Equal(t, token, *updated.Token)
	assert.True(t, updated.TokenExpire.UTC().After(time.Now().UTC()))

	cleanupUsers(db, saved.UserID)
}

func TestGenerateToken_ShouldAlwaysRotate(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved := createAndGetUser(t, db, repo)

	firstToken, err := repo.GenerateToken(db, saved.UserID, 24*time.Hour)
	assert.NoError(t, err)

	secondToken, err := repo.GenerateToken(db, saved.UserID, 24*time.Hour)
	assert.NoError(t, err)

	assert.NotEqual(t, firstToken, secondToken, "token should always rotate on GenerateToken")

	cleanupUsers(db, saved.UserID)
}

// ============================================================
// TEST: FindByToken
// ============================================================

func TestFindByToken_Success(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved := createAndGetUserWithToken(t, db, repo)

	result, err := repo.FindByToken(db, *saved.Token)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, saved.UserID, result.UserID)

	cleanupUsers(db, saved.UserID)
}

func TestFindByToken_NotFound_ShouldReturnNil(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()

	result, err := repo.FindByToken(db, "non-existent-token")
	assert.NoError(t, err)
	assert.Nil(t, result)
}

// ============================================================
// TEST: RefreshToken
// ============================================================

func TestRefreshToken_Success(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved := createAndGetUserWithToken(t, db, repo)
	oldToken := *saved.Token

	result, err := repo.RefreshToken(db, oldToken, 24*time.Hour)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEqual(t, oldToken, *result.Token, "token should be rotated")
	assert.True(t, result.TokenExpire.UTC().After(time.Now().UTC()))

	cleanupUsers(db, saved.UserID)
}

func TestRefreshToken_ShouldFail_WhenTokenNotFound(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()

	result, err := repo.RefreshToken(db, "invalid-token-string", 24*time.Hour)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.EqualError(t, err, "refresh token not found")
}

func TestRefreshToken_ShouldFail_WhenTokenExpired(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved := createAndGetUser(t, db, repo)

	expiredTime := time.Now().UTC().Add(-1 * time.Hour)
	token := uuid.New().String()
	db.Model(&domains.Users{}).Where("user_id = ?", saved.UserID).Updates(map[string]interface{}{
		"token":        token,
		"token_expire": expiredTime,
	})

	result, err := repo.RefreshToken(db, token, 24*time.Hour)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.EqualError(t, err, "refresh token expired")

	cleanupUsers(db, saved.UserID)
}

// ============================================================
// TEST: CheckTokenValid
// ============================================================

func TestCheckTokenValid_ShouldReturnTrue(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved := createAndGetUserWithToken(t, db, repo)

	isValid := repo.CheckTokenValid(db, saved)
	assert.True(t, isValid)

	cleanupUsers(db, saved.UserID)
}

func TestCheckTokenValid_ShouldReturnFalse_WhenExpired(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved := createAndGetUser(t, db, repo)

	expiredTime := time.Now().UTC().Add(-1 * time.Hour)
	token := uuid.New().String()
	db.Model(&domains.Users{}).Where("user_id = ?", saved.UserID).Updates(map[string]interface{}{
		"token":        token,
		"token_expire": expiredTime,
	})

	saved.Token = &token
	saved.TokenExpire = &expiredTime

	isValid := repo.CheckTokenValid(db, saved)
	assert.False(t, isValid)

	cleanupUsers(db, saved.UserID)
}

func TestCheckTokenValid_ShouldReturnFalse_WhenTokenMismatch(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved := createAndGetUserWithToken(t, db, repo)

	wrongToken := uuid.New().String()
	saved.Token = &wrongToken

	isValid := repo.CheckTokenValid(db, saved)
	assert.False(t, isValid)

	cleanupUsers(db, saved.UserID)
}

// ============================================================
// TEST: ChangePassword
// ============================================================

func TestChangePassword_Success(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved := createAndGetUser(t, db, repo)
	oldPassword := saved.Password

	saved.Password = "newpassword456"
	err := repo.ChangePassword(db, saved)
	assert.NoError(t, err)

	var updated domains.Users
	db.Where("user_id = ?", saved.UserID).First(&updated)
	assert.NotEqual(t, oldPassword, updated.Password, "password must be changed")

	cleanupUsers(db, saved.UserID)
}

// ============================================================
// TEST: FindByUsernameOrEmail
// ============================================================

func TestFindByUsernameOrEmail_ByEmail_Success(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved := createAndGetUser(t, db, repo)

	result, err := repo.FindByUsernameOrEmail(db, saved.Email)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, saved.Email, result.Email)

	cleanupUsers(db, saved.UserID)
}

func TestFindByUsernameOrEmail_ByUsername_Success(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved := createAndGetUser(t, db, repo)

	result, err := repo.FindByUsernameOrEmail(db, saved.Username)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, saved.Username, result.Username)

	cleanupUsers(db, saved.UserID)
}

func TestFindByUsernameOrEmail_NotFound_ShouldReturnNil(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()

	result, err := repo.FindByUsernameOrEmail(db, "notexist@mail.com")
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestFindByUsernameOrEmail_EmptyInput_ShouldReturnError(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()

	result, err := repo.FindByUsernameOrEmail(db, "")
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestFindByUsernameOrEmail_WithRolePreload_Success(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved := createUserWithRole(t, db, repo, "SuperAdmin")

	result, err := repo.FindByUsernameOrEmail(db, saved.Username, "UserRoles.Role")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.UserRoles)
	assert.Equal(t, "SuperAdmin", result.UserRoles[0].Role.Name)

	cleanupUserRoles(db, saved.UserID)
	cleanupUsers(db, saved.UserID)
}

func TestFindByUsernameOrEmail_WithTenantPreload_Success(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved := createAndGetUser(t, db, repo)

	result, err := repo.FindByUsernameOrEmail(db, saved.Email, "Tenant")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, saved.Email, result.Email)

	cleanupUsers(db, saved.UserID)
}

// ============================================================
// TEST: CheckPasswordValid
// ============================================================

func TestCheckPasswordValid_ByEmail_ShouldReturnTrue(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved, plainPassword := createUserWithPlainPassword(t, db, repo)

	isValid, err := repo.CheckPasswordValid(db, saved.Email, plainPassword)
	assert.NoError(t, err)
	assert.True(t, isValid)

	cleanupUsers(db, saved.UserID)
}

func TestCheckPasswordValid_ByUsername_ShouldReturnTrue(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved, plainPassword := createUserWithPlainPassword(t, db, repo)

	isValid, err := repo.CheckPasswordValid(db, saved.Username, plainPassword)
	assert.NoError(t, err)
	assert.True(t, isValid)

	cleanupUsers(db, saved.UserID)
}

func TestCheckPasswordValid_WrongPassword_ShouldReturnFalse(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved := createAndGetUser(t, db, repo)

	isValid, err := repo.CheckPasswordValid(db, saved.Username, "wrongpassword")
	assert.NoError(t, err)
	assert.False(t, isValid)

	cleanupUsers(db, saved.UserID)
}

func TestCheckPasswordValid_UserNotFound_ShouldReturnFalse(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()

	isValid, err := repo.CheckPasswordValid(db, "ghost@mail.com", "password123")
	assert.NoError(t, err)
	assert.False(t, isValid)
}

// ============================================================
// TEST: CheckSuperAdminExist
// ============================================================

func TestCheckSuperAdminExist_ShouldReturnFalse_WhenNone(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()

	db.Exec("DELETE FROM user_roles ur USING roles r WHERE ur.role_id = r.id AND r.name = 'SuperAdmin'")

	exists, err := repo.CheckSuperAdminExist(db)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestCheckSuperAdminExist_ShouldReturnTrue_WhenExists(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved := createUserWithRole(t, db, repo, "SuperAdmin")

	exists, err := repo.CheckSuperAdminExist(db)
	assert.NoError(t, err)
	assert.True(t, exists)

	cleanupUserRoles(db, saved.UserID)
	cleanupUsers(db, saved.UserID)
}

// ============================================================
// TEST: ResetToken
// ============================================================

func TestResetToken_Success(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved := createAndGetUserWithToken(t, db, repo)

	assert.NotNil(t, saved.Token)
	assert.NotNil(t, saved.TokenExpire)

	err := repo.ResetToken(db, saved)
	assert.NoError(t, err)

	var updated domains.Users
	db.Where("user_id = ?", saved.UserID).First(&updated)
	assert.Nil(t, updated.Token)
	assert.Nil(t, updated.TokenExpire)

	cleanupUsers(db, saved.UserID)
}

func TestResetToken_ShouldSucceedEvenIfTokenAlreadyNil(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved := createAndGetUser(t, db, repo)

	err := repo.ResetToken(db, saved)
	assert.NoError(t, err)

	var updated domains.Users
	db.Where("user_id = ?", saved.UserID).First(&updated)
	assert.Nil(t, updated.Token)
	assert.Nil(t, updated.TokenExpire)

	cleanupUsers(db, saved.UserID)
}

// ============================================================
// TEST: GetByUserId
// ============================================================

func TestGetByUserId_Success(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved := createAndGetUser(t, db, repo)

	result, err := repo.GetByUserId(db, saved.UserID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, saved.UserID, result.UserID)

	cleanupUsers(db, saved.UserID)
}

func TestGetByUserId_ShouldFail_NotFound(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()

	result, err := repo.GetByUserId(db, uuid.New())
	assert.Error(t, err)
	assert.Nil(t, result)
}

// ============================================================
// TEST: UpdateTenantSchema
// ============================================================

func TestUpdateTenantSchema_Success(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved := createAndGetUser(t, db, repo)

	err := repo.UpdateTenantSchema(db, saved)
	assert.NoError(t, err)

	var updated domains.Users
	db.Where("user_id = ?", saved.UserID).First(&updated)
	assert.NotNil(t, updated.TenantSchema)
	assert.Equal(t, saved.Username, *updated.TenantSchema)

	cleanupUsers(db, saved.UserID)
}

func TestUpdateTenantSchema_ShouldFail_UserNotFound(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()

	ghost := domains.Users{
		UserID:   uuid.New(),
		Username: "ghostuser",
	}

	err := repo.UpdateTenantSchema(db, ghost)
	assert.NoError(t, err)
}

// ============================================================
// TEST: Update
// ============================================================

func TestUpdate_Success_UserFields(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved := createAndGetUser(t, db, repo)

	saved.Name = "Updated Name"
	saved.Gender = "Male"
	err := repo.Update(db, saved)
	assert.NoError(t, err)

	var updated domains.Users
	db.Where("user_id = ?", saved.UserID).First(&updated)
	assert.Equal(t, "Updated Name", updated.Name)
	assert.Equal(t, "Male", updated.Gender)

	cleanupUsers(db, saved.UserID)
}

func TestUpdate_Success_Role(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved := createUserWithRole(t, db, repo, "SuperAdmin")

	var newRole domains.Roles
	db.Where("name = ?", "Admin").First(&newRole)

	saved.UserRoles = []domains.UserRoles{{UserID: saved.UserID, RoleID: newRole.ID}}
	err := repo.Update(db, saved)
	assert.NoError(t, err)

	roleName, err := repo.GetUserRole(db, saved.UserID)
	assert.NoError(t, err)
	assert.Equal(t, "Admin", roleName)

	cleanupUserRoles(db, saved.UserID)
	cleanupUsers(db, saved.UserID)
}

func TestUpdate_Success_Password(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved := createAndGetUser(t, db, repo)

	saved.Password = "newpassword123"
	err := repo.Update(db, saved)
	assert.NoError(t, err)

	var updated domains.Users
	db.Where("user_id = ?", saved.UserID).First(&updated)
	assert.True(t, helpers.CheckPassword(updated.Password, "newpassword123"))

	cleanupUsers(db, saved.UserID)
}

func TestUpdate_ShouldNotChangePassword_WhenEmpty(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved, plainPassword := createUserWithPlainPassword(t, db, repo)
	oldHashedPassword := saved.Password

	saved.Password = ""
	saved.Name = "Updated Name"
	err := repo.Update(db, saved)
	assert.NoError(t, err)

	var updated domains.Users
	db.Where("user_id = ?", saved.UserID).First(&updated)
	assert.Equal(t, oldHashedPassword, updated.Password)
	assert.True(t, helpers.CheckPassword(updated.Password, plainPassword))

	cleanupUsers(db, saved.UserID)
}

// ============================================================
// TEST: GetUsers
// ============================================================

func TestGetUsers_Success(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved := createAndGetUser(t, db, repo)

	users, total, err := repo.GetUsers(db, domains.Pagination{Page: 1, Limit: 10})
	assert.NoError(t, err)
	assert.NotEmpty(t, users)
	assert.Greater(t, total, 0)

	cleanupUsers(db, saved.UserID)
}

func TestGetUsers_Pagination(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved1 := createAndGetUser(t, db, repo)
	saved2 := createAndGetUser(t, db, repo)

	users, total, err := repo.GetUsers(db, domains.Pagination{Page: 1, Limit: 1})
	assert.NoError(t, err)
	assert.Len(t, users, 1)
	assert.GreaterOrEqual(t, total, 2)

	cleanupUsers(db, saved1.UserID)
	cleanupUsers(db, saved2.UserID)
}

// ============================================================
// TEST: FilterUsers
// ============================================================

func TestFilterUsers_ByName_Success(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved := createAndGetUser(t, db, repo)

	users, total, err := repo.FilterUsers(db, "Test User", "", "", domains.Pagination{Page: 1, Limit: 10})
	assert.NoError(t, err)
	assert.NotEmpty(t, users)
	assert.Greater(t, total, 0)

	cleanupUsers(db, saved.UserID)
}

func TestFilterUsers_ByRole_Success(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved := createUserWithRole(t, db, repo, "SuperAdmin")

	users, total, err := repo.FilterUsers(db, "", "", "SuperAdmin", domains.Pagination{Page: 1, Limit: 10})
	assert.NoError(t, err)
	assert.NotEmpty(t, users)
	assert.Greater(t, total, 0)

	cleanupUserRoles(db, saved.UserID)
	cleanupUsers(db, saved.UserID)
}

func TestFilterUsers_NoMatch_ShouldReturnEmpty(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()

	users, total, err := repo.FilterUsers(db, "xyznotexist", "", "", domains.Pagination{Page: 1, Limit: 10})
	assert.NoError(t, err)
	assert.Empty(t, users)
	assert.Equal(t, 0, total)
}

// ============================================================
// TEST: UpdateUserStatusActive
// ============================================================

func TestUpdateUserStatusActive_Deactivate(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved := createAndGetUser(t, db, repo)

	saved.IsActive = false
	err := repo.UpdateUserStatusActive(db, saved)
	assert.NoError(t, err)

	var updated domains.Users
	db.Where("user_id = ?", saved.UserID).First(&updated)
	assert.False(t, updated.IsActive)

	cleanupUsers(db, saved.UserID)
}

func TestUpdateUserStatusActive_Activate(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved := createAndGetUser(t, db, repo)

	db.Model(&domains.Users{}).Where("user_id = ?", saved.UserID).Update("is_active", false)

	saved.IsActive = true
	err := repo.UpdateUserStatusActive(db, saved)
	assert.NoError(t, err)

	var updated domains.Users
	db.Where("user_id = ?", saved.UserID).First(&updated)
	assert.True(t, updated.IsActive)

	cleanupUsers(db, saved.UserID)
}

// ============================================================
// TEST: DeleteByUserId
// ============================================================

func TestDeleteByUserId_Success(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()
	saved := createAndGetUser(t, db, repo)

	err := repo.DeleteByUserId(db, saved.UserID)
	assert.NoError(t, err)

	var deleted domains.Users
	result := db.Where("user_id = ?", saved.UserID).First(&deleted)
	assert.Error(t, result.Error)
}

func TestDeleteByUserId_ShouldFail_NotFound(t *testing.T) {
	db := testhelper.SetUpDbTest(t)
	repo := impl.NewUserRepoImpl()

	err := repo.DeleteByUserId(db, uuid.New())
	assert.NoError(t, err)
}
