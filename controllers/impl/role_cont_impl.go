package impl

import (
	"backend/exceptions"
	"backend/helpers"
	"backend/models/services"

	"github.com/gin-gonic/gin"
)

type RoleContImpl struct {
	RoleServ services.RoleServ
}

func NewRoleContImpl(roleServ services.RoleServ) *RoleContImpl {
	return &RoleContImpl{RoleServ: roleServ}
}

// GetRoles @Summary      Get All Roles
// @Description  Ambil semua data role
// @Tags         Roles
// @Produce      json
// @Success      200  {object}  helpers.ApiResponse{data=[]domains.Roles}
// @Failure      400  {object}  helpers.ApiResponse
// @Failure      401  {object}  helpers.ApiResponse
// @Router       /roles [get]
func (cont *RoleContImpl) GetRoles(context *gin.Context) {
	result, err := cont.RoleServ.GetRoles()
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
