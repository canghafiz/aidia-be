package impl

import (
	"backend/helpers"
	"backend/models/domains"
	"backend/models/repositories"
	"backend/models/requests/user"
	"backend/models/responses/pagination"
	res "backend/models/responses/user"
	"fmt"
	"log"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UsersServImpl struct {
	Db        *gorm.DB
	Validator *validator.Validate
	UserRepo  repositories.UsersRepo
	JwtKey    string
}

func NewUsersServImpl(db *gorm.DB, validator *validator.Validate, userRepo repositories.UsersRepo, jwtKey string) *UsersServImpl {
	return &UsersServImpl{Db: db, Validator: validator, UserRepo: userRepo, JwtKey: jwtKey}
}

func (serv *UsersServImpl) Login(request user.LoginRequest) (*res.LoginResponse, error) {
	errValidator := helpers.ErrValidator(request, serv.Validator)
	if errValidator != nil {
		return nil, errValidator
	}

	// Find user by username or email
	findUser, err := serv.UserRepo.FindByUsernameOrEmail(serv.Db, request.UsernameOrEmail)
	if err != nil {
		log.Printf("[UserRepo.FindByUsernameOrEmail] error: %v", err)
		return nil, fmt.Errorf("failed to login, please try again later")
	}
	if findUser == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Get user role
	role, errRole := serv.UserRepo.GetUserRole(serv.Db, findUser.UserID)
	if errRole != nil {
		log.Printf("[UserRepo.GetUserRole] error: %v", errRole)
		return nil, fmt.Errorf("failed to login, please try again later")
	}

	// Check status
	if !findUser.IsActive {
		return nil, fmt.Errorf("user is not active")
	}

	// Check Password
	valid, errPw := serv.UserRepo.CheckPasswordValid(serv.Db, request.UsernameOrEmail, request.Password)
	if errPw != nil {
		log.Printf("[UserRepo.CheckPasswordValid] error: %v", errPw)
		return nil, fmt.Errorf("failed to login, please try again later")
	}
	if !valid {
		return nil, fmt.Errorf("password invalid")
	}

	// Generate access token (JWT)
	accessToken, errJwt := helpers.GenerateJWT(serv.JwtKey, helpers.TokenDuration, res.ToResponse(*findUser, role))
	if errJwt != nil {
		return nil, errJwt
	}

	return res.ToLoginResponse(accessToken, role, *findUser), nil
}

func (serv *UsersServImpl) ChangePw(accessToken string, request user.ChangePwRequest) error {
	errValidator := helpers.ErrValidator(request, serv.Validator)
	if errValidator != nil {
		return errValidator
	}

	data, err := helpers.DecodeJWT(accessToken, serv.JwtKey)
	if err != nil {
		return err
	}

	userId, ok := data["user_id"].(string)
	if !ok {
		return fmt.Errorf("invalid token payload")
	}

	userUUID, errUuid := uuid.Parse(userId)
	if errUuid != nil {
		return fmt.Errorf("invalid user_id format")
	}

	// Validate current password
	username, _ := data["username"].(string)
	valid, errPw := serv.UserRepo.CheckPasswordValid(serv.Db, username, request.CurrentPassword)
	if errPw != nil {
		log.Printf("[UserRepo.CheckPasswordValid] error: %v", errPw)
		return fmt.Errorf("failed to change pw, please try again later")
	}
	if !valid {
		return fmt.Errorf("current password is invalid")
	}

	model := user.ChangePwToDomain(request)
	model.UserID = userUUID

	errRepo := serv.UserRepo.ChangePassword(serv.Db, model)
	if errRepo != nil {
		log.Printf("[UserRepo.ChangePassword] error: %v", errRepo)
		return fmt.Errorf("failed to change pw, please try again later")
	}

	return nil
}

func (serv *UsersServImpl) Me(accessToken string) (*res.Response, error) {
	data, err := helpers.DecodeJWT(accessToken, serv.JwtKey)
	if err != nil {
		return nil, err
	}

	// Preload tenant + business profile
	findUser, errFind := serv.UserRepo.FindByUsernameOrEmail(
		serv.Db,
		data["username"].(string),
		"Tenant.BusinessProfile",
	)
	if errFind != nil || findUser == nil {
		return nil, fmt.Errorf("user not found")
	}

	role, _ := serv.UserRepo.GetUserRole(serv.Db, findUser.UserID)

	result := res.ToResponse(*findUser, role)
	return &result, nil
}

func (serv *UsersServImpl) CheckSuperAdminExist() (bool, error) {
	result, err := serv.UserRepo.CheckSuperAdminExist(serv.Db)
	if err != nil {
		log.Printf("[UserRepo.CheckSuperAdminExist] error: %v", err)
		return false, fmt.Errorf("failed to check super admin exist, please try again later")
	}

	return result, nil
}

func (serv *UsersServImpl) CreateSuperAdmin(request user.CreateSuperAdminRequest) error {
	if err := helpers.ErrValidator(request, serv.Validator); err != nil {
		return err
	}

	exists, err := serv.UserRepo.CheckSuperAdminExist(serv.Db)
	if err != nil {
		log.Printf("[UserRepo.CheckSuperAdminExist] error: %v", err)
		return fmt.Errorf("failed to create super admin, please try again later")
	}
	if exists {
		return fmt.Errorf("super admin already exists")
	}

	return serv.Db.Transaction(func(tx *gorm.DB) error {
		newUser := user.CreateSuperAdminRequestToUser(request)
		if err := serv.UserRepo.Create(tx, newUser); err != nil {
			return err
		}

		created, err := serv.UserRepo.FindByUsernameOrEmail(tx, newUser.Email)
		if err != nil || created == nil {
			log.Printf("[UserRepo.FindByUsernameOrEmail] error: %v", err)
			return fmt.Errorf("failed to create super admin, please try again later")
		}

		return serv.UserRepo.AssignRole(tx, created.UserID, "SuperAdmin")
	})
}

func (serv *UsersServImpl) CreateUser(accessToken string, request user.CreateUserRequest) error {
	// ── Phase 1: Semua validasi sebelum apapun ──
	role, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"SuperAdmin", "Admin"})
	if err != nil || !ok {
		return err
	}

	if err := helpers.ErrValidator(request, serv.Validator); err != nil {
		return err
	}

	if existing, _ := serv.UserRepo.FindByUsernameOrEmail(serv.Db, request.Username); existing != nil {
		return fmt.Errorf("user already exists")
	}

	if existing, _ := serv.UserRepo.FindByUsernameOrEmail(serv.Db, request.Email); existing != nil {
		return fmt.Errorf("user already exists")
	}

	model := user.CreateUserRequestToDomain(request)
	model.IsActive = *role == "SuperAdmin"

	var roleData domains.Roles
	if len(model.UserRoles) > 0 {
		if err := serv.Db.Where("id = ?", model.UserRoles[0].RoleID).First(&roleData).Error; err != nil {
			return fmt.Errorf("role not found")
		}
	}

	isClient := roleData.Name == "Client"

	// Normalize schema — lowercase, spasi dan strip jadi underscore
	normalizedSchema := helpers.NormalizeSchema(model.Username)

	dropSchema := func() {
		if isClient {
			if err := helpers.DropTenantSchema(serv.Db, normalizedSchema); err != nil {
				log.Printf("[CreateUser] DropTenantSchema cleanup error: %v", err)
			}
		}
	}

	// ── Phase 2: DDL dulu (CreateTenantSchema) ──
	if isClient {
		if err := helpers.CreateTenantSchema(serv.Db, normalizedSchema); err != nil {
			log.Printf("[CreateUser] CreateTenantSchema error: %v", err)
			return fmt.Errorf("failed to create tenant schema")
		}
	}

	// ── Phase 3: DML dalam transaction ──
	tx := serv.Db.Begin()
	if tx.Error != nil {
		dropSchema()
		return fmt.Errorf("failed to start transaction")
	}

	rollback := func() {
		tx.Rollback()
		dropSchema()
	}

	defer func() {
		if r := recover(); r != nil {
			rollback()
		}
	}()

	if err := serv.UserRepo.Create(tx, model); err != nil {
		rollback()
		log.Printf("[CreateUser] Create error: %v", err)
		if isDuplicateError(err) {
			return fmt.Errorf("user already exists")
		}
		return fmt.Errorf("failed to create user")
	}

	saved, err := serv.UserRepo.FindByUsernameOrEmail(tx, model.Username)
	if err != nil || saved == nil {
		rollback()
		return fmt.Errorf("failed to retrieve created user")
	}

	if err := serv.UserRepo.AssignRole(tx, saved.UserID, roleData.Name); err != nil {
		rollback()
		log.Printf("[CreateUser] AssignRole error: %v", err)
		return fmt.Errorf("failed to assign role")
	}

	if isClient {
		// Simpan normalized schema ke tenant_schema field
		saved.TenantSchema = &normalizedSchema
		if err := serv.UserRepo.UpdateTenantSchema(tx, *saved); err != nil {
			rollback()
			log.Printf("[CreateUser] UpdateTenantSchema error: %v", err)
			return fmt.Errorf("failed to update tenant schema")
		}

		tenant := domains.Tenant{UserID: saved.UserID, Role: "owner", IsActive: true}
		if err := tx.Create(&tenant).Error; err != nil {
			rollback()
			log.Printf("[CreateUser] Create tenant error: %v", err)
			return fmt.Errorf("failed to create tenant")
		}

		bp := domains.BusinessProfile{TenantId: tenant.TenantID}
		if model.Tenant != nil && model.Tenant.BusinessProfile != nil {
			bp.BusinessName = model.Tenant.BusinessProfile.BusinessName
			bp.Phone = model.Tenant.BusinessProfile.Phone
			bp.Address = model.Tenant.BusinessProfile.Address
		}
		if err := tx.Create(&bp).Error; err != nil {
			rollback()
			log.Printf("[CreateUser] Create business_profile error: %v", err)
			return fmt.Errorf("failed to create business profile")
		}
	}

	if *role == "Admin" {
		action := "Not Approved"
		if err := serv.UserRepo.CreateApprovalLogs(tx, domains.TenantApprovalLogs{
			UserID: saved.UserID,
			Action: &action,
		}); err != nil {
			rollback()
			log.Printf("[CreateUser] CreateApprovalLogs error: %v", err)
			return fmt.Errorf("failed to create approval logs")
		}
	}

	if err := tx.Commit().Error; err != nil {
		rollback()
		return fmt.Errorf("failed to commit transaction")
	}

	return nil
}

