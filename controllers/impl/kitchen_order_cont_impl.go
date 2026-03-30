package impl

import (
	"backend/exceptions"
	"backend/helpers"
	"backend/hub"
	"backend/models/repositories"
	reqKitchen "backend/models/requests/kitchen_order"
	resKitchen "backend/models/responses/kitchen_order"
	"backend/models/services"
	"encoding/json"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type KitchenOrderContImpl struct {
	KitchenOrderServ services.KitchenOrderServ
	UserRepo         repositories.UsersRepo
	Db               *gorm.DB
}

func NewKitchenOrderContImpl(
	kitchenOrderServ services.KitchenOrderServ,
	userRepo repositories.UsersRepo,
	db *gorm.DB,
) *KitchenOrderContImpl {
	return &KitchenOrderContImpl{
		KitchenOrderServ: kitchenOrderServ,
		UserRepo:         userRepo,
		Db:               db,
	}
}

// GetDisplay @Summary      Get Kitchen Display
// @Description  Ambil semua order di kitchen display berdasarkan status
// @Tags         Kitchen Display
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path  string true "Client ID"
// @Success      200        {object}  helpers.ApiResponse{data=kitchen_order.KitchenDisplayResponse}
// @Failure      401        {object}  helpers.ApiResponse
// @Failure      500        {object}  helpers.ApiResponse
// @Router       /client/{client_id}/kitchen-display [get]
func (cont *KitchenOrderContImpl) GetDisplay(ctx *gin.Context) {
	accessToken := helpers.GetJwtToken(ctx)

	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	result, err := cont.KitchenOrderServ.GetDisplay(accessToken, clientID)
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

// Stream @Summary      Kitchen Display Stream (SSE)
// @Description  Subscribe realtime kitchen display via Server-Sent Events. Authentication via Bearer token OR query parameter 'token'
// @Tags         Kitchen Display
// @Produce      text/event-stream
// @Param        client_id  path  string  true  "Client ID"
// @Param        token      query string  false  "JWT Token (alternative to Authorization header)"
// @Success      200        {object}  resKitchen.KitchenSSEEvent
// @Failure      401        {object}  helpers.ApiResponse
// @Failure      500        {object}  helpers.ApiResponse
// @Router       /client/{client_id}/kitchen-display/stream [get]
func (cont *KitchenOrderContImpl) Stream(ctx *gin.Context) {
	// Handle CORS preflight OPTIONS request
	if ctx.Request.Method == "OPTIONS" {
		ctx.Header("Access-Control-Allow-Origin", "*")
		ctx.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
		ctx.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept, X-Requested-With, ngrok-skip-browser-warning")
		ctx.Header("Access-Control-Max-Age", "86400")
		ctx.AbortWithStatus(204)
		return
	}

	// Get token from Authorization header OR query parameter
	accessToken := helpers.GetJwtToken(ctx)
	if accessToken == "" {
		// Fallback: try query parameter
		accessToken = ctx.Query("token")
	}

	if accessToken == "" {
		exceptions.ErrorHandler(ctx, fmt.Errorf("authorization required"))
		return
	}

	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	// Resolve schema dari clientID
	user, err := cont.UserRepo.GetByUserId(cont.Db, clientID)
	if err != nil {
		exceptions.ErrorHandler(ctx, fmt.Errorf("user not found"))
		return
	}

	// Gunakan TenantSchema yang sudah normalized
	if user.TenantSchema == nil || *user.TenantSchema == "" {
		exceptions.ErrorHandler(ctx, fmt.Errorf("tenant schema not found"))
		return
	}
	schema := helpers.NormalizeSchema(*user.TenantSchema)

	// Ambil data awal
	result, err := cont.KitchenOrderServ.GetDisplay(accessToken, clientID)
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	// Set SSE headers dengan CORS yang benar
	ctx.Header("Access-Control-Allow-Origin", "*")
	ctx.Header("Access-Control-Allow-Credentials", "false") // harus false kalau Allow-Origin: *
	ctx.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
	ctx.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept, X-Requested-With, ngrok-skip-browser-warning")
	ctx.Header("Access-Control-Expose-Headers", "Content-Type")
	ctx.Header("ngrok-skip-browser-warning", "true") // skip ngrok interstitial page
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("X-Accel-Buffering", "no")
	ctx.Header("Transfer-Encoding", "chunked")

	// Subscribe ke hub pakai schema
	h := hub.GetKitchenHub()
	ch := h.Subscribe(schema)
	defer func() {
		h.Unsubscribe(schema, ch)
	}()

	// Kirim data awal
	initEvent := resKitchen.KitchenSSEEvent{
		Type: "init",
		Data: result,
	}
	initData, err := json.Marshal(initEvent)
	if err != nil {
		log.Printf("[KitchenSSE] marshal init error: %v", err)
		return
	}
	if _, err := fmt.Fprintf(ctx.Writer, "data: %s\n\n", initData); err != nil {
		log.Printf("[KitchenSSE] write init error: %v", err)
		return
	}
	ctx.Writer.Flush()

	// Listen update atau disconnect
	clientGone := ctx.Request.Context().Done()
	for {
		select {
		case <-clientGone:
			log.Printf("[KitchenSSE] client disconnected, schema: %s", schema)
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if _, err := fmt.Fprintf(ctx.Writer, "data: %s\n\n", msg); err != nil {
				log.Printf("[KitchenSSE] write error: %v", err)
				return
			}
			ctx.Writer.Flush()
		}
	}
}

// UpdateStatus @Summary      Update Kitchen Order Status
// @Description  Update status kitchen order (new_order/cooking/packing/ready)
// @Tags         Kitchen Display
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        client_id   path  string                                    true "Client ID"
// @Param        kitchen_id  path  string                                    true "Kitchen Order ID"
// @Param        request     body  kitchen_order.UpdateKitchenStatusRequest  true "Update Kitchen Status Request"
// @Success      200         {object}  helpers.ApiResponse
// @Failure      400         {object}  helpers.ApiResponse
// @Failure      401         {object}  helpers.ApiResponse
// @Failure      500         {object}  helpers.ApiResponse
// @Router       /client/{client_id}/kitchen-display/{kitchen_id}/status [patch]
func (cont *KitchenOrderContImpl) UpdateStatus(ctx *gin.Context) {
	accessToken := helpers.GetJwtToken(ctx)

	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	kitchenID, err := helpers.ParseUUID(ctx, "kitchen_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	var request reqKitchen.UpdateKitchenStatusRequest
	if err := helpers.ReadFromRequestBody(ctx, &request); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	if err := cont.KitchenOrderServ.UpdateStatus(accessToken, clientID, kitchenID, request); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	response := helpers.ApiResponse{Success: true, Code: 200, Data: nil}
	if err := helpers.WriteToResponseBody(ctx, response.Code, response); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
}
