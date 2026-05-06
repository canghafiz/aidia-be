package impl

import (
	"backend/exceptions"
	"backend/helpers"
	"backend/models/domains"
	reqProduct "backend/models/requests/product"
	"backend/models/services"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ProductContImpl struct {
	ProductServ services.ProductServ
}

func NewProductContImpl(productServ services.ProductServ) *ProductContImpl {
	return &ProductContImpl{ProductServ: productServ}
}

// Create @Summary      Create Product
// @Description  Create a new product along with images and categories
// @Tags         Product
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        client_id        path      string    true  "Client ID"
// @Param        name             formData  string    true  "Product name"
// @Param        weight           formData  number    true  "Product weight"
// @Param        price            formData  number    true  "Selling price"
// @Param        original_price   formData  number    true  "Original price"
// @Param        description      formData  string    false "Product description"
// @Param        delivery_sub_group_name      formData  string    true  "Delivery Sub Group Name (UUID)"
// @Param        is_out_of_stock  formData  boolean   false "Out of stock"
// @Param        category_ids     formData  []string  false "Category IDs (UUID)"
// @Param        images           formData  file      false "Product images"
// @Success      200              {object}  helpers.ApiResponse
// @Failure      400              {object}  helpers.ApiResponse
// @Failure      401              {object}  helpers.ApiResponse
// @Failure      500              {object}  helpers.ApiResponse
// @Router       /client/{client_id}/products [post]
func (cont *ProductContImpl) Create(ctx *gin.Context) {
	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	request, err := parseCreateProductForm(ctx)
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	form, err := ctx.MultipartForm()
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
	images := form.File["images"]

	if err := cont.ProductServ.Create(clientID, *request, images); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	response := helpers.ApiResponse{Success: true, Code: 200, Data: nil}
	if err := helpers.WriteToResponseBody(ctx, response.Code, response); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
}

// Update @Summary      Update Product
// @Description  Update a product along with images and categories
// @Tags         Product
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        client_id        path      string    true  "Client ID"
// @Param        product_id       path      string    true  "Product ID"
// @Param        name             formData  string    true  "Product name"
// @Param        weight           formData  number    true  "Product weight"
// @Param        price            formData  number    true  "Selling price"
// @Param        original_price   formData  number    true  "Original price"
// @Param        description      formData  string    false "Product description"
// @Param        delivery_sub_group_name      formData  string    true  "Delivery Sub Group Name (UUID)"
// @Param        is_out_of_stock  formData  boolean   false "Out of stock"
// @Param        is_active        formData  boolean   false "Active status"
// @Param        category_ids     formData  []string  false "Category IDs (UUID)"
// @Param        images           formData  file      false "New product images (leave empty to keep existing)"
// @Success      200              {object}  helpers.ApiResponse
// @Failure      400              {object}  helpers.ApiResponse
// @Failure      401              {object}  helpers.ApiResponse
// @Failure      500              {object}  helpers.ApiResponse
// @Router       /client/{client_id}/products/{product_id} [put]
func (cont *ProductContImpl) Update(ctx *gin.Context) {
	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	productID, err := helpers.ParseUUID(ctx, "product_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	request, err := parseUpdateProductForm(ctx)
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	form, err := ctx.MultipartForm()
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
	images := form.File["images"]

	if err := cont.ProductServ.Update(clientID, productID, *request, images); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	response := helpers.ApiResponse{Success: true, Code: 200, Data: nil}
	if err := helpers.WriteToResponseBody(ctx, response.Code, response); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
}

// GetAll @Summary      Get All Products
// @Description  Get all products with pagination
// @Tags         Product
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path   string  true  "Client ID"
// @Param        page       query  int     false "Page"
// @Param        limit      query  int     false "Limit"
// @Success      200        {object}  helpers.ApiResponse{data=pagination.Response}
// @Failure      401        {object}  helpers.ApiResponse
// @Failure      500        {object}  helpers.ApiResponse
// @Router       /client/{client_id}/products [get]
func (cont *ProductContImpl) GetAll(ctx *gin.Context) {
	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	pg := domains.ParsePagination(ctx)

	result, err := cont.ProductServ.GetAll(clientID, pg)
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

// GetByID @Summary      Get Product By ID
// @Description  Get product details by ID
// @Tags         Product
// @Produce      json
// @Security     BearerAuth
// @Param        client_id   path  string true "Client ID"
// @Param        product_id  path  string true "Product ID"
// @Success      200         {object}  helpers.ApiResponse
// @Failure      400         {object}  helpers.ApiResponse
// @Failure      401         {object}  helpers.ApiResponse
// @Failure      500         {object}  helpers.ApiResponse
// @Router       /client/{client_id}/products/{product_id} [get]
func (cont *ProductContImpl) GetByID(ctx *gin.Context) {
	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	productID, err := helpers.ParseUUID(ctx, "product_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	result, err := cont.ProductServ.GetByID(clientID, productID)
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

// Delete @Summary      Delete Product
// @Description  Delete a product along with its images and categories
// @Tags         Product
// @Produce      json
// @Security     BearerAuth
// @Param        client_id   path  string true "Client ID"
// @Param        product_id  path  string true "Product ID"
// @Success      200         {object}  helpers.ApiResponse
// @Failure      400         {object}  helpers.ApiResponse
// @Failure      401         {object}  helpers.ApiResponse
// @Failure      500         {object}  helpers.ApiResponse
// @Router       /client/{client_id}/products/{product_id} [delete]
func (cont *ProductContImpl) Delete(ctx *gin.Context) {
	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	productID, err := helpers.ParseUUID(ctx, "product_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	if err := cont.ProductServ.Delete(clientID, productID); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	response := helpers.ApiResponse{Success: true, Code: 200, Data: nil}
	if err := helpers.WriteToResponseBody(ctx, response.Code, response); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
}

// ============================================================
// HELPERS
// ============================================================

func parseCreateProductForm(ctx *gin.Context) (*reqProduct.CreateProductRequest, error) {
	weight, err := strconv.ParseFloat(ctx.PostForm("weight"), 64)
	if err != nil {
		return nil, fmt.Errorf("invalid weight value")
	}
	price, err := strconv.ParseFloat(ctx.PostForm("price"), 64)
	if err != nil {
		return nil, fmt.Errorf("invalid price value")
	}
	originalPrice, err := strconv.ParseFloat(ctx.PostForm("original_price"), 64)
	if err != nil {
		return nil, fmt.Errorf("invalid original_price value")
	}
	deliveryID, err := uuid.Parse(ctx.PostForm("delivery_sub_group_name"))
	if err != nil {
		return nil, fmt.Errorf("invalid delivery_sub_group_name value")
	}

	var description *string
	if d := ctx.PostForm("description"); d != "" {
		description = &d
	}

	productQuantity := 0
	if pqStr := ctx.PostForm("product_quantity"); pqStr != "" {
		if pq, e := strconv.Atoi(pqStr); e == nil && pq >= 0 {
			productQuantity = pq
		}
	}

	categoryIDs, err := parseUUIDs(ctx.PostFormArray("category_ids"))
	if err != nil {
		return nil, fmt.Errorf("invalid category_ids value")
	}

	tagIDs, err := parseUUIDs(ctx.PostFormArray("tag_ids"))
	if err != nil {
		return nil, fmt.Errorf("invalid tag_ids value")
	}

	return &reqProduct.CreateProductRequest{
		Name:            ctx.PostForm("name"),
		Weight:          weight,
		Price:           price,
		OriginalPrice:   originalPrice,
		Description:     description,
		DeliveryID:      deliveryID,
		IsOutOfStock:    ctx.PostForm("is_out_of_stock") == "true",
		ProductQuantity: productQuantity,
		CategoryIDs:     categoryIDs,
		TagIDs:          tagIDs,
	}, nil
}

func parseUpdateProductForm(ctx *gin.Context) (*reqProduct.UpdateProductRequest, error) {
	weight, err := strconv.ParseFloat(ctx.PostForm("weight"), 64)
	if err != nil {
		return nil, fmt.Errorf("invalid weight value")
	}
	price, err := strconv.ParseFloat(ctx.PostForm("price"), 64)
	if err != nil {
		return nil, fmt.Errorf("invalid price value")
	}
	originalPrice, err := strconv.ParseFloat(ctx.PostForm("original_price"), 64)
	if err != nil {
		return nil, fmt.Errorf("invalid original_price value")
	}
	deliveryID, err := uuid.Parse(ctx.PostForm("delivery_sub_group_name"))
	if err != nil {
		return nil, fmt.Errorf("invalid delivery_sub_group_name value")
	}

	var description *string
	if d := ctx.PostForm("description"); d != "" {
		description = &d
	}

	productQuantity := 0
	if pqStr := ctx.PostForm("product_quantity"); pqStr != "" {
		if pq, e := strconv.Atoi(pqStr); e == nil && pq >= 0 {
			productQuantity = pq
		}
	}

	categoryIDs, err := parseUUIDs(ctx.PostFormArray("category_ids"))
	if err != nil {
		return nil, fmt.Errorf("invalid category_ids value")
	}

	tagIDs, err := parseUUIDs(ctx.PostFormArray("tag_ids"))
	if err != nil {
		return nil, fmt.Errorf("invalid tag_ids value")
	}

	return &reqProduct.UpdateProductRequest{
		Name:            ctx.PostForm("name"),
		Weight:          weight,
		Price:           price,
		OriginalPrice:   originalPrice,
		Description:     description,
		DeliveryID:      deliveryID,
		IsOutOfStock:    ctx.PostForm("is_out_of_stock") == "true",
		IsActive:        ctx.PostForm("is_active") == "true",
		ProductQuantity: productQuantity,
		CategoryIDs:     categoryIDs,
		TagIDs:          tagIDs,
	}, nil
}

func parseUUIDs(ids []string) ([]uuid.UUID, error) {
	var result []uuid.UUID
	for _, id := range ids {
		parsed, err := uuid.Parse(id)
		if err != nil {
			return nil, err
		}
		result = append(result, parsed)
	}
	return result, nil
}
