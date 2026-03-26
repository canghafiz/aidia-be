package impl

import (
	"backend/helpers"
	"backend/models/domains"
	"errors"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserRepoImpl struct {
}

func NewUserRepoImpl() *UserRepoImpl {
	return &UserRepoImpl{}
}

func (repo *UserRepoImpl) Create(db *gorm.DB, user domains.Users) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashed)

	return db.Omit("UserRoles", "Tenant").Create(&user).Error
}

func (repo *UserRepoImpl) AssignRole(db *gorm.DB, userID uuid.UUID, roleName string) error {
	var role domains.Roles
	err := db.Where("name = ?", roleName).First(&role).Error
	if err != nil {
		return err
	}

	userRole := domains.UserRoles{
		UserID: userID,
		RoleID: role.ID,
	}

	return db.Create(&userRole).Error
}

func (repo *UserRepoImpl) ChangePassword(db *gorm.DB, user domains.Users) error {
	pw := helpers.HashedPassword(user.Password)

	return db.Model(&domains.Users{}).
		Where("user_id = ?", user.UserID).
		Update("password", pw).Error
}

func (repo *UserRepoImpl) GetUserRole(db *gorm.DB, userID uuid.UUID) (string, error) {
	var role domains.Roles
	err := db.Joins("JOIN user_roles ur ON ur.role_id = roles.id").
		Where("ur.user_id = ?", userID).
		First(&role).Error
	if err != nil {
		return "", err
	}
	return role.Name, nil
}

