package impl

import (
	"backend/exceptions"
	"backend/helpers"
	reqDelivery "backend/models/requests/delivery_setting"
	"backend/models/services"

	"github.com/gin-gonic/gin"
)

type DeliverySettingContImpl struct {
	DeliverySettingServ services.DeliverySettingServ
}

func NewDeliverySettingContImpl(deliverySettingServ services.DeliverySettingServ) *DeliverySettingContImpl {
	return &DeliverySettingContImpl{DeliverySettingServ: deliverySettingServ}
}

// Create @Summary      Create Delivery Setting
// @Description  Buat pengaturan delivery baru
// @Tags         Delivery Setting
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path      string                                       true "Client ID"
// @Param        request    body      delivery.CreateDeliverySettingRequest true "Create Delivery Setting Request"
// @Success      200        {object}  helpers.ApiResponse
// @Failure      400        {object}  helpers.ApiResponse
// @Failure      401        {object}  helpers.ApiResponse
// @Failure      500        {object}  helpers.ApiResponse
// @Router       /client/{client_id}/delivery-settings [post]
func (cont *DeliverySettingContImpl) Create(ctx *gin.Context) {
	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	var request reqDelivery.CreateDeliverySettingRequest
	if err := helpers.ReadFromRequestBody(ctx, &request); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	if err := cont.DeliverySettingServ.Create(clientID, request); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	response := helpers.ApiResponse{Success: true, Code: 200, Data: nil}
	if err := helpers.WriteToResponseBody(ctx, response.Code, response); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
}

// Update @Summary      Update Delivery Setting
// @Description  Update pengaturan delivery berdasarkan sub group name
// @Tags         Delivery Setting
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        client_id       path      string                                       true "Client ID"
// @Param        sub_group_name  path      string                                       true "Sub Group Name"
// @Param        request         body      delivery.UpdateDeliverySettingRequest true "Update Delivery Setting Request"
// @Success      200             {object}  helpers.ApiResponse
// @Failure      400             {object}  helpers.ApiResponse
// @Failure      401             {object}  helpers.ApiResponse
// @Failure      500             {object}  helpers.ApiResponse
// @Router       /client/{client_id}/delivery-settings/{sub_group_name} [put]
func (cont *DeliverySettingContImpl) Update(ctx *gin.Context) {
	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	subGroupName := ctx.Param("sub_group_name")

	var request reqDelivery.UpdateDeliverySettingRequest
	if err := helpers.ReadFromRequestBody(ctx, &request); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	if err := cont.DeliverySettingServ.Update(clientID, subGroupName, request); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	response := helpers.ApiResponse{Success: true, Code: 200, Data: nil}
	if err := helpers.WriteToResponseBody(ctx, response.Code, response); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
}

// GetAll @Summary      Get All Delivery Settings
// @Description  Ambil semua pengaturan delivery
// @Tags         Delivery Setting
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path      string true "Client ID"
// @Success      200        {object}  helpers.ApiResponse{data=[]delivery_setting.DeliverySettingResponse}
// @Failure      401        {object}  helpers.ApiResponse
// @Failure      500        {object}  helpers.ApiResponse
// @Router       /client/{client_id}/delivery-settings [get]
func (cont *DeliverySettingContImpl) GetAll(ctx *gin.Context) {
	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	result, err := cont.DeliverySettingServ.GetAll(clientID)
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	response := helpers.ApiResponse{Success: true, Code: 200, Data: result}
	if err := helpers.WriteToResponseBody(ctx, response.Code, response); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
}

// GetBySubGroupName @Summary      Get Delivery Setting By Sub Group Name
// @Description  Ambil detail pengaturan delivery berdasarkan subgroup name
// @Tags         Delivery Setting
// @Produce      json
// @Security     BearerAuth
// @Param        client_id       path      string true "Client ID"
// @Param        sub_group_name  path      string true "Sub Group Name"
// @Success      200             {object}  helpers.ApiResponse{data=delivery_setting.DeliverySettingResponse}
// @Failure      400             {object}  helpers.ApiResponse
// @Failure      401             {object}  helpers.ApiResponse
// @Failure      500             {object}  helpers.ApiResponse
// @Router       /client/{client_id}/delivery-settings/{sub_group_name} [get]
func (cont *DeliverySettingContImpl) GetBySubGroupName(ctx *gin.Context) {
	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	subGroupName := ctx.Param("sub_group_name")

	result, err := cont.DeliverySettingServ.GetBySubGroupName(clientID, subGroupName)
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	response := helpers.ApiResponse{Success: true, Code: 200, Data: result}
	if err := helpers.WriteToResponseBody(ctx, response.Code, response); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
}

// Delete @Summary      Delete Delivery Setting
// @Description  Hapus pengaturan delivery berdasarkan sub group name
// @Tags         Delivery Setting
// @Produce      json
// @Security     BearerAuth
// @Param        client_id       path      string true "Client ID"
// @Param        sub_group_name  path      string true "Sub Group Name"
// @Success      200             {object}  helpers.ApiResponse
// @Failure      400             {object}  helpers.ApiResponse
// @Failure      401             {object}  helpers.ApiResponse
// @Failure      500             {object}  helpers.ApiResponse
// @Router       /client/{client_id}/delivery-settings/{sub_group_name} [delete]
func (cont *DeliverySettingContImpl) Delete(ctx *gin.Context) {
	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	subGroupName := ctx.Param("sub_group_name")

	if err := cont.DeliverySettingServ.Delete(clientID, subGroupName); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	response := helpers.ApiResponse{Success: true, Code: 200, Data: nil}
	if err := helpers.WriteToResponseBody(ctx, response.Code, response); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
}
