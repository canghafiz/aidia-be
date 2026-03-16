package impl

import (
	"backend/exceptions"
	"backend/helpers"
	"backend/models/domains"
	"backend/models/requests/user"
	"backend/models/services"

	"github.com/gin-gonic/gin"
)

type UsersContImpl struct {
	UsersService services.UsersServ
}

func NewUsersContImpl(usersService services.UsersServ) *UsersContImpl {
	return &UsersContImpl{UsersService: usersService}
}

// Login @Summary      Login
// @Description  Login menggunakan username atau email beserta password
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body      user.LoginRequest true "Login Request"
// @Success      200     {object}  helpers.ApiResponse{data=user.LoginResponse}
// @Failure      400     {object}  helpers.ApiResponse
// @Failure      401     {object}  helpers.ApiResponse
// @Router       /auth/login [post]
func (cont *UsersContImpl) Login(context *gin.Context) {
	request := user.LoginRequest{}
	errParse := helpers.ReadFromRequestBody(context, &request)
	if errParse != nil {
		exceptions.ErrorHandler(context, errParse)
		return
	}

	result, errServ := cont.UsersService.Login(request)
	if errServ != nil {
		exceptions.ErrorHandler(context, errServ)
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

// ChangePw @Summary      Change Password
// @Description  Ganti password user yang sedang login
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body      user.ChangePwRequest true "Change Password Request"
// @Success      200     {object}  helpers.ApiResponse
// @Failure      400     {object}  helpers.ApiResponse
// @Failure      401     {object}  helpers.ApiResponse
// @Router       /auth/change-password [patch]
func (cont *UsersContImpl) ChangePw(context *gin.Context) {
	jwt := helpers.GetJwtToken(context)

	request := user.ChangePwRequest{}
	errParse := helpers.ReadFromRequestBody(context, &request)
	if errParse != nil {
		exceptions.ErrorHandler(context, errParse)
		return
	}

	err := cont.UsersService.ChangePw(jwt, request)
	if err != nil {
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

// Me @Summary      Get Current User
// @Description  Ambil data user yang sedang login beserta tenant dan business profile
// @Tags         Auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  helpers.ApiResponse{data=user.Response}
// @Failure      401  {object}  helpers.ApiResponse
// @Router       /auth/me [get]
func (cont *UsersContImpl) Me(context *gin.Context) {
	jwt := helpers.GetJwtToken(context)

	result, err := cont.UsersService.Me(jwt)
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

// CheckSuperAdminExist @Summary      Check SuperAdmin Exist
// @Description  Cek apakah SuperAdmin sudah ada di sistem (jika true === ada, tidak perlu panggil api create super admin), dipanggil pertama kali sebelum login
// @Tags         Auth
// @Produce      json
// @Success      200  {object}  helpers.ApiResponse{data=bool}
// @Failure      400  {object}  helpers.ApiResponse
// @Router       /auth/check-superadmin-exist [get]
func (cont *UsersContImpl) CheckSuperAdminExist(context *gin.Context) {
	result, err := cont.UsersService.CheckSuperAdminExist()
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

// CreateSuperAdmin @Summary      Create SuperAdmin
// @Description  Buat akun SuperAdmin, hanya bisa dilakukan sekali
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body      user.CreateSuperAdminRequest true "Create SuperAdmin Request"
// @Success      200     {object}  helpers.ApiResponse
// @Failure      400     {object}  helpers.ApiResponse
// @Router       /auth/create-superadmin [post]
func (cont *UsersContImpl) CreateSuperAdmin(context *gin.Context) {
	request := user.CreateSuperAdminRequest{}
	errParse := helpers.ReadFromRequestBody(context, &request)
	if errParse != nil {
		exceptions.ErrorHandler(context, errParse)
		return
	}

	errServ := cont.UsersService.CreateSuperAdmin(request)
	if errServ != nil {
		exceptions.ErrorHandler(context, errServ)
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

// CreateUser @Summary      Create User
// @Description  Buat user baru role admin / client, untuk data role id didapetin dari manggil api roles
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body      user.CreateUserRequest true "Create User Request"
// @Success      200     {object}  helpers.ApiResponse
// @Failure      400     {object}  helpers.ApiResponse
// @Failure      401     {object}  helpers.ApiResponse
// @Router       /users [post]
func (cont *UsersContImpl) CreateUser(context *gin.Context) {
	jwtToken := helpers.GetJwtToken(context)

	request := user.CreateUserRequest{}
	if err := helpers.ReadFromRequestBody(context, &request); err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	if err := cont.UsersService.CreateUser(jwtToken, request); err != nil {
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

// UpdateProfileClient @Summary      Update Profile Client
// @Description  Update profile user login role client berdasarkan user_id
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        user_id  path      string                  true "User ID"
// @Param        request  body      user.UpdateProfileRequest true "Update Profile Request"
// @Success      200      {object}  helpers.ApiResponse
// @Failure      400      {object}  helpers.ApiResponse
// @Failure      401      {object}  helpers.ApiResponse
// @Router       /users/{user_id}/client [put]
func (cont *UsersContImpl) UpdateProfileClient(context *gin.Context) {
	jwtToken := helpers.GetJwtToken(context)

	userID, err := helpers.ParseUUID(context, "user_id")
	if err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	request := user.UpdateProfileRequest{}
	if err := helpers.ReadFromRequestBody(context, &request); err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	if err := cont.UsersService.UpdateProfileClient(jwtToken, userID, request); err != nil {
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

// UpdateProfileNonClient @Summary      Update Profile Non Client
// @Description  Update profile user login role yang bukan client berdasarkan user_id
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        user_id  path      string                  true "User ID"
// @Param        request  body      user.UpdateProfileNonClientRequest true "Update Profile Non Client Request"
// @Success      200      {object}  helpers.ApiResponse
// @Failure      400      {object}  helpers.ApiResponse
// @Failure      401      {object}  helpers.ApiResponse
// @Router       /users/{user_id}/non-client [put]
func (cont *UsersContImpl) UpdateProfileNonClient(context *gin.Context) {
	jwtToken := helpers.GetJwtToken(context)

	userID, err := helpers.ParseUUID(context, "user_id")
	if err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	request := user.UpdateProfileNonClientRequest{}
	if err := helpers.ReadFromRequestBody(context, &request); err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	if err := cont.UsersService.UpdateProfileNonClient(jwtToken, userID, request); err != nil {
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

// EditUserData @Summary      Edit User Data
// @Description  Edit user data lain berdasarkan user_id
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        user_id  path      string                  true "User ID"
// @Param        request  body      user.EditUserDataRequest true "Update User Data Request"
// @Success      200      {object}  helpers.ApiResponse
// @Failure      400      {object}  helpers.ApiResponse
// @Failure      401      {object}  helpers.ApiResponse
// @Router       /users/{user_id}/other [put]
func (cont *UsersContImpl) EditUserData(context *gin.Context) {
	jwtToken := helpers.GetJwtToken(context)

	userID, err := helpers.ParseUUID(context, "user_id")
	if err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	request := user.EditUserDataRequest{}
	if err := helpers.ReadFromRequestBody(context, &request); err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	if err := cont.UsersService.EditUserData(jwtToken, userID, request); err != nil {
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

// GetByUserId @Summary      Get User By ID
// @Description  Ambil data user berdasarkan user_id
// @Tags         Users
// @Produce      json
// @Param        user_id  path      string true "User ID"
// @Success      200      {object}  helpers.ApiResponse{data=user.SingleResponse}
// @Failure      400      {object}  helpers.ApiResponse
// @Failure      401      {object}  helpers.ApiResponse
// @Router       /users/{user_id} [get]
func (cont *UsersContImpl) GetByUserId(context *gin.Context) {
	userID, err := helpers.ParseUUID(context, "user_id")
	if err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	result, err := cont.UsersService.GetByUserId(userID)
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

// GetUsers @Summary      Get All Users
// @Description  Ambil semua user dengan pagination
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Param        page   query     int false "Page"
// @Param        limit  query     int false "Limit"
// @Success      200    {object}  helpers.ApiResponse{data=pagination.Response}
// @Failure      401    {object}  helpers.ApiResponse
// @Router       /users [get]
func (cont *UsersContImpl) GetUsers(context *gin.Context) {
	jwtToken := helpers.GetJwtToken(context)

	pg := domains.ParsePagination(context)

	result, err := cont.UsersService.GetUsers(jwtToken, pg)
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

// FilterUsers @Summary      Filter Users
// @Description  Filter user berdasarkan name, email, dan role dengan pagination
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Param        name   query     string false "Name"
// @Param        email  query     string false "Email"
// @Param        role   query     string false "Role"
// @Param        page   query     int    false "Page"
// @Param        limit  query     int    false "Limit"
// @Success      200    {object}  helpers.ApiResponse{data=pagination.Response}
// @Failure      401    {object}  helpers.ApiResponse
// @Router       /users/filter [get]
func (cont *UsersContImpl) FilterUsers(context *gin.Context) {
	name := context.Query("name")
	email := context.Query("email")
	role := context.Query("role")
	pg := domains.ParsePagination(context)

	result, err := cont.UsersService.FilterUsers(name, email, role, pg)
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

// DeleteByUserId @Summary      Delete User
// @Description  Hapus user berdasarkan user_id
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Param        user_id  path      string true "User ID"
// @Success      200      {object}  helpers.ApiResponse
// @Failure      400      {object}  helpers.ApiResponse
// @Failure      401      {object}  helpers.ApiResponse
// @Router       /users/{user_id} [delete]
func (cont *UsersContImpl) DeleteByUserId(context *gin.Context) {
	jwtToken := helpers.GetJwtToken(context)

	userID, err := helpers.ParseUUID(context, "user_id")
	if err != nil {
		exceptions.ErrorHandler(context, err)
		return
	}

	if err := cont.UsersService.DeleteByUserId(jwtToken, userID); err != nil {
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