func (serv *UsersServImpl) UpdateProfileClient(accessToken string, userID uuid.UUID, request user.UpdateProfileRequest) error {
	// Check role
	_, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"Client"})
	if err != nil || !ok {
		return err
	}

	errValidator := helpers.ErrValidator(request, serv.Validator)
	if errValidator != nil {
		return errValidator
	}

	// Get existing user to fill TenantID
	existing, err := serv.UserRepo.GetByUserId(serv.Db, userID)
	if err != nil || existing == nil {
		return fmt.Errorf("user not found")
	}

	model := user.UpdateProfileRequestToDomain(request)
	model.UserID = userID

	// Set TenantID from existing user
	if model.Tenant != nil && existing.Tenant != nil {
		model.Tenant.TenantID = existing.Tenant.TenantID
	}

	if err := serv.UserRepo.Update(serv.Db, model); err != nil {
		log.Printf("[UpdateProfileClient] Update error: %v", err)
		return fmt.Errorf("failed to update user")
	}

	return nil
}

func (serv *UsersServImpl) UpdateProfileNonClient(accessToken string, userID uuid.UUID, request user.UpdateProfileNonClientRequest) error {
	// Check role
	_, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"SuperAdmin", "Admin"})
	if err != nil || !ok {
		return err
	}

	errValidator := helpers.ErrValidator(request, serv.Validator)
	if errValidator != nil {
		return errValidator
	}

	// Get existing user to fill TenantID
	existing, err := serv.UserRepo.GetByUserId(serv.Db, userID)
	if err != nil || existing == nil {
		return fmt.Errorf("user not found")
	}

	model := user.UpdateProfileNonClientRequestToDomain(request)
	model.UserID = userID

	if err := serv.UserRepo.Update(serv.Db, model); err != nil {
		log.Printf("[UpdateProfileNonClient] Update error: %v", err)
		return fmt.Errorf("failed to update user")
	}

	return nil
}

