package impl

import (
	"backend/helpers"
	"backend/models/domains"
	"backend/models/repositories"
	reqProduct "backend/models/requests/product"
	"backend/models/responses/pagination"
	resProduct "backend/models/responses/product"
	"backend/models/services"
	"fmt"
	"log"
	"mime/multipart"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProductServImpl struct {
	Db                  *gorm.DB
	Validator           *validator.Validate
	UserRepo            repositories.UsersRepo
	ProductRepo         repositories.ProductRepo
	DeliverySettingRepo repositories.DeliverySettingRepo
	FileServ            services.FileServ
}

func NewProductServImpl(
	db *gorm.DB,
	validator *validator.Validate,
	userRepo repositories.UsersRepo,
	productRepo repositories.ProductRepo,
	deliverySettingRepo repositories.DeliverySettingRepo,
	fileServ services.FileServ,
) *ProductServImpl {
	return &ProductServImpl{
		Db:                  db,
		Validator:           validator,
		UserRepo:            userRepo,
		ProductRepo:         productRepo,
		DeliverySettingRepo: deliverySettingRepo,
		FileServ:            fileServ,
	}
}

func (serv *ProductServImpl) getSchema(userID uuid.UUID) (string, error) {
	user, err := serv.UserRepo.GetByUserId(serv.Db, userID)
	if err != nil {
		return "", fmt.Errorf("user not found")
	}
	return user.Username, nil
}

func (serv *ProductServImpl) checkClientRole(userID uuid.UUID) error {
	role, err := serv.UserRepo.GetUserRole(serv.Db, userID)
	if err != nil {
		return fmt.Errorf("failed to get user role")
	}
	if role != "Client" {
		return fmt.Errorf("access denied")
	}
	return nil
}

func (serv *ProductServImpl) getDelivery(schema string, deliveryID uuid.UUID) *domains.DeliverySetting {
	settings, err := serv.DeliverySettingRepo.GetBySubGroupName(serv.Db, schema, deliveryID.String())
	if err != nil || len(settings) == 0 {
		return nil
	}
	deliveries := domains.ToDeliverySetting(settings)
	if len(deliveries) == 0 {
		return nil
	}
	return &deliveries[0]
}

func (serv *ProductServImpl) rollbackFiles(uploadedFiles []domains.File) {
	for _, f := range uploadedFiles {
		if errDel := serv.FileServ.DeleteFile(f.FileURL); errDel != nil {
			log.Printf("[FileServ].DeleteFile error: %v", errDel)
		}
	}
}

func (serv *ProductServImpl) Create(userID uuid.UUID, request reqProduct.CreateProductRequest, images []*multipart.FileHeader) error {
	if err := serv.checkClientRole(userID); err != nil {
		return err
	}

	if err := helpers.ErrValidator(request, serv.Validator); err != nil {
		return err
	}

	schema, err := serv.getSchema(userID)
	if err != nil {
		return err
	}

	var uploadedFiles []domains.File
	if len(images) > 0 {
		uploadedFiles, err = serv.FileServ.UploadFiles(images)
		if err != nil {
			return fmt.Errorf("failed to upload images: %w", err)
		}
	}

	tx := serv.Db.Begin()
	if tx.Error != nil {
		serv.rollbackFiles(uploadedFiles)
		return fmt.Errorf("failed to start transaction")
	}

	domain := reqProduct.CreateProductToDomain(request)
	product, err := serv.ProductRepo.Create(tx, schema, domain)
	if err != nil {
		tx.Rollback()
		serv.rollbackFiles(uploadedFiles)
		log.Printf("[ProductRepo].Create error: %v", err)
		return fmt.Errorf("failed to create product")
	}

	if len(uploadedFiles) > 0 {
		var productImages []domains.ProductImage
		for _, f := range uploadedFiles {
			productImages = append(productImages, domains.ProductImage{
				ProductID: product.ID,
				Image:     f.FileURL,
				IsActive:  true,
			})
		}
		if err := serv.ProductRepo.CreateImages(tx, schema, productImages); err != nil {
			tx.Rollback()
			serv.rollbackFiles(uploadedFiles)
			log.Printf("[ProductRepo].CreateImages error: %v", err)
			return fmt.Errorf("failed to create product images")
		}
	}

	if len(request.CategoryIDs) > 0 {
		var dtos []domains.ProductCategoryDto
		for _, catID := range request.CategoryIDs {
			dtos = append(dtos, domains.ProductCategoryDto{
				ProductID:  product.ID,
				CategoryID: catID,
			})
		}
		if err := serv.ProductRepo.CreateCategoryDtos(tx, schema, dtos); err != nil {
			tx.Rollback()
			serv.rollbackFiles(uploadedFiles)
			log.Printf("[ProductRepo].CreateCategoryDtos error: %v", err)
			return fmt.Errorf("failed to create product categories")
		}
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		serv.rollbackFiles(uploadedFiles)
		return fmt.Errorf("failed to commit transaction")
	}

	return nil
}

func (serv *ProductServImpl) Update(userID uuid.UUID, productID uuid.UUID, request reqProduct.UpdateProductRequest, images []*multipart.FileHeader) error {
	if err := serv.checkClientRole(userID); err != nil {
		return err
	}

	if err := helpers.ErrValidator(request, serv.Validator); err != nil {
		return err
	}

	schema, err := serv.getSchema(userID)
	if err != nil {
		return err
	}

	_, err = serv.ProductRepo.GetByID(serv.Db, schema, productID)
	if err != nil {
		return fmt.Errorf("product not found")
	}

	var uploadedFiles []domains.File
	if len(images) > 0 {
		uploadedFiles, err = serv.FileServ.UploadFiles(images)
		if err != nil {
			return fmt.Errorf("failed to upload images: %w", err)
		}
	}

	tx := serv.Db.Begin()
	if tx.Error != nil {
		serv.rollbackFiles(uploadedFiles)
		return fmt.Errorf("failed to start transaction")
	}

	domain := reqProduct.UpdateProductToDomain(request)
	domain.ID = productID
	if err := serv.ProductRepo.Update(tx, schema, domain); err != nil {
		tx.Rollback()
		serv.rollbackFiles(uploadedFiles)
		log.Printf("[ProductRepo].Update error: %v", err)
		return fmt.Errorf("failed to update product")
	}

	if len(images) > 0 {
		if err := serv.ProductRepo.DeleteImagesByProductID(tx, schema, productID); err != nil {
			tx.Rollback()
			serv.rollbackFiles(uploadedFiles)
			log.Printf("[ProductRepo].DeleteImagesByProductID error: %v", err)
			return fmt.Errorf("failed to delete old images")
		}

		var productImages []domains.ProductImage
		for _, f := range uploadedFiles {
			productImages = append(productImages, domains.ProductImage{
				ProductID: productID,
				Image:     f.FileURL,
				IsActive:  true,
			})
		}
		if err := serv.ProductRepo.CreateImages(tx, schema, productImages); err != nil {
			tx.Rollback()
			serv.rollbackFiles(uploadedFiles)
			log.Printf("[ProductRepo].CreateImages error: %v", err)
			return fmt.Errorf("failed to create new images")
		}
	}

	if err := serv.ProductRepo.DeleteCategoryDtosByProductID(tx, schema, productID); err != nil {
		tx.Rollback()
		serv.rollbackFiles(uploadedFiles)
		log.Printf("[ProductRepo].DeleteCategoryDtosByProductID error: %v", err)
		return fmt.Errorf("failed to delete old categories")
	}

	if len(request.CategoryIDs) > 0 {
		var dtos []domains.ProductCategoryDto
		for _, catID := range request.CategoryIDs {
			dtos = append(dtos, domains.ProductCategoryDto{
				ProductID:  productID,
				CategoryID: catID,
			})
		}
		if err := serv.ProductRepo.CreateCategoryDtos(tx, schema, dtos); err != nil {
			tx.Rollback()
			serv.rollbackFiles(uploadedFiles)
			log.Printf("[ProductRepo].CreateCategoryDtos error: %v", err)
			return fmt.Errorf("failed to create new categories")
		}
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		serv.rollbackFiles(uploadedFiles)
		return fmt.Errorf("failed to commit transaction")
	}

	return nil
}

func (serv *ProductServImpl) GetAll(userID uuid.UUID, pg domains.Pagination) (*pagination.Response, error) {
	if err := serv.checkClientRole(userID); err != nil {
		return nil, err
	}

	schema, err := serv.getSchema(userID)
	if err != nil {
		return nil, err
	}

	products, total, err := serv.ProductRepo.GetAll(serv.Db, schema, pg)
	if err != nil {
		log.Printf("[ProductRepo].GetAll error: %v", err)
		return nil, fmt.Errorf("failed to get products")
	}

	// Build delivery map dan categories map
	deliveryMap := map[string]*domains.DeliverySetting{}
	categoriesMap := map[string][]domains.ProductCategory{}

	for _, p := range products {
		deliveryID := p.DeliveryID.String()
		if _, exists := deliveryMap[deliveryID]; !exists {
			deliveryMap[deliveryID] = serv.getDelivery(schema, p.DeliveryID)
		}

		categories, err := serv.ProductRepo.GetCategoriesByProductID(serv.Db, schema, p.ID)
		if err != nil {
			log.Printf("[ProductRepo].GetCategoriesByProductID error: %v", err)
			categories = []domains.ProductCategory{}
		}
		categoriesMap[p.ID.String()] = categories
	}

	result := resProduct.ToProductPaginationResponse(products, deliveryMap, categoriesMap, total, pg.Page, pg.Limit)
	return &result, nil
}

func (serv *ProductServImpl) GetByID(userID uuid.UUID, productID uuid.UUID) (*resProduct.Response, error) {
	if err := serv.checkClientRole(userID); err != nil {
		return nil, err
	}

	schema, err := serv.getSchema(userID)
	if err != nil {
		return nil, err
	}

	product, err := serv.ProductRepo.GetByID(serv.Db, schema, productID)
	if err != nil {
		log.Printf("[ProductRepo].GetByID error: %v", err)
		return nil, fmt.Errorf("product not found")
	}

	delivery := serv.getDelivery(schema, product.DeliveryID)

	categories, err := serv.ProductRepo.GetCategoriesByProductID(serv.Db, schema, productID)
	if err != nil {
		log.Printf("[ProductRepo].GetCategoriesByProductID error: %v", err)
		categories = []domains.ProductCategory{}
	}

	response := resProduct.ToProductResponse(*product, delivery, categories)
	return &response, nil
}

func (serv *ProductServImpl) Delete(userID uuid.UUID, productID uuid.UUID) error {
	if err := serv.checkClientRole(userID); err != nil {
		return err
	}

	schema, err := serv.getSchema(userID)
	if err != nil {
		return err
	}

	product, err := serv.ProductRepo.GetByID(serv.Db, schema, productID)
	if err != nil {
		return fmt.Errorf("product not found")
	}

	tx := serv.Db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to start transaction")
	}

	if err := serv.ProductRepo.DeleteImagesByProductID(tx, schema, productID); err != nil {
		tx.Rollback()
		log.Printf("[ProductRepo].DeleteImagesByProductID error: %v", err)
		return fmt.Errorf("failed to delete product images")
	}

	if err := serv.ProductRepo.DeleteCategoryDtosByProductID(tx, schema, productID); err != nil {
		tx.Rollback()
		log.Printf("[ProductRepo].DeleteCategoryDtosByProductID error: %v", err)
		return fmt.Errorf("failed to delete product categories")
	}

	if err := serv.ProductRepo.Delete(tx, schema, productID); err != nil {
		tx.Rollback()
		log.Printf("[ProductRepo].Delete error: %v", err)
		return fmt.Errorf("failed to delete product")
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to commit transaction")
	}

	for _, img := range product.Images {
		if errDel := serv.FileServ.DeleteFile(img.Image); errDel != nil {
			log.Printf("[FileServ].DeleteFile error: %v", errDel)
		}
	}

	return nil
}
