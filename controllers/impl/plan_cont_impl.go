package impl

import (
	"backend/exceptions"
	"backend/helpers"
	"backend/models/domains"
	"backend/models/requests/plan"
	"backend/models/services"

	"github.com/gin-gonic/gin"
)

type PlanContImpl struct {
	PlanServ services.PlanServ
}

func NewPlanContImpl(planServ services.PlanServ) *PlanContImpl {
	return &PlanContImpl{PlanServ: planServ}
}

// Create @Summary      Create Plan
// @Description  Buat plan baru (Hanya untuk SuperAdmin)
// @Tags         Plan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body      plan.CreateRequest true "Create Plan Request"
// @Success      200     {object}  helpers.ApiResponse
// @Failure      400     {object}  helpers.ApiResponse
// @Failure      401     {object}  helpers.ApiResponse
// @Router       /plans [post]
func (cont *PlanContImpl) Create(context *gin.Context) {
	jwtToken := helpers.GetJwtToken(context)

	request := plan.CreateRequest{}
	if err := helpers.ReadFromRequestBody(context, &request); err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	if err := cont.PlanServ.Create(jwtToken, request); err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	response := helpers.ApiResponse{
		Success: true,
		Code:    200,
		Data:    nil,
	}

	if err := helpers.WriteToResponseBody(context, response.Code, response); err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}
}

// Update @Summary      Update Plan
// @Description  Update plan berdasarkan plan_id (Hanya untuk SuperAdmin)
// @Tags         Plan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        plan_id  path      string             true "Plan ID"
// @Param        request  body      plan.UpdateRequest true "Update Plan Request"
// @Success      200      {object}  helpers.ApiResponse
// @Failure      400      {object}  helpers.ApiResponse
// @Failure      401      {object}  helpers.ApiResponse
// @Router       /plans/{plan_id} [put]
func (cont *PlanContImpl) Update(context *gin.Context) {
	jwtToken := helpers.GetJwtToken(context)

	planID, err := helpers.ParseUUID(context, "plan_id")
	if err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	request := plan.UpdateRequest{}
	if err := helpers.ReadFromRequestBody(context, &request); err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	if err := cont.PlanServ.Update(jwtToken, planID, request); err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	response := helpers.ApiResponse{
		Success: true,
		Code:    200,
		Data:    nil,
	}

	if err := helpers.WriteToResponseBody(context, response.Code, response); err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}
}

// ToggleIsActive @Summary      Toggle Plan Active Status
// @Description  Toggle status aktif plan berdasarkan plan_id (Hanya untuk SuperAdmin)
// @Tags         Plan
// @Produce      json
// @Security     BearerAuth
// @Param        plan_id  path      string true "Plan ID"
// @Success      200      {object}  helpers.ApiResponse
// @Failure      400      {object}  helpers.ApiResponse
// @Failure      401      {object}  helpers.ApiResponse
// @Router       /plans/{plan_id}/toggle [patch]
func (cont *PlanContImpl) ToggleIsActive(context *gin.Context) {
	jwtToken := helpers.GetJwtToken(context)

	planID, err := helpers.ParseUUID(context, "plan_id")
	if err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	if err := cont.PlanServ.ToggleIsActive(jwtToken, planID); err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	response := helpers.ApiResponse{
		Success: true,
		Code:    200,
		Data:    nil,
	}

	if err := helpers.WriteToResponseBody(context, response.Code, response); err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}
}

// GetAll @Summary      Get All Plans
// @Description  Ambil semua plan dengan pagination
// @Tags         Plan
// @Produce      json
// @Security     BearerAuth
// @Param        page   query     int false "Page"
// @Param        limit  query     int false "Limit"
// @Success      200    {object}  helpers.ApiResponse{data=pagination.Response}
// @Failure      401    {object}  helpers.ApiResponse
// @Router       /plans [get]
func (cont *PlanContImpl) GetAll(context *gin.Context) {
	jwtToken := helpers.GetJwtToken(context)

	pg := domains.ParsePagination(context)

	result, err := cont.PlanServ.GetAll(jwtToken, pg)
	if err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	response := helpers.ApiResponse{
		Success: true,
		Code:    200,
		Data:    result,
	}

	if err := helpers.WriteToResponseBody(context, response.Code, response); err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}
}

// GetById @Summary      Get Plan By ID
// @Description  Ambil data plan berdasarkan plan_id
// @Tags         Plan
// @Produce      json
// @Security     BearerAuth
// @Param        plan_id  path      string true "Plan ID"
// @Success      200      {object}  helpers.ApiResponse{data=plan.Response}
// @Failure      400      {object}  helpers.ApiResponse
// @Failure      401      {object}  helpers.ApiResponse
// @Router       /plans/{plan_id} [get]
func (cont *PlanContImpl) GetById(context *gin.Context) {
	jwtToken := helpers.GetJwtToken(context)

	planID, err := helpers.ParseUUID(context, "plan_id")
	if err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	result, err := cont.PlanServ.GetById(jwtToken, planID)
	if err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	response := helpers.ApiResponse{
		Success: true,
		Code:    200,
		Data:    result,
	}

	if err := helpers.WriteToResponseBody(context, response.Code, response); err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}
}

// Delete @Summary      Delete Plan
// @Description  Hapus plan berdasarkan plan_id (Hanya untuk SuperAdmin)
// @Tags         Plan
// @Produce      json
// @Security     BearerAuth
// @Param        plan_id  path      string true "Plan ID"
// @Success      200      {object}  helpers.ApiResponse
// @Failure      400      {object}  helpers.ApiResponse
// @Failure      401      {object}  helpers.ApiResponse
// @Router       /plans/{plan_id} [delete]
func (cont *PlanContImpl) Delete(context *gin.Context) {
	jwtToken := helpers.GetJwtToken(context)

	planID, err := helpers.ParseUUID(context, "plan_id")
	if err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	if err := cont.PlanServ.Delete(jwtToken, planID); err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	response := helpers.ApiResponse{
		Success: true,
		Code:    200,
		Data:    nil,
	}

	if err := helpers.WriteToResponseBody(context, response.Code, response); err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}
}
