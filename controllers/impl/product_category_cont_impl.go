package impl

import (
	"backend/exceptions"
	"backend/helpers"
	reqCat "backend/models/requests/product_category"
	"backend/models/services"

	"github.com/gin-gonic/gin"
)

type ProductCategoryContImpl struct {
	ProductCategoryServ services.ProductCategoryServ
}

func NewProductCategoryContImpl(productCategoryServ services.ProductCategoryServ) *ProductCategoryContImpl {
	return &ProductCategoryContImpl{ProductCategoryServ: productCategoryServ}
}

// Create @Summary      Create Product Category
// @Description  Buat kategori produk baru
// @Tags         Product Category
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path      string                                        true "Client ID"
// @Param        request    body      product_category.CreateProductCategoryRequest true "Create Product Category Request"
// @Success      200        {object}  helpers.ApiResponse
// @Failure      400        {object}  helpers.ApiResponse
// @Failure      401        {object}  helpers.ApiResponse
// @Failure      500        {object}  helpers.ApiResponse
// @Router       /client/{client_id}/product-categories [post]
func (cont *ProductCategoryContImpl) Create(ctx *gin.Context) {
	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	var request reqCat.CreateProductCategoryRequest
	if err := helpers.ReadFromRequestBody(ctx, &request); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	if err := cont.ProductCategoryServ.Create(clientID, request); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	response := helpers.ApiResponse{Success: true, Code: 200, Data: nil}
	if err := helpers.WriteToResponseBody(ctx, response.Code, response); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
}

// Update @Summary      Update Product Category
// @Description  Update kategori produk berdasarkan ID
// @Tags         Product Category
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        client_id    path      string                                        true "Client ID"
// @Param        category_id  path      string                                        true "Category ID"
// @Param        request      body      product_category.UpdateProductCategoryRequest true "Update Product Category Request"
// @Success      200          {object}  helpers.ApiResponse
// @Failure      400          {object}  helpers.ApiResponse
// @Failure      401          {object}  helpers.ApiResponse
// @Failure      500          {object}  helpers.ApiResponse
// @Router       /client/{client_id}/product-categories/{category_id} [put]
func (cont *ProductCategoryContImpl) Update(ctx *gin.Context) {
	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	categoryID, err := helpers.ParseUUID(ctx, "category_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	var request reqCat.UpdateProductCategoryRequest
	if err := helpers.ReadFromRequestBody(ctx, &request); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	if err := cont.ProductCategoryServ.Update(clientID, categoryID, request); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	response := helpers.ApiResponse{Success: true, Code: 200, Data: nil}
	if err := helpers.WriteToResponseBody(ctx, response.Code, response); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
}

// GetAll @Summary      Get All Product Categories
// @Description  Ambil semua kategori produk, filter berdasarkan visibility
// @Tags         Product Category
// @Produce      json
// @Security     BearerAuth
// @Param        client_id   path      string true  "Client ID"
// @Param        is_visible  query     bool   false "Filter by visibility (default: true)"
// @Success      200         {object}  helpers.ApiResponse{data=[]product_category.ProductCategoryResponse}
// @Failure      401         {object}  helpers.ApiResponse
// @Failure      500         {object}  helpers.ApiResponse
// @Router       /client/{client_id}/product-categories [get]
func (cont *ProductCategoryContImpl) GetAll(ctx *gin.Context) {
	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	isVisible := true
	if v := ctx.Query("is_visible"); v == "false" {
		isVisible = false
	}

	result, err := cont.ProductCategoryServ.GetAll(clientID, isVisible)
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

// GetByID @Summary      Get Product Category By ID
// @Description  Ambil detail kategori produk beserta produk yang berelasi
// @Tags         Product Category
// @Produce      json
// @Security     BearerAuth
// @Param        client_id    path      string true "Client ID"
// @Param        category_id  path      string true "Category ID"
// @Success      200          {object}  helpers.ApiResponse{data=product_category.ProductCategoryDetailResponse}
// @Failure      400          {object}  helpers.ApiResponse
// @Failure      401          {object}  helpers.ApiResponse
// @Failure      500          {object}  helpers.ApiResponse
// @Router       /client/{client_id}/product-categories/{category_id} [get]
func (cont *ProductCategoryContImpl) GetByID(ctx *gin.Context) {
	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	categoryID, err := helpers.ParseUUID(ctx, "category_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	result, err := cont.ProductCategoryServ.GetByID(clientID, categoryID)
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

// Delete @Summary      Delete Product Category
// @Description  Hapus kategori produk berdasarkan ID
// @Tags         Product Category
// @Produce      json
// @Security     BearerAuth
// @Param        client_id    path      string true "Client ID"
// @Param        category_id  path      string true "Category ID"
// @Success      200          {object}  helpers.ApiResponse
// @Failure      400          {object}  helpers.ApiResponse
// @Failure      401          {object}  helpers.ApiResponse
// @Failure      500          {object}  helpers.ApiResponse
// @Router       /client/{client_id}/product-categories/{category_id} [delete]
func (cont *ProductCategoryContImpl) Delete(ctx *gin.Context) {
	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	categoryID, err := helpers.ParseUUID(ctx, "category_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	if err := cont.ProductCategoryServ.Delete(clientID, categoryID); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}

	response := helpers.ApiResponse{Success: true, Code: 200, Data: nil}
	if err := helpers.WriteToResponseBody(ctx, response.Code, response); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
}
