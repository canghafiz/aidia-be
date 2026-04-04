package dependencies

import (
	"backend/controllers"
	implCont "backend/controllers/impl"
	"backend/models/repositories"
	"backend/models/repositories/impl"
	implServ "backend/models/services/impl"

	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

type Dependency struct {
	Db     *gorm.DB
	JwtKey string

	UsersRepo                       repositories.UsersRepo
	UsersCont                       controllers.UsersCont
	RoleCont                        controllers.RoleCont
	SettingCont                     controllers.SettingCont
	PlanCont                        controllers.PlanCont
	ApprovalCont                    controllers.ApprovalCont
	PaymentCont                     controllers.PaymentCont
	SubsCont                        controllers.SubsCont
	ProductCategoryCont             controllers.ProductCategoryCont
	DeliverySettingCont             controllers.DeliverySettingCont
	DeliveryAvailabilitySettingCont controllers.DeliveryAvailabilitySettingCont
	ProductCont                     controllers.ProductCont
	CustomerCont                    controllers.CustomerCont
	OrderCont                       controllers.OrderCont
	OrderPaymentCont                controllers.OrderPaymentCont
	KitchenOrderCont                controllers.KitchenOrderCont
	ChatCont                        controllers.ChatCont
	TelegramCont                    controllers.TelegramCont
}

func NewDependency(db *gorm.DB, validator *validator.Validate, jwtKey string) *Dependency {
	// Repo
	userRepo := impl.NewUserRepoImpl()
	roleRepo := impl.NewRoleRepoImpl()
	settingRepo := impl.NewSettingRepoImpl()
	planRepo := impl.NewPlanRepoImpl()
	approvalRepo := impl.NewApprovalLogsRepoImpl()
	tenantRepo := impl.NewTenantRepoImpl()
	tenantPlanRepo := impl.NewTenantPlanRepoImpl()
	tenantUsageRepo := impl.NewTenantUsageRepoImpl()
	productCategoryRepo := impl.NewProductCategoryRepoImpl()
	deliverySettingRepo := impl.NewDeliverySettingRepoImpl()
	deliveryAvailabilitySettingRepo := impl.NewDeliveryAvailabilitySettingRepoImpl()
	productRepo := impl.NewProductRepoImpl()
	customerRepo := impl.NewCustomerRepoImpl()
	orderRepo := impl.NewOrderRepoImpl()
	orderPaymentRepo := impl.NewOrderPaymentRepoImpl()
	kitchenOrderRepo := impl.NewKitchenOrderRepoImpl()
	guestRepo := impl.NewGuestRepoImpl()
	guestMessageRepo := impl.NewGuestMessageRepoImpl()

	// Serv
	n8nServ := implServ.NewN8NServImpl()
	userServ := implServ.NewUsersServImpl(db, validator, userRepo, jwtKey)
	roleServ := implServ.NewRoleServImpl(db, roleRepo)
	settingServ := implServ.NewSettingServImpl(db, jwtKey, settingRepo)
	planServ := implServ.NewPlanServImpl(db, validator, planRepo, jwtKey)
	approvalServ := implServ.NewApprovalServImpl(db, jwtKey, approvalRepo, userRepo)
	paymentServ := implServ.NewPaymentServImpl(db, jwtKey, userRepo, tenantPlanRepo, planRepo, tenantRepo, settingRepo, orderPaymentRepo, orderRepo)
	subsServ := implServ.NewSubsServImpl(db, jwtKey, tenantRepo, tenantUsageRepo)
	productCategoryServ := implServ.NewProductCategoryServImpl(db, jwtKey, validator, userRepo, productCategoryRepo)
	deliverySettingServ := implServ.NewDeliverySettingServImpl(db, validator, userRepo, deliverySettingRepo)
	deliveryAvailabilitySettingServ := implServ.NewDeliveryAvailabilitySettingServImpl(db, validator, userRepo, deliveryAvailabilitySettingRepo, deliverySettingRepo)
	fileServ := implServ.NewFileServImpl()
	productServ := implServ.NewProductServImpl(db, validator, userRepo, productRepo, deliverySettingRepo, fileServ)
	customerServ := implServ.NewCustomerServImpl(db, validator, userRepo, customerRepo, jwtKey)
	orderServ := implServ.NewOrderServImpl(db, jwtKey, validator, userRepo, customerRepo, orderRepo, productRepo, deliverySettingRepo)
	orderPaymentServ := implServ.NewOrderPaymentServImpl(db, jwtKey, validator, userRepo, orderPaymentRepo)
	kitchenOrderServ := implServ.NewKitchenOrderServImpl(db, jwtKey, validator, userRepo, kitchenOrderRepo)
	chatServ := implServ.NewChatServImpl(db, jwtKey, guestRepo, guestMessageRepo, userRepo, settingRepo)

	return &Dependency{
		JwtKey: jwtKey,
		Db:     db,

		UsersRepo:                       userRepo,
		UsersCont:                       implCont.NewUsersContImpl(userServ),
		RoleCont:                        implCont.NewRoleContImpl(roleServ),
		SettingCont:                     implCont.NewSettingContImpl(settingServ, userRepo, db),
		PlanCont:                        implCont.NewPlanContImpl(planServ),
		ApprovalCont:                    implCont.NewApprovalCont(approvalServ),
		PaymentCont:                     implCont.NewPaymentContImpl(paymentServ),
		SubsCont:                        implCont.NewSubsContImpl(subsServ),
		ProductCategoryCont:             implCont.NewProductCategoryContImpl(productCategoryServ),
		DeliverySettingCont:             implCont.NewDeliverySettingContImpl(deliverySettingServ),
		DeliveryAvailabilitySettingCont: implCont.NewDeliveryAvailabilitySettingContImpl(deliveryAvailabilitySettingServ),
		ProductCont:                     implCont.NewProductContImpl(productServ),
		CustomerCont:                    implCont.NewCustomerContImpl(customerServ),
		OrderCont:                       implCont.NewOrderContImpl(orderServ),
		OrderPaymentCont:                implCont.NewOrderPaymentContImpl(orderPaymentServ),
		KitchenOrderCont:                implCont.NewKitchenOrderContImpl(kitchenOrderServ, userRepo, db),
		ChatCont:                        implCont.NewChatContImpl(chatServ, guestRepo, userRepo, db, jwtKey),
		TelegramCont:                    implCont.NewTelegramContImpl(guestRepo, guestMessageRepo, settingRepo, userRepo, productRepo, orderRepo, orderPaymentRepo, customerRepo, n8nServ, db),
	}
}
