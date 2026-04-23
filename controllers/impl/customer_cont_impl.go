package impl

import (
	"backend/exceptions"
	"backend/helpers"
	"backend/models/domains"
	req "backend/models/requests/customer"
	"backend/models/services"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type CustomerContImpl struct {
	CustomerServ services.CustomerServ
}

func NewCustomerContImpl(customerServ services.CustomerServ) *CustomerContImpl {
	return &CustomerContImpl{CustomerServ: customerServ}
}

// Create @Summary      Create Customer
// @Description  Buat customer baru secara generic. Saat ini mendukung flow WhatsApp dari form CRM dengan `account_type = Whatsapp`.
// @Tags         Customer
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path      string                         true "Client ID"
// @Param        request    body      customer.CreateCustomerRequest true "Create Customer Request"
// @Success      200        {object}  helpers.ApiResponse{data=customer.Response}
// @Failure      400        {object}  helpers.ApiResponse
// @Failure      500        {object}  helpers.ApiResponse
// @Router       /client/{client_id}/customers [post]
func (cont *CustomerContImpl) Create(ctx *gin.Context) {
	jwt := helpers.GetJwtToken(ctx)

	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	var request req.CreateCustomerRequest
	if err := helpers.ReadFromRequestBody(ctx, &request); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	accountType := strings.ToLower(strings.TrimSpace(request.AccountType))
	switch accountType {
	case "whatsapp":
		result, err := cont.CustomerServ.CreateWhatsApp(jwt, clientID, req.CreateWhatsAppCustomerRequest{
			Name:             request.Name,
			PhoneCountryCode: request.PhoneCountryCode,
			PhoneNumber:      request.PhoneNumber,
		})
		if err != nil {
			exceptions.ErrorHandler(ctx, err)
			return
		}
		response := helpers.ApiResponse{Success: true, Code: 200, Data: result}
		if err := helpers.WriteToResponseBody(ctx, response.Code, response); err != nil {
			exceptions.ErrorHandler(ctx, err)
		}
		return
	case "telegram":
		result, err := cont.CustomerServ.CreateTelegram(jwt, clientID, req.CreateTelegramCustomerRequest{
			Name:     request.Name,
			Username: request.Username,
		})
		if err != nil {
			exceptions.ErrorHandler(ctx, err)
			return
		}
		response := helpers.ApiResponse{Success: true, Code: 200, Data: result}
		if err := helpers.WriteToResponseBody(ctx, response.Code, response); err != nil {
			exceptions.ErrorHandler(ctx, err)
		}
		return
	default:
		exceptions.ErrorHandler(ctx, fmt.Errorf("unsupported account_type"))
		return
	}
}

func (cont *CustomerContImpl) Update(ctx *gin.Context) {
	jwt := helpers.GetJwtToken(ctx)

	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	customerID, err := strconv.Atoi(ctx.Param("customer_id"))
	if err != nil {
		exceptions.ErrorHandler(ctx, fmt.Errorf("invalid customer_id"))
		return
	}

	var request req.CreateCustomerRequest
	if err := helpers.ReadFromRequestBody(ctx, &request); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	result, err := cont.CustomerServ.Update(jwt, clientID, customerID, request)
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	response := helpers.ApiResponse{Success: true, Code: 200, Data: result}
	if err := helpers.WriteToResponseBody(ctx, response.Code, response); err != nil {
		exceptions.ErrorHandler(ctx, err)
	}
}

// CreateTelegram @Summary      Create Telegram Customer
// @Description  Buat customer baru via Telegram. Memvalidasi username ke Telegram API. Jika sudah terdaftar, kembalikan data yang ada.
// @Tags         Customer
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path      string                                true "Client ID"
// @Param        request    body      customer.CreateTelegramCustomerRequest true "Create Telegram Customer Request"
// @Success      200        {object}  helpers.ApiResponse{data=customer.Response}
// @Failure      400        {object}  helpers.ApiResponse
// @Failure      500        {object}  helpers.ApiResponse
// @Router       /client/{client_id}/customers/telegram [post]
func (cont *CustomerContImpl) CreateTelegram(ctx *gin.Context) {
	jwt := helpers.GetJwtToken(ctx)

	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	var request req.CreateTelegramCustomerRequest
	if err := helpers.ReadFromRequestBody(ctx, &request); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	result, err := cont.CustomerServ.CreateTelegram(jwt, clientID, request)
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

// CreateWhatsApp @Summary      Create WhatsApp Customer
// @Description  Buat customer baru via WhatsApp. Memvalidasi nomor ke WhatsApp API. Jika sudah terdaftar, kembalikan data yang ada.
// @Tags         Customer
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path      string                                 true "Client ID"
// @Param        request    body      customer.CreateWhatsAppCustomerRequest  true "Create WhatsApp Customer Request"
// @Success      200        {object}  helpers.ApiResponse{data=customer.Response}
// @Failure      400        {object}  helpers.ApiResponse
// @Failure      500        {object}  helpers.ApiResponse
// @Router       /client/{client_id}/customers/whatsapp [post]
func (cont *CustomerContImpl) CreateWhatsApp(ctx *gin.Context) {
	jwt := helpers.GetJwtToken(ctx)

	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	var request req.CreateWhatsAppCustomerRequest
	if err := helpers.ReadFromRequestBody(ctx, &request); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	result, err := cont.CustomerServ.CreateWhatsApp(jwt, clientID, request)
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

// GetAll @Summary      Get All Customers
// @Description  Ambil semua customer dengan pagination
// @Tags         Customer
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path   string  true  "Client ID"
// @Param        page       query  int     false "Page"
// @Param        limit      query  int     false "Limit"
// @Success      200        {object}  helpers.ApiResponse{data=pagination.Response}
// @Failure      401        {object}  helpers.ApiResponse
// @Failure      500        {object}  helpers.ApiResponse
// @Router       /client/{client_id}/customers [get]
func (cont *CustomerContImpl) GetAll(ctx *gin.Context) {
	jwt := helpers.GetJwtToken(ctx)

	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	pg := domains.ParsePagination(ctx)

	result, err := cont.CustomerServ.GetAll(jwt, clientID, pg)
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

// GetByID @Summary      Get Customer By ID
// @Description  Ambil detail customer berdasarkan ID
// @Tags         Customer
// @Produce      json
// @Security     BearerAuth
// @Param        client_id    path  string true "Client ID"
// @Param        customer_id  path  int    true "Customer ID"
// @Success      200          {object}  helpers.ApiResponse{data=customer.Response}
// @Failure      400          {object}  helpers.ApiResponse
// @Failure      401          {object}  helpers.ApiResponse
// @Failure      500          {object}  helpers.ApiResponse
// @Router       /client/{client_id}/customers/{customer_id} [get]
func (cont *CustomerContImpl) GetByID(ctx *gin.Context) {
	jwt := helpers.GetJwtToken(ctx)

	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	customerID, err := strconv.Atoi(ctx.Param("customer_id"))
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	result, err := cont.CustomerServ.GetByID(jwt, clientID, customerID)
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
