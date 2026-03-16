package impl

import (
	"backend/exceptions"
	"backend/helpers"
	reqAvailability "backend/models/requests/delivery_avaibility"
	"backend/models/services"

	"github.com/gin-gonic/gin"
)

type DeliveryAvailabilitySettingContImpl struct {
	DeliveryAvailabilitySettingServ services.DeliveryAvailabilitySettingServ
}

func NewDeliveryAvailabilitySettingContImpl(serv services.DeliveryAvailabilitySettingServ) *DeliveryAvailabilitySettingContImpl {
	return &DeliveryAvailabilitySettingContImpl{DeliveryAvailabilitySettingServ: serv}
}

// Create @Summary      Create Delivery Availability Setting
// @Description  Buat pengaturan ketersediaan delivery baru, field type=available/not available
// @Tags         Delivery Availability Setting
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path      string                                                   true "Client ID"
// @Param        request    body      reqAvailability.CreateDeliveryAvailabilityRequest        true "Create Delivery Availability Request"
// @Success      200        {object}  helpers.ApiResponse
// @Failure      400        {object}  helpers.ApiResponse
// @Failure      401        {object}  helpers.ApiResponse
// @Failure      500        {object}  helpers.ApiResponse
// @Router       /client/{client_id}/delivery-availability-settings [post]
func (cont *DeliveryAvailabilitySettingContImpl) Create(ctx *gin.Context) {
	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	var request reqAvailability.CreateDeliveryAvailabilityRequest
	if err := helpers.ReadFromRequestBody(ctx, &request); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	if err := cont.DeliveryAvailabilitySettingServ.Create(clientID, request); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	response := helpers.ApiResponse{Success: true, Code: 200, Data: nil}
	if err := helpers.WriteToResponseBody(ctx, response.Code, response); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
}

// Update @Summary      Update Delivery Availability Setting
// @Description  Update pengaturan ketersediaan delivery berdasarkan sub group name, field type=available/not available
// @Tags         Delivery Availability Setting
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        client_id       path      string                                                   true "Client ID"
// @Param        sub_group_name  path      string                                                   true "Sub Group Name"
// @Param        request         body      reqAvailability.UpdateDeliveryAvailabilityRequest        true "Update Delivery Availability Request"
// @Success      200             {object}  helpers.ApiResponse
// @Failure      400             {object}  helpers.ApiResponse
// @Failure      401             {object}  helpers.ApiResponse
// @Failure      500             {object}  helpers.ApiResponse
// @Router       /client/{client_id}/delivery-availability-settings/{sub_group_name} [put]
func (cont *DeliveryAvailabilitySettingContImpl) Update(ctx *gin.Context) {
	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	subGroupName := ctx.Param("sub_group_name")

	var request reqAvailability.UpdateDeliveryAvailabilityRequest
	if err := helpers.ReadFromRequestBody(ctx, &request); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	if err := cont.DeliveryAvailabilitySettingServ.Update(clientID, subGroupName, request); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	response := helpers.ApiResponse{Success: true, Code: 200, Data: nil}
	if err := helpers.WriteToResponseBody(ctx, response.Code, response); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
}

// GetAll @Summary      Get All Delivery Availability Settings
// @Description  Ambil semua pengaturan ketersediaan delivery beserta nama delivery yang berelasi
// @Tags         Delivery Availability Setting
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path      string true "Client ID"
// @Success      200        {object}  helpers.ApiResponse{data=[]delivery_avaibility.DeliveryAvailabilityResponse}
// @Failure      401        {object}  helpers.ApiResponse
// @Failure      500        {object}  helpers.ApiResponse
// @Router       /client/{client_id}/delivery-availability-settings [get]
func (cont *DeliveryAvailabilitySettingContImpl) GetAll(ctx *gin.Context) {
	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	result, err := cont.DeliveryAvailabilitySettingServ.GetAll(clientID)
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

// GetBySubGroupName @Summary      Get Delivery Availability Setting By Sub Group Name
// @Description  Ambil detail pengaturan ketersediaan delivery berdasarkan sub group name
// @Tags         Delivery Availability Setting
// @Produce      json
// @Security     BearerAuth
// @Param        client_id       path      string true "Client ID"
// @Param        sub_group_name  path      string true "Sub Group Name"
// @Success      200             {object}  helpers.ApiResponse{data=delivery_avaibility.DeliveryAvailabilityResponse}
// @Failure      400             {object}  helpers.ApiResponse
// @Failure      401             {object}  helpers.ApiResponse
// @Failure      500             {object}  helpers.ApiResponse
// @Router       /client/{client_id}/delivery-availability-settings/{sub_group_name} [get]
func (cont *DeliveryAvailabilitySettingContImpl) GetBySubGroupName(ctx *gin.Context) {
	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	subGroupName := ctx.Param("sub_group_name")

	result, err := cont.DeliveryAvailabilitySettingServ.GetBySubGroupName(clientID, subGroupName)
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

// Delete @Summary      Delete Delivery Availability Setting
// @Description  Hapus pengaturan ketersediaan delivery berdasarkan sub group name
// @Tags         Delivery Availability Setting
// @Produce      json
// @Security     BearerAuth
// @Param        client_id       path      string true "Client ID"
// @Param        sub_group_name  path      string true "Sub Group Name"
// @Success      200             {object}  helpers.ApiResponse
// @Failure      400             {object}  helpers.ApiResponse
// @Failure      401             {object}  helpers.ApiResponse
// @Failure      500             {object}  helpers.ApiResponse
// @Router       /client/{client_id}/delivery-availability-settings/{sub_group_name} [delete]
func (cont *DeliveryAvailabilitySettingContImpl) Delete(ctx *gin.Context) {
	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	subGroupName := ctx.Param("sub_group_name")

	if err := cont.DeliveryAvailabilitySettingServ.Delete(clientID, subGroupName); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	response := helpers.ApiResponse{Success: true, Code: 200, Data: nil}
	if err := helpers.WriteToResponseBody(ctx, response.Code, response); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
}
