package impl

import (
	"backend/helpers"
	"backend/models/domains"
	"backend/models/repositories"
	reqDelivery "backend/models/requests/delivery_setting"
	resDelivery "backend/models/responses/delivery_setting"
	"fmt"
	"log"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DeliverySettingServImpl struct {
	Db                  *gorm.DB
	Validator           *validator.Validate
	UserRepo            repositories.UsersRepo
	DeliverySettingRepo repositories.DeliverySettingRepo
}

func NewDeliverySettingServImpl(
	db *gorm.DB,
	validator *validator.Validate,
	userRepo repositories.UsersRepo,
	deliverySettingRepo repositories.DeliverySettingRepo,
) *DeliverySettingServImpl {
	return &DeliverySettingServImpl{
		Db:                  db,
		Validator:           validator,
		UserRepo:            userRepo,
		DeliverySettingRepo: deliverySettingRepo,
	}
}

func (serv *DeliverySettingServImpl) getSchema(userID uuid.UUID) (string, error) {
	return helpers.GetSchema(serv.Db, serv.UserRepo, userID)
}

func (serv *DeliverySettingServImpl) checkClientRole(userID uuid.UUID) error {
	role, err := serv.UserRepo.GetUserRole(serv.Db, userID)
	if err != nil {
		return fmt.Errorf("failed to get user role")
	}
	if role != "Client" {
		return fmt.Errorf("access denied")
	}
	return nil
}

func (serv *DeliverySettingServImpl) Create(userID uuid.UUID, request reqDelivery.CreateDeliverySettingRequest) error {
	if err := serv.checkClientRole(userID); err != nil {
		return err
	}

	if err := helpers.ErrValidator(request, serv.Validator); err != nil {
		return err
	}

	schema, err := serv.getSchema(userID)
	if err != nil {
		return err
	}

	settings := reqDelivery.CreateDeliverySettingToDomain(request)
	if err := serv.DeliverySettingRepo.Create(serv.Db, schema, settings); err != nil {
		log.Printf("[DeliverySettingRepo].Create error: %v", err)
		// Check if error is duplicate constraint
		if err.Error() != "" && contains(err.Error(), "duplicate key") {
			return fmt.Errorf("delivery name already exist")
		}
		return fmt.Errorf("failed to create delivery setting")
	}

	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (serv *DeliverySettingServImpl) Update(userID uuid.UUID, subGroupName string, request reqDelivery.UpdateDeliverySettingRequest) error {
	if err := serv.checkClientRole(userID); err != nil {
		return err
	}

	if err := helpers.ErrValidator(request, serv.Validator); err != nil {
		return err
	}

	schema, err := serv.getSchema(userID)
	if err != nil {
		return err
	}

	existing, err := serv.DeliverySettingRepo.GetBySubGroupName(serv.Db, schema, subGroupName)
	if err != nil || len(existing) == 0 {
		return fmt.Errorf("delivery setting not found")
	}

	settings := reqDelivery.UpdateDeliverySettingToDomain(request, subGroupName)
	if err := serv.DeliverySettingRepo.Update(serv.Db, schema, settings); err != nil {
		log.Printf("[DeliverySettingRepo].Update error: %v", err)
		return fmt.Errorf("failed to update delivery setting")
	}

	return nil
}

func (serv *DeliverySettingServImpl) GetAll(userID uuid.UUID) ([]resDelivery.DeliverySettingResponse, error) {
	if err := serv.checkClientRole(userID); err != nil {
		return nil, err
	}

	schema, err := serv.getSchema(userID)
	if err != nil {
		return nil, err
	}

	settings, err := serv.DeliverySettingRepo.GetAll(serv.Db, schema)
	if err != nil {
		log.Printf("[DeliverySettingRepo].GetAll error: %v", err)
		return nil, fmt.Errorf("failed to get delivery settings")
	}

	deliveries := domains.ToDeliverySetting(settings)
	return resDelivery.ToDeliverySettingResponses(deliveries), nil
}

func (serv *DeliverySettingServImpl) GetBySubGroupName(userID uuid.UUID, subGroupName string) (*resDelivery.DeliverySettingResponse, error) {
	if err := serv.checkClientRole(userID); err != nil {
		return nil, err
	}

	schema, err := serv.getSchema(userID)
	if err != nil {
		return nil, err
	}

	settings, err := serv.DeliverySettingRepo.GetBySubGroupName(serv.Db, schema, subGroupName)
	if err != nil || len(settings) == 0 {
		return nil, fmt.Errorf("delivery setting not found")
	}

	deliveries := domains.ToDeliverySetting(settings)
	if len(deliveries) == 0 {
		return nil, fmt.Errorf("delivery setting not found")
	}

	response := resDelivery.ToDeliverySettingResponse(deliveries[0])
	return &response, nil
}

func (serv *DeliverySettingServImpl) Delete(userID uuid.UUID, subGroupName string) error {
	if err := serv.checkClientRole(userID); err != nil {
		return err
	}

	schema, err := serv.getSchema(userID)
	if err != nil {
		return err
	}

	existing, err := serv.DeliverySettingRepo.GetBySubGroupName(serv.Db, schema, subGroupName)
	if err != nil || len(existing) == 0 {
		return fmt.Errorf("delivery setting not found")
	}

	if err := serv.DeliverySettingRepo.Delete(serv.Db, schema, subGroupName); err != nil {
		log.Printf("[DeliverySettingRepo].Delete error: %v", err)
		return fmt.Errorf("failed to delete delivery setting")
	}

	return nil
}
