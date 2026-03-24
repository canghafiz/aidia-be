package impl

import (
	"backend/helpers"
	"backend/hub"
	"backend/models/repositories"
	reqKitchen "backend/models/requests/kitchen_order"
	resKitchen "backend/models/responses/kitchen_order"
	"encoding/json"
	"fmt"
	"log"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type KitchenOrderServImpl struct {
	Db               *gorm.DB
	JwtKey           string
	Validator        *validator.Validate
	UserRepo         repositories.UsersRepo
	KitchenOrderRepo repositories.KitchenOrderRepo
}

func NewKitchenOrderServImpl(
	db *gorm.DB,
	jwtKey string,
	validator *validator.Validate,
	userRepo repositories.UsersRepo,
	kitchenOrderRepo repositories.KitchenOrderRepo,
) *KitchenOrderServImpl {
	return &KitchenOrderServImpl{
		Db:               db,
		JwtKey:           jwtKey,
		Validator:        validator,
		UserRepo:         userRepo,
		KitchenOrderRepo: kitchenOrderRepo,
	}
}

func (serv *KitchenOrderServImpl) getSchema(clientID uuid.UUID) (string, error) {
	user, err := serv.UserRepo.GetByUserId(serv.Db, clientID)
	if err != nil {
		return "", fmt.Errorf("user not found")
	}
	return user.Username, nil
}

func (serv *KitchenOrderServImpl) checkRole(accessToken string) error {
	_, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"SuperAdmin", "Admin", "Client"})
	if err != nil || !ok {
		return fmt.Errorf("access denied")
	}
	return nil
}

func (serv *KitchenOrderServImpl) GetDisplay(accessToken string, clientID uuid.UUID) (*resKitchen.KitchenDisplayResponse, error) {
	if err := serv.checkRole(accessToken); err != nil {
		return nil, err
	}

	schema, err := serv.getSchema(clientID)
	if err != nil {
		return nil, err
	}

	orders, err := serv.KitchenOrderRepo.GetAll(serv.Db, schema)
	if err != nil {
		log.Printf("[KitchenOrderRepo].GetAll error: %v", err)
		return nil, fmt.Errorf("failed to get kitchen orders")
	}

	display := resKitchen.ToKitchenDisplayResponse(orders)
	return &display, nil
}

func (serv *KitchenOrderServImpl) UpdateStatus(accessToken string, clientID uuid.UUID, id uuid.UUID, request reqKitchen.UpdateKitchenStatusRequest) error {
	if err := serv.checkRole(accessToken); err != nil {
		return err
	}

	if err := helpers.ErrValidator(request, serv.Validator); err != nil {
		return err
	}

	schema, err := serv.getSchema(clientID)
	if err != nil {
		return err
	}

	_, err = serv.KitchenOrderRepo.GetByID(serv.Db, schema, id)
	if err != nil {
		log.Printf("[KitchenOrderRepo].GetByID error: %v", err)
		return fmt.Errorf("kitchen order not found")
	}

	if err := serv.KitchenOrderRepo.UpdateStatus(serv.Db, schema, id, request.Status); err != nil {
		log.Printf("[KitchenOrderRepo].UpdateStatus error: %v", err)
		return fmt.Errorf("failed to update kitchen status")
	}

	go serv.broadcastUpdate(schema)

	return nil
}

func (serv *KitchenOrderServImpl) broadcastUpdate(schema string) {
	orders, err := serv.KitchenOrderRepo.GetAll(serv.Db, schema)
	if err != nil {
		log.Printf("[KitchenOrderServ] broadcastUpdate error: %v", err)
		return
	}

	display := resKitchen.ToKitchenDisplayResponse(orders)
	event := resKitchen.KitchenSSEEvent{
		Type: "update",
		Data: display,
	}

	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("[KitchenOrderServ] marshal error: %v", err)
		return
	}

	hub.GetKitchenHub().Broadcast(schema, string(data))
}
