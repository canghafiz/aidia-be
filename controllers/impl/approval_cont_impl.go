package impl

import (
	"backend/exceptions"
	"backend/helpers"
	"backend/models/domains"
	"backend/models/services"

	"github.com/gin-gonic/gin"
)

type ApprovalCont struct {
	ApprovalServ services.ApprovalServ
}

func NewApprovalCont(approvalServ services.ApprovalServ) *ApprovalCont {
	return &ApprovalCont{ApprovalServ: approvalServ}
}

// Approval @Summary      Approve User (Hanya untuk role SuperAdmin)
// @Description  Approve user berdasarkan approval_id (Hanya untuk role SuperAdmin)
// @Tags         Approval
// @Produce      json
// @Security     BearerAuth
// @Param        approval_id  path      string true "User ID"
// @Success      200      {object}  helpers.ApiResponse
// @Failure      400      {object}  helpers.ApiResponse
// @Failure      401      {object}  helpers.ApiResponse
// @Router       /approvals/{approval_id} [patch]
func (cont *ApprovalCont) Approval(context *gin.Context) {
	jwtToken := helpers.GetJwtToken(context)

	userID, err := helpers.ParseUUID(context, "approval_id")
	if err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	if err := cont.ApprovalServ.Approve(jwtToken, userID); err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	response := helpers.ApiResponse{
		Success: true,
		Code:    200,
		Data:    nil,
	}

	errResponse := helpers.WriteToResponseBody(context, response.Code, response)
	if errResponse != nil {
		exceptions.ErrorHandler(context, errResponse)
		return
	}
}

// GetAll @Summary      Get All Approval Logs
// @Description  Ambil semua approval logs dengan pagination (Hanya untuk role SuperAdmin)
// @Tags         Approval
// @Produce      json
// @Security     BearerAuth
// @Param        page   query     int false "Page"
// @Param        limit  query     int false "Limit"
// @Success      200    {object}  helpers.ApiResponse{data=pagination.Response}
// @Failure      401    {object}  helpers.ApiResponse
// @Router       /approvals [get]
func (cont *ApprovalCont) GetAll(context *gin.Context) {
	jwtToken := helpers.GetJwtToken(context)
	pg := domains.ParsePagination(context)

	result, err := cont.ApprovalServ.GetAll(jwtToken, pg)
	if err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	response := helpers.ApiResponse{
		Success: true,
		Code:    200,
		Data:    result,
	}

	errResponse := helpers.WriteToResponseBody(context, response.Code, response)
	if errResponse != nil {
		exceptions.ErrorHandler(context, errResponse)
		return
	}
}

// Delete @Summary      Delete Approval Log (Hanya untuk role SuperAdmin)ear
// @Description  Hapus approval log berdasarkan approval_id (Hanya untuk role SuperAdmin)
// @Tags         Approval
// @Produce      json
// @Security     BearerAuth
// @Param        approval_id  path      string true "User ID"
// @Success      200      {object}  helpers.ApiResponse
// @Failure      400      {object}  helpers.ApiResponse
// @Failure      401      {object}  helpers.ApiResponse
// @Router       /approvals/{approval_id} [delete]
func (cont *ApprovalCont) Delete(context *gin.Context) {
	jwtToken := helpers.GetJwtToken(context)

	userID, err := helpers.ParseUUID(context, "approval_id")
	if err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	if err := cont.ApprovalServ.Delete(jwtToken, userID); err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	response := helpers.ApiResponse{
		Success: true,
		Code:    200,
		Data:    nil,
	}

	errResponse := helpers.WriteToResponseBody(context, response.Code, response)
	if errResponse != nil {
		exceptions.ErrorHandler(context, errResponse)
		return
	}
}
