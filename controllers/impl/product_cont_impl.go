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
// @Description  Buat produk baru beserta gambar dan kategori
// @Tags         Product
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        client_id        path      string    true  "Client ID"
// @Param        name             formData  string    true  "Nama produk"
// @Param        code             formData  string    true  "Kode produk"
// @Param        weight           formData  number    true  "Berat produk"
// @Param        price            formData  number    true  "Harga jual"
// @Param        original_price   formData  number    true  "Harga asli"
// @Param        description      formData  string    false "Deskripsi produk"
// @Param        delivery_sub_group_name      formData  string    true  "Delivery Sub Group Name (UUID)"
// @Param        is_out_of_stock  formData  boolean   false "Stok habis"
// @Param        category_ids     formData  []string  false "Category IDs (UUID)"
// @Param        images           formData  file      false "Gambar produk"
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
// @Description  Update produk beserta gambar dan kategori
// @Tags         Product
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        client_id        path      string    true  "Client ID"
// @Param        product_id       path      string    true  "Product ID"
// @Param        name             formData  string    true  "Nama produk"
// @Param        code             formData  string    true  "Kode produk"
// @Param        weight           formData  number    true  "Berat produk"
// @Param        price            formData  number    true  "Harga jual"
// @Param        original_price   formData  number    true  "Harga asli"
// @Param        description      formData  string    false "Deskripsi produk"
// @Param        delivery_sub_group_name      formData  string    true  "Delivery Sub Group Name (UUID)"
// @Param        is_out_of_stock  formData  boolean   false "Stok habis"
// @Param        is_active        formData  boolean   false "Status aktif"
// @Param        category_ids     formData  []string  false "Category IDs (UUID)"
// @Param        images           formData  file      false "Gambar produk baru (kosongkan jika tidak ingin mengubah)"
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
// @Description  Ambil semua produk dengan pagination
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
// @Description  Ambil detail produk berdasarkan ID
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
// @Description  Hapus produk beserta gambar dan kategori
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

	categoryIDs, err := parseUUIDs(ctx.PostFormArray("category_ids"))
	if err != nil {
		return nil, fmt.Errorf("invalid category_ids value")
	}

	return &reqProduct.CreateProductRequest{
		Name:          ctx.PostForm("name"),
		Weight:        weight,
		Price:         price,
		OriginalPrice: originalPrice,
		Description:   description,
		DeliveryID:    deliveryID,
		IsOutOfStock:  ctx.PostForm("is_out_of_stock") == "true",
		CategoryIDs:   categoryIDs,
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

	categoryIDs, err := parseUUIDs(ctx.PostFormArray("category_ids"))
	if err != nil {
		return nil, fmt.Errorf("invalid category_ids value")
	}

	return &reqProduct.UpdateProductRequest{
		Name:          ctx.PostForm("name"),
		Weight:        weight,
		Price:         price,
		OriginalPrice: originalPrice,
		Description:   description,
		DeliveryID:    deliveryID,
		IsOutOfStock:  ctx.PostForm("is_out_of_stock") == "true",
		IsActive:      ctx.PostForm("is_active") == "true",
		CategoryIDs:   categoryIDs,
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
