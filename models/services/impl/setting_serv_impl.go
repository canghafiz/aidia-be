package impl

import (
	"backend/helpers"
	"backend/models/domains"
	"backend/models/repositories"
	req "backend/models/requests/setting"
	"backend/models/responses/setting"
	"fmt"
	"log"

	"gorm.io/gorm"
)

type SettingServImpl struct {
	Db          *gorm.DB
	JwtKey      string
	SettingRepo repositories.SettingRepo
}

func NewSettingServImpl(db *gorm.DB, jwtKey string, settingRepo repositories.SettingRepo) *SettingServImpl {
	return &SettingServImpl{Db: db, JwtKey: jwtKey, SettingRepo: settingRepo}
}

func (serv *SettingServImpl) getSchema(accessToken string, role *string) (string, error) {
	if *role == "SuperAdmin" || *role == "Admin" {
		return "public", nil
	}
	schema, err := helpers.GetUsernameFromToken(accessToken, serv.JwtKey)
	if err != nil {
		return "", err
	}
	return *schema, nil
}

func (serv *SettingServImpl) GetNotification(accessToken string) (*setting.GroupResponse, error) {
	role, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"SuperAdmin", "Admin", "Client"})
	if err != nil || !ok {
		return nil, err
	}

	schema, err := serv.getSchema(accessToken, role)
	if err != nil {
		return nil, err
	}

	result, errResult := serv.SettingRepo.GetByGroupName(serv.Db, schema, "notification")
	if errResult != nil {
		log.Printf("[SettingRepo].GetByGroupName error: %v", errResult)
		return nil, fmt.Errorf("failed to get setting notification")
	}

	response := setting.ToGroupResponse(result)
	return response, nil
}

func (serv *SettingServImpl) GetIntegration(accessToken string) (*setting.GroupResponse, error) {
	role, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"SuperAdmin", "Admin", "Client"})
	if err != nil || !ok {
		return nil, err
	}

	schema, err := serv.getSchema(accessToken, role)
	if err != nil {
		return nil, err
	}

	var result []domains.Setting
	var errResult error

	if *role == "Client" {
		result, errResult = serv.SettingRepo.GetByGroupAndSubGroupName(serv.Db, schema, "integration", "Telegram")
	} else {
		result, errResult = serv.SettingRepo.GetByGroupName(serv.Db, schema, "integration")
	}

	if errResult != nil {
		log.Printf("[SettingRepo].GetIntegration error: %v", errResult)
		return nil, fmt.Errorf("failed to get setting integration")
	}

	response := setting.ToGroupResponse(result)
	return response, nil
}

func (serv *SettingServImpl) UpdateBySubgroupName(accessToken, subGroupName string, requests req.UpdateBySubgroupRequest) error {
	role, ok, err := helpers.GetUserRoleFromToken(accessToken, serv.JwtKey, []string{"SuperAdmin", "Admin", "Client"})
	if err != nil || !ok {
		return err
	}

	schema, err := serv.getSchema(accessToken, role)
	if err != nil {
		return err
	}

	settings := req.UpdateSettingItemsToSettings(subGroupName, requests.Settings)

	if err := serv.SettingRepo.UpdateBySubGroupName(serv.Db, schema, settings); err != nil {
		log.Printf("[SettingRepo].UpdateBySubGroupName error: %v", err)
		return fmt.Errorf("failed to update setting")
	}

	return nil
}
