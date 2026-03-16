package impl

import (
	"backend/helpers"
	"backend/models/domains"
	"backend/models/repositories"
	"backend/models/responses/approve"
	"backend/models/responses/pagination"
	"fmt"
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ApprovalServImpl struct {
	Db           *gorm.DB
	ApprovalRepo repositories.ApprovalLogsRepo
	UserRepo     repositories.UsersRepo
	JwtKey       string
}

func NewApprovalServImpl(db *gorm.DB, jwtKey string, approvalRepo repositories.ApprovalLogsRepo, userRepo repositories.UsersRepo) *ApprovalServImpl {
	return &ApprovalServImpl{Db: db, JwtKey: jwtKey, ApprovalRepo: approvalRepo, UserRepo: userRepo}
}

func (serv *ApprovalServImpl) Approve(accessToken string, approveId uuid.UUID) error {
	_, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"SuperAdmin"})
	if err != nil || !ok {
		return err
	}

	actionId, errActionId := helpers.GetUserIdFromToken(accessToken, serv.JwtKey)
	if errActionId != nil {
		return err
	}

	approvalLog, errApproval := serv.ApprovalRepo.GetByID(serv.Db, approveId)
	if errApproval != nil {
		log.Printf("[ApprovalRepo].GetByID error: %v", errApproval)
		return fmt.Errorf("approval log not found")
	}

	tx := serv.Db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to start transaction")
	}

	// Activate user
	if err := serv.UserRepo.UpdateUserStatusActive(tx, domains.Users{
		UserID:   approvalLog.UserID,
		IsActive: true,
	}); err != nil {
		tx.Rollback()
		log.Printf("[UserRepo].UpdateUserStatusActive error: %v", err)
		return fmt.Errorf("failed to approve user")
	}

	// Update approval log action = "Approved"
	actionIdParse, _ := uuid.Parse(*actionId)
	action := "Approved"
	if err := serv.ApprovalRepo.Approve(tx, domains.TenantApprovalLogs{
		ID:       approveId,
		Action:   &action,
		ActionBy: &actionIdParse,
	}); err != nil {
		tx.Rollback()
		log.Printf("[ApprovalRepo].Approve error: %v", err)
		return fmt.Errorf("failed to update approval log")
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to commit transaction")
	}

	return nil
}

func (serv *ApprovalServImpl) GetAll(accessToken string, pg domains.Pagination) (*pagination.Response, error) {
	_, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"SuperAdmin"})
	if err != nil || !ok {
		return nil, err
	}

	logs, total, errResult := serv.ApprovalRepo.GetAll(serv.Db, pg)
	if errResult != nil {
		log.Printf("[ApprovalRepo].GetAll error: %v", errResult)
		return nil, fmt.Errorf("failed to get approval logs")
	}

	responses := approve.ToResponses(logs)
	result := pagination.ToResponse(responses, total, pg.Page, pg.Limit)
	return &result, nil
}

func (serv *ApprovalServImpl) Delete(accessToken string, approveId uuid.UUID) error {
	_, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"SuperAdmin"})
	if err != nil || !ok {
		return err
	}

	if err := serv.ApprovalRepo.Delete(serv.Db, approveId); err != nil {
		log.Printf("[ApprovalRepo].Delete error: %v", err)
		return fmt.Errorf("failed to delete approval log")
	}

	return nil
}