func (repo *UserRepoImpl) FindByToken(db *gorm.DB, token string) (*domains.Users, error) {
	var user domains.Users
	err := db.Where("token = ?", token).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (repo *UserRepoImpl) FindByUsernameOrEmail(db *gorm.DB, usernameOrEmail string, preloads ...string) (*domains.Users, error) {
	if usernameOrEmail == "" {
		return nil, errors.New("username or email is required")
	}

	var user domains.Users

	query := db.Model(&domains.Users{})

	// Apply preloads dynamically
	for _, preload := range preloads {
		query = query.Preload(preload)
	}

	// If contains "@" treat as email, otherwise as username
	if strings.Contains(usernameOrEmail, "@") {
		query = query.Where("email = ?", usernameOrEmail)
	} else {
		query = query.Where("username = ?", usernameOrEmail)
	}

	err := query.First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (repo *UserRepoImpl) CheckPasswordValid(db *gorm.DB, usernameOrEmail, password string) (bool, error) {
	result, err := repo.FindByUsernameOrEmail(db, usernameOrEmail)
	if err != nil {
		return false, err
	}
	if result == nil {
		return false, nil
	}

	return helpers.CheckPassword(result.Password, password), nil
}

func (repo *UserRepoImpl) CheckSuperAdminExist(db *gorm.DB) (bool, error) {
	var count int64
	err := db.Model(&domains.Users{}).
		Joins("JOIN user_roles ur ON ur.user_id = users.user_id").
		Joins("JOIN roles r ON r.id = ur.role_id").
		Where("r.name = ?", "SuperAdmin").
		Where("users.is_active = ?", true).
		Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (repo *UserRepoImpl) GetByUserId(db *gorm.DB, userID uuid.UUID) (*domains.Users, error) {
	var user domains.Users
	err := db.
		Preload("Tenant.BusinessProfile").
		Preload("UserRoles.Role").
		Where("user_id = ?", userID).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (repo *UserRepoImpl) UpdateTenantSchema(db *gorm.DB, user domains.Users) error {
	err := db.Model(&domains.Users{}).
		Where("user_id = ?", user.UserID).
		Update("tenant_schema", user.TenantSchema).Error
	if err != nil {
		return err
	}

	return nil
}

func (repo *UserRepoImpl) Update(db *gorm.DB, user domains.Users) error {
	// ── Update users table ──
	userUpdates := map[string]interface{}{}
	if user.Username != "" {
		userUpdates["username"] = user.Username
	}
	if user.Name != "" {
		userUpdates["name"] = user.Name
	}
	if user.Email != "" {
		userUpdates["email"] = user.Email
	}
	if user.Gender != "" {
		userUpdates["gender"] = user.Gender
	}
	if user.Password != "" {
		userUpdates["password"] = helpers.HashedPassword(user.Password)
	}
	userUpdates["is_active"] = user.IsActive

	if len(userUpdates) > 0 {
		if err := db.Model(&domains.Users{}).
			Where("user_id = ?", user.UserID).
			Updates(userUpdates).Error; err != nil {
			return err
		}
	}

	// ── Update role (delete old + insert new) ──
	if len(user.UserRoles) > 0 && user.UserRoles[0].RoleID != (uuid.UUID{}) {
		if err := db.Where("user_id = ?", user.UserID).
			Delete(&domains.UserRoles{}).Error; err != nil {
			return err
		}
		if err := db.Create(&domains.UserRoles{
			UserID: user.UserID,
			RoleID: user.UserRoles[0].RoleID,
		}).Error; err != nil {
			return err
		}
	}

	// ── Update business_profile ──
	if user.Tenant != nil && user.Tenant.BusinessProfile != nil {
		bp := user.Tenant.BusinessProfile
		bpUpdates := map[string]interface{}{}
		if bp.BusinessName != "" {
			bpUpdates["business_name"] = bp.BusinessName
		}
		if bp.Phone != "" {
			bpUpdates["phone"] = bp.Phone
		}
		if bp.Address != "" {
			bpUpdates["address"] = bp.Address
		}
		if len(bpUpdates) > 0 {
			if err := db.Model(&domains.BusinessProfile{}).
				Where("tenant_id = ?", user.Tenant.TenantID).
				Updates(bpUpdates).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

func (repo *UserRepoImpl) GetUsers(db *gorm.DB, exceptId string, pagination domains.Pagination) ([]domains.Users, int, error) {
	var users []domains.Users
	var total int64

	query := db.Model(&domains.Users{}).
		Joins("JOIN user_roles ur ON ur.user_id = users.user_id").
		Joins("JOIN roles r ON r.id = ur.role_id").
		Where("r.name != ?", "SuperAdmin")

	if exceptId != "" {
		query = query.Where("users.user_id != ?", exceptId)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.
		Preload("Tenant.BusinessProfile").
		Preload("UserRoles.Role").
		Limit(pagination.Limit).
		Offset(pagination.Offset()).
		Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, int(total), nil
}

func (repo *UserRepoImpl) GetUsersRoleClient(db *gorm.DB, pagination domains.Pagination) ([]domains.Users, int, error) {
	var users []domains.Users
	var total int64

	query := db.Model(&domains.Users{}).
		Joins("JOIN user_roles ur ON ur.user_id = users.user_id").
		Joins("JOIN roles r ON r.id = ur.role_id").
		Where("r.name == ?", "Client")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.
		Preload("Tenant.BusinessProfile").
		Preload("UserRoles.Role").
		Limit(pagination.Limit).
		Offset(pagination.Offset()).
		Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, int(total), nil
}

func (repo *UserRepoImpl) FilterUsers(db *gorm.DB, name, email, role string, pagination domains.Pagination) ([]domains.Users, int, error) {
	var users []domains.Users
	var total int64

	query := db.Model(&domains.Users{})

	if name != "" {
		query = query.Where("name ILIKE ?", "%"+name+"%")
	}
	if email != "" {
		query = query.Where("email ILIKE ?", "%"+email+"%")
	}
	if role != "" {
		query = query.
			Joins("JOIN user_roles ur ON ur.user_id = users.user_id").
			Joins("JOIN roles r ON r.id = ur.role_id").
			Where("r.name = ?", role)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.
		Preload("Tenant.BusinessProfile").
		Preload("UserRoles.Role").
		Limit(pagination.Limit).
		Offset(pagination.Offset()).
		Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, int(total), nil
}

func (repo *UserRepoImpl) UpdateUserStatusActive(db *gorm.DB, users domains.Users) error {
	return db.Model(&domains.Users{}).
		Where("user_id = ?", users.UserID).
		Update("is_active", users.IsActive).Error
}

func (repo *UserRepoImpl) CreateApprovalLogs(db *gorm.DB, model domains.TenantApprovalLogs) error {
	err := db.Create(&model).Error
	if err != nil {
		return err
	}

	return nil
}

func (repo *UserRepoImpl) DeleteByUserId(db *gorm.DB, userID uuid.UUID) error {
	return db.Where("user_id = ?", userID).
		Delete(&domains.Users{}).Error
}
