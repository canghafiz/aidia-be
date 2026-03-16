package impl

import (
	"backend/helpers"
	"backend/models/repositories"
	subsRes "backend/models/responses/subs"
	"fmt"
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SubsServImpl struct {
	Db              *gorm.DB
	JwtKey          string
	TenantRepo      repositories.TenantRepo
	TenantUsageRepo repositories.TenantUsageRepo
}

func NewSubsServImpl(
	db *gorm.DB,
	jwtKey string,
	tenantRepo repositories.TenantRepo,
	tenantUsageRepo repositories.TenantUsageRepo,
) *SubsServImpl {
	return &SubsServImpl{
		Db:              db,
		JwtKey:          jwtKey,
		TenantRepo:      tenantRepo,
		TenantUsageRepo: tenantUsageRepo,
	}
}

func (serv *SubsServImpl) GetCurrentSubs(accessToken string) (*subsRes.Response, error) {
	_, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"Client"})
	if err != nil || !ok {
		return nil, err
	}

	userIDStr, err := helpers.GetUserIdFromToken(accessToken, serv.JwtKey)
	if err != nil {
		return nil, err
	}
	userID, _ := uuid.Parse(*userIDStr)

	tenant, err := serv.TenantRepo.GetByUserID(serv.Db, userID)
	if err != nil {
		return nil, fmt.Errorf("tenant not found")
	}

	// Ambil semua usage dari plan yang masih aktif
	activeUsages, err := serv.TenantUsageRepo.GetActiveUsageByTenantID(serv.Db, tenant.TenantID)
	if err != nil {
		log.Printf("[SubsServ] GetActiveUsageByTenantID error: %v", err)
		return nil, fmt.Errorf("failed to get active subscriptions")
	}

	// Ambil free usage (tenant_plan_id = NULL)
	freeUsage, err := serv.TenantUsageRepo.GetFreeUsageByTenantID(serv.Db, tenant.TenantID)
	if err != nil {
		log.Printf("[SubsServ] GetFreeUsageByTenantID error: %v", err)
		// Tidak return error — free usage mungkin belum ada
		freeUsage = nil
	}

	response := subsRes.ToSubsResponse(activeUsages, freeUsage)
	return &response, nil
}