func (serv *UsersServImpl) EditUserData(accessToken string, userID uuid.UUID, request user.EditUserDataRequest) error {
	// Check role
	_, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"SuperAdmin", "Admin"})
	if err != nil || !ok {
		return err
	}

	errValidator := helpers.ErrValidator(request, serv.Validator)
	if errValidator != nil {
		return errValidator
	}

	// Get existing user to fill TenantID
	existing, err := serv.UserRepo.GetByUserId(serv.Db, userID)
	if err != nil || existing == nil {
		return fmt.Errorf("user not found")
	}

	model := user.EditUserDataRequestToDomain(request)
	model.UserID = userID

	if err := serv.UserRepo.Update(serv.Db, model); err != nil {
		log.Printf("[EditUserData] Update error: %v", err)
		return fmt.Errorf("failed to update user")
	}

	return nil
}

func (serv *UsersServImpl) GetByUserId(userID uuid.UUID) (*res.SingleResponse, error) {
	found, err := serv.UserRepo.GetByUserId(serv.Db, userID)
	if err != nil || found == nil {
		return nil, fmt.Errorf("user not found")
	}

	role, _ := serv.UserRepo.GetUserRole(serv.Db, userID)
	result := res.ToSingleResponse(*found, role)
	return &result, nil
}

func (serv *UsersServImpl) GetUsers(accessToken string, pg domains.Pagination) (pagination.Response, error) {
	// Get user id
	userId, err := helpers.GetUserIdFromToken(accessToken, serv.JwtKey)
	if err != nil {
		return pagination.Response{}, err
	}

	users, total, err := serv.UserRepo.GetUsers(serv.Db, *userId, pg)
	if err != nil {
		log.Printf("[GetUsers] error: %v", err)
		return pagination.Response{}, fmt.Errorf("failed to get users")
	}

	return pagination.ToResponse(
		res.ToResponsesList(users),
		total,
		pg.Page,
		pg.Limit,
	), nil
}

