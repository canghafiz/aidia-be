package impl

import (
	"backend/helpers"
	"backend/models/domains"
	"backend/models/repositories"
	reqAvailability "backend/models/requests/delivery_avaibility"
	resAvailability "backend/models/responses/delivery_avaibility"
	"fmt"
	"log"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DeliveryAvailabilitySettingServImpl struct {
	Db                              *gorm.DB
	Validator                       *validator.Validate
	UserRepo                        repositories.UsersRepo
	DeliveryAvailabilitySettingRepo repositories.DeliveryAvailabilitySettingRepo
	DeliverySettingRepo             repositories.DeliverySettingRepo
}

func NewDeliveryAvailabilitySettingServImpl(
	db *gorm.DB,
	validator *validator.Validate,
	userRepo repositories.UsersRepo,
	deliveryAvailabilitySettingRepo repositories.DeliveryAvailabilitySettingRepo,
	deliverySettingRepo repositories.DeliverySettingRepo,
) *DeliveryAvailabilitySettingServImpl {
	return &DeliveryAvailabilitySettingServImpl{
		Db:                              db,
		Validator:                       validator,
		UserRepo:                        userRepo,
		DeliveryAvailabilitySettingRepo: deliveryAvailabilitySettingRepo,
		DeliverySettingRepo:             deliverySettingRepo,
	}
}

func (serv *DeliveryAvailabilitySettingServImpl) getSchema(userID uuid.UUID) (string, error) {
	user, err := serv.UserRepo.GetByUserId(serv.Db, userID)
	if err != nil {
		return "", fmt.Errorf("user not found")
	}
	return user.Username, nil
}

func (serv *DeliveryAvailabilitySettingServImpl) checkClientRole(userID uuid.UUID) error {
	role, err := serv.UserRepo.GetUserRole(serv.Db, userID)
	if err != nil {
		return fmt.Errorf("failed to get user role")
	}
	if role != "Client" {
		return fmt.Errorf("access denied")
	}
	return nil
}

func (serv *DeliveryAvailabilitySettingServImpl) buildDeliveryMap(schema string) (map[string]string, error) {
	deliverySettings, err := serv.DeliverySettingRepo.GetAll(serv.Db, schema)
	if err != nil {
		return nil, err
	}
	deliveries := domains.ToDeliverySetting(deliverySettings)
	deliveryMap := map[string]string{}
	for _, d := range deliveries {
		deliveryMap[d.SubGroupName] = d.Name
	}
	return deliveryMap, nil
}

func (serv *DeliveryAvailabilitySettingServImpl) Create(userID uuid.UUID, request reqAvailability.CreateDeliveryAvailabilityRequest) error {
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

	existing, err := serv.DeliverySettingRepo.GetBySubGroupName(serv.Db, schema, request.DeliverySubGroup)
	if err != nil || len(existing) == 0 {
		return fmt.Errorf("delivery subgroup not found")
	}

	settings := reqAvailability.CreateDeliveryAvailabilityToDomain(request)
	if err := serv.DeliveryAvailabilitySettingRepo.Create(serv.Db, schema, settings); err != nil {
		log.Printf("[DeliveryAvailabilitySettingRepo].Create error: %v", err)
		return fmt.Errorf("failed to create delivery availability setting")
	}

	return nil
}

func (serv *DeliveryAvailabilitySettingServImpl) Update(userID uuid.UUID, subGroupName string, request reqAvailability.UpdateDeliveryAvailabilityRequest) error {
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

	existing, err := serv.DeliveryAvailabilitySettingRepo.GetBySubGroupName(serv.Db, schema, subGroupName)
	if err != nil || len(existing) == 0 {
		return fmt.Errorf("delivery availability setting not found")
	}

	deliveryExisting, err := serv.DeliverySettingRepo.GetBySubGroupName(serv.Db, schema, request.DeliverySubGroup)
	if err != nil || len(deliveryExisting) == 0 {
		return fmt.Errorf("delivery subgroup not found")
	}

	settings := reqAvailability.UpdateDeliveryAvailabilityToDomain(request, subGroupName)
	if err := serv.DeliveryAvailabilitySettingRepo.Update(serv.Db, schema, settings); err != nil {
		log.Printf("[DeliveryAvailabilitySettingRepo].Update error: %v", err)
		return fmt.Errorf("failed to update delivery availability setting")
	}

	return nil
}

func (serv *DeliveryAvailabilitySettingServImpl) GetAll(userID uuid.UUID) ([]resAvailability.DeliveryAvailabilityResponse, error) {
	if err := serv.checkClientRole(userID); err != nil {
		return nil, err
	}

	schema, err := serv.getSchema(userID)
	if err != nil {
		return nil, err
	}

	settings, err := serv.DeliveryAvailabilitySettingRepo.GetAll(serv.Db, schema)
	if err != nil {
		log.Printf("[DeliveryAvailabilitySettingRepo].GetAll error: %v", err)
		return nil, fmt.Errorf("failed to get delivery availability settings")
	}

	deliveryMap, err := serv.buildDeliveryMap(schema)
	if err != nil {
		log.Printf("[DeliverySettingRepo].GetAll error: %v", err)
		return nil, fmt.Errorf("failed to get delivery settings")
	}

	availabilities := domains.ToDeliveryAvailabilitySetting(settings)
	return resAvailability.ToDeliveryAvailabilityResponses(availabilities, deliveryMap), nil
}

func (serv *DeliveryAvailabilitySettingServImpl) GetBySubGroupName(userID uuid.UUID, subGroupName string) (*resAvailability.DeliveryAvailabilityResponse, error) {
	if err := serv.checkClientRole(userID); err != nil {
		return nil, err
	}

	schema, err := serv.getSchema(userID)
	if err != nil {
		return nil, err
	}

	settings, err := serv.DeliveryAvailabilitySettingRepo.GetBySubGroupName(serv.Db, schema, subGroupName)
	if err != nil || len(settings) == 0 {
		return nil, fmt.Errorf("delivery availability setting not found")
	}

	deliveryMap, err := serv.buildDeliveryMap(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery settings")
	}

	availabilities := domains.ToDeliveryAvailabilitySetting(settings)
	if len(availabilities) == 0 {
		return nil, fmt.Errorf("delivery availability setting not found")
	}

	response := resAvailability.ToDeliveryAvailabilityResponse(availabilities[0], deliveryMap[availabilities[0].DeliverySubGroup])
	return &response, nil
}

func (serv *DeliveryAvailabilitySettingServImpl) Delete(userID uuid.UUID, subGroupName string) error {
	if err := serv.checkClientRole(userID); err != nil {
		return err
	}

	schema, err := serv.getSchema(userID)
	if err != nil {
		return err
	}

	existing, err := serv.DeliveryAvailabilitySettingRepo.GetBySubGroupName(serv.Db, schema, subGroupName)
	if err != nil || len(existing) == 0 {
		return fmt.Errorf("delivery availability setting not found")
	}

	if err := serv.DeliveryAvailabilitySettingRepo.Delete(serv.Db, schema, subGroupName); err != nil {
		log.Printf("[DeliveryAvailabilitySettingRepo].Delete error: %v", err)
		return fmt.Errorf("failed to delete delivery availability setting")
	}

	return nil
}
