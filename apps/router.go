package apps

import (
	"backend/dependencies"
	"backend/middlewares"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Router struct {
	Dependency *dependencies.Dependency
	Engine     *gin.Engine
}

func NewRouter(r Router) *Router {
	r.Engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.Engine.StaticFS("/assets", http.Dir("./assets"))

	// Setup CORS global
	r.Engine.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           86400,
	}))

	middleware := middlewares.Middleware(r.Dependency.JwtKey)

	generalGroup := r.Engine.Group("api/v1/")
	{
		authGroup := generalGroup.Group("auth")
		{
			authGroup.POST("/create-superadmin", r.Dependency.UsersCont.CreateSuperAdmin)
			authGroup.POST("/login", r.Dependency.UsersCont.Login)
			authGroup.GET("/check-superadmin-exist", r.Dependency.UsersCont.CheckSuperAdminExist)

			middleware := authGroup.Use(middleware)
			{
				middleware.GET("/me", r.Dependency.UsersCont.Me)
				middleware.PATCH("/change-password", r.Dependency.UsersCont.ChangePw)
			}
		}

		usersGroup := generalGroup.Group("users")
		{
			usersGroup.GET("", r.Dependency.UsersCont.GetUsers)
			usersGroup.GET("/clients", r.Dependency.UsersCont.GetClients)
			usersGroup.GET("/filter", r.Dependency.UsersCont.FilterUsers)
			usersGroup.GET("/:user_id", r.Dependency.UsersCont.GetByUserId)

			middleware := usersGroup.Use(middleware)
			{
				middleware.POST("", r.Dependency.UsersCont.CreateUser)
				middleware.PUT("/:user_id/client", r.Dependency.UsersCont.UpdateProfileClient)
				middleware.PUT("/:user_id/non-client", r.Dependency.UsersCont.UpdateProfileNonClient)
				middleware.PUT("/:user_id/other", r.Dependency.UsersCont.EditUserData)
				middleware.DELETE("/:user_id", r.Dependency.UsersCont.DeleteByUserId)
			}
		}

		roleGroup := generalGroup.Group("roles")
		{
			roleGroup.GET("", r.Dependency.RoleCont.GetRoles)
		}

		settingGroup := generalGroup.Group("settings").Use(middleware)
		{
			settingGroup.GET("/notification", r.Dependency.SettingCont.GetNotification)
			settingGroup.GET("/integration", r.Dependency.SettingCont.GetIntegration)
			settingGroup.PATCH("/subgroup-name/:sub_group_name", r.Dependency.SettingCont.UpdateBySubgroupName)
		}

		// Telegram AI Prompt settings (per client)
		settingClientGroup := generalGroup.Group("client/:client_id/settings").Use(middleware)
		{
			settingClientGroup.GET("/telegram-ai-prompt", r.Dependency.SettingCont.GetClientTelegramAIPrompt)
			settingClientGroup.PATCH("/telegram-ai-prompt", r.Dependency.SettingCont.UpdateClientTelegramAIPrompt)
		}

		planGroup := generalGroup.Group("plans").Use(middleware)
		{
			planGroup.POST("", r.Dependency.PlanCont.Create)
			planGroup.PUT("/:plan_id", r.Dependency.PlanCont.Update)
			planGroup.PATCH("/:plan_id/toggle", r.Dependency.PlanCont.ToggleIsActive)
			planGroup.GET("", r.Dependency.PlanCont.GetAll)
			planGroup.GET("/:plan_id", r.Dependency.PlanCont.GetById)
			planGroup.DELETE("/:plan_id", r.Dependency.PlanCont.Delete)
		}

		approvalGroup := generalGroup.Group("approvals").Use(middleware)
		{
			approvalGroup.GET("", r.Dependency.ApprovalCont.GetAll)
			approvalGroup.PATCH("/:approval_id", r.Dependency.ApprovalCont.Approval)
			approvalGroup.DELETE("/:approval_id", r.Dependency.ApprovalCont.Delete)
		}

		// Payment
		paymentGroup := generalGroup.Group("payments")
		{
			// Platform (tenant beli plan Aidia)
			platformGroup := paymentGroup.Group("platform")
			{
				// Webhook tanpa middleware — validasi lewat Stripe-Signature header
				platformGroup.POST("/webhook", r.Dependency.PaymentCont.HandlePlatformWebhook)

				platformAuth := platformGroup.Use(middleware)
				{
					platformAuth.POST("/checkout/:plan_id", r.Dependency.PaymentCont.CreatePlatformCheckout)
					platformAuth.POST("/invoices/:invoice_id/pay", r.Dependency.PaymentCont.CreatePaymentFromExisting)
					platformAuth.GET("/invoices", r.Dependency.PaymentCont.GetPlatformInvoices)
					platformAuth.GET("/invoices/:invoice_id", r.Dependency.PaymentCont.GetPlatformInvoiceByID)
				}
			}

			// Client (customer bayar order tenant)
			clientGroup := paymentGroup.Group("client/:client_id")
			{
				// Webhook tanpa middleware — schema dari path param, validasi lewat Stripe-Signature header
				clientGroup.POST("/webhook/:schema", r.Dependency.PaymentCont.HandleClientWebhook)

				clientAuth := clientGroup.Use(middleware)
				{
					clientAuth.POST("/checkout/:order_id", r.Dependency.PaymentCont.CreateClientCheckout)
					clientAuth.GET("/invoices", r.Dependency.PaymentCont.GetClientInvoices)
				}
			}
		}

		// Client subs
		subs := generalGroup.Group("/subs")
		subs.Use(middleware)
		{
			subs.GET("/current", r.Dependency.SubsCont.GetCurrentSubs)
		}

		// Product Category
		productCategoryGroup := generalGroup.Group("client/:client_id/product-categories").Use(middleware)
		{
			productCategoryGroup.POST("", r.Dependency.ProductCategoryCont.Create)
			productCategoryGroup.PUT("/:category_id", r.Dependency.ProductCategoryCont.Update)
			productCategoryGroup.GET("", r.Dependency.ProductCategoryCont.GetAll)
			productCategoryGroup.GET("/:category_id", r.Dependency.ProductCategoryCont.GetByID)
			productCategoryGroup.DELETE("/:category_id", r.Dependency.ProductCategoryCont.Delete)
		}

		// Delivery setting
		deliveryGroup := generalGroup.Group("client/:client_id/delivery-settings").Use(middleware)
		{
			deliveryGroup.POST("", r.Dependency.DeliverySettingCont.Create)
			deliveryGroup.PUT("/:sub_group_name", r.Dependency.DeliverySettingCont.Update)
			deliveryGroup.GET("", r.Dependency.DeliverySettingCont.GetAll)
			deliveryGroup.GET("/:sub_group_name", r.Dependency.DeliverySettingCont.GetBySubGroupName)
			deliveryGroup.DELETE("/:sub_group_name", r.Dependency.DeliverySettingCont.Delete)
		}

		// Delivery avaibility setting
		deliveryAvailabilityGroup := generalGroup.Group("client/:client_id/delivery-availability-settings").Use(middleware)
		{
			deliveryAvailabilityGroup.POST("", r.Dependency.DeliveryAvailabilitySettingCont.Create)
			deliveryAvailabilityGroup.PUT("/:sub_group_name", r.Dependency.DeliveryAvailabilitySettingCont.Update)
			deliveryAvailabilityGroup.GET("", r.Dependency.DeliveryAvailabilitySettingCont.GetAll)
			deliveryAvailabilityGroup.GET("/:sub_group_name", r.Dependency.DeliveryAvailabilitySettingCont.GetBySubGroupName)
			deliveryAvailabilityGroup.DELETE("/:sub_group_name", r.Dependency.DeliveryAvailabilitySettingCont.Delete)
		}

		// Product
		productGroup := generalGroup.Group("client/:client_id/products").Use(middleware)
		{
			productGroup.POST("", r.Dependency.ProductCont.Create)
			productGroup.PUT("/:product_id", r.Dependency.ProductCont.Update)
			productGroup.GET("", r.Dependency.ProductCont.GetAll)
			productGroup.GET("/:product_id", r.Dependency.ProductCont.GetByID)
			productGroup.DELETE("/:product_id", r.Dependency.ProductCont.Delete)
		}

		// Customer
		customerGroup := generalGroup.Group("client/:client_id/customers").Use(middleware)
		{
			customerGroup.POST("", r.Dependency.CustomerCont.Create)
			customerGroup.GET("", r.Dependency.CustomerCont.GetAll)
			customerGroup.GET("/:customer_id", r.Dependency.CustomerCont.GetByID)
		}

		// Order
		orderGroup := generalGroup.Group("client/:client_id/orders").Use(middleware)
		{
			orderGroup.POST("", r.Dependency.OrderCont.Create)
			orderGroup.GET("", r.Dependency.OrderCont.GetAll)
			orderGroup.GET("/:order_id", r.Dependency.OrderCont.GetByID)
			orderGroup.PATCH("/:order_id/status", r.Dependency.OrderCont.UpdateStatus)
		}

		// Order Payment
		orderPaymentGroup := generalGroup.Group("client/:client_id/order-payments").Use(middleware)
		{
			orderPaymentGroup.GET("", r.Dependency.OrderPaymentCont.GetAll)
			orderPaymentGroup.GET("/:payment_id", r.Dependency.OrderPaymentCont.GetByID)
			orderPaymentGroup.PATCH("/:payment_id/status", r.Dependency.OrderPaymentCont.UpdateStatus)
		}

		kitchenGroup := generalGroup.Group("client/:client_id/kitchen-display")
		{
			kitchenGroup.GET("", r.Dependency.KitchenOrderCont.GetDisplay)
			kitchenGroup.GET("/stream", r.Dependency.KitchenOrderCont.Stream)
			kitchenGroup.OPTIONS("/stream", func(ctx *gin.Context) {
				ctx.Header("Access-Control-Allow-Origin", "*")
				ctx.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
				ctx.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept, X-Requested-With")
				ctx.Header("Access-Control-Max-Age", "86400")
				ctx.Header("Access-Control-Expose-Headers", "Content-Type")
				ctx.AbortWithStatus(204)
			})
			kitchenGroup.PATCH("/:kitchen_id/status", r.Dependency.KitchenOrderCont.UpdateStatus)
		}

		// Chat (Real-time conversations)
		chatGroup := generalGroup.Group("client/:client_id/chats").Use(middleware)
		{
			chatGroup.GET("", r.Dependency.ChatCont.GetConversations)
			chatGroup.GET("/stream", r.Dependency.ChatCont.Stream)
			chatGroup.GET("/:guest_id", r.Dependency.ChatCont.GetConversationDetail)
			chatGroup.POST("/:guest_id/messages", r.Dependency.ChatCont.SendManualReply)
		}

		// Telegram Bot Management
		telegramGroup := generalGroup.Group("client/:client_id/telegram").Use(middleware)
		{
			telegramGroup.PATCH("/bot-token", r.Dependency.SettingCont.UpdateTelegramBotToken)
			telegramGroup.POST("/request-phone", r.Dependency.TelegramCont.RequestPhone)
		}

		// Telegram Webhook (public - no middleware)
		telegramWebhookGroup := generalGroup.Group("webhook/telegram")
		{
			telegramWebhookGroup.POST("/:schema", r.Dependency.TelegramCont.Webhook)
		}

		// Internal API for n8n (no auth, can add internal API key middleware later)
		internalGroup := generalGroup.Group("internal")
		{
			internalGroup.GET("/telegram/:schema/ai-prompt", r.Dependency.TelegramCont.GetAIPromptForSchema)
		}
	}

	return &Router{
		Dependency: r.Dependency,
		Engine:     r.Engine,
	}
}