func (serv *UsersServImpl) FilterUsers(name, email, role string, pg domains.Pagination) (pagination.Response, error) {
	users, total, err := serv.UserRepo.FilterUsers(serv.Db, name, email, role, pg)
	if err != nil {
		log.Printf("[FilterUsers] error: %v", err)
		return pagination.Response{}, fmt.Errorf("failed to filter users")
	}

	return pagination.ToResponse(
		res.ToResponsesList(users),
		total,
		pg.Page,
		pg.Limit,
	), nil
}

func (serv *UsersServImpl) DeleteByUserId(accessToken string, userID uuid.UUID) error {
	// Check role
	_, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"SuperAdmin", "Admin"})
	if err != nil || !ok {
		return err
	}

	existing, err := serv.UserRepo.GetByUserId(serv.Db, userID)
	if err != nil || existing == nil {
		return fmt.Errorf("user not found")
	}

	// Drop tenant schema
	if existing.TenantSchema != nil && *existing.TenantSchema != "" {
		if err := helpers.DropTenantSchema(serv.Db, *existing.TenantSchema); err != nil {
			log.Printf("[DeleteByUserId] DropTenantSchema error: %v", err)
			return fmt.Errorf("failed to delete tenant schema")
		}
	}

	if err := serv.UserRepo.DeleteByUserId(serv.Db, userID); err != nil {
		log.Printf("[DeleteByUserId] error: %v", err)
		return fmt.Errorf("failed to delete user")
	}

	return nil
}

func isDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique")
}
