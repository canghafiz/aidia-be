package impl

import (
	"backend/exceptions"
	"backend/helpers"
	reqTag "backend/models/requests/product_tag"
	"backend/models/services"

	"github.com/gin-gonic/gin"
)

type ProductTagContImpl struct {
	ProductTagServ services.ProductTagServ
}

func NewProductTagContImpl(productTagServ services.ProductTagServ) *ProductTagContImpl {
	return &ProductTagContImpl{ProductTagServ: productTagServ}
}

// Create @Summary      Create Product Tag
// @Tags         Product Tag
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path  string  true "Client ID"
// @Param        request    body  product_tag.CreateProductTagRequest true "Create Tag"
// @Success      200  {object}  helpers.ApiResponse
// @Router       /client/{client_id}/product-tags [post]
func (c *ProductTagContImpl) Create(ctx *gin.Context) {
	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
	var req reqTag.CreateProductTagRequest
	if err := helpers.ReadFromRequestBody(ctx, &req); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
	if err := c.ProductTagServ.Create(clientID, req); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
	resp := helpers.ApiResponse{Success: true, Code: 200, Data: nil}
	if err := helpers.WriteToResponseBody(ctx, resp.Code, resp); err != nil {
		exceptions.ErrorHandler(ctx, err)
	}
}

// Update @Summary      Update Product Tag
// @Tags         Product Tag
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path  string  true "Client ID"
// @Param        tag_id     path  string  true "Tag ID"
// @Param        request    body  product_tag.UpdateProductTagRequest true "Update Tag"
// @Success      200  {object}  helpers.ApiResponse
// @Router       /client/{client_id}/product-tags/{tag_id} [put]
func (c *ProductTagContImpl) Update(ctx *gin.Context) {
	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
	tagID, err := helpers.ParseUUID(ctx, "tag_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
	var req reqTag.UpdateProductTagRequest
	if err := helpers.ReadFromRequestBody(ctx, &req); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
	if err := c.ProductTagServ.Update(clientID, tagID, req); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
	resp := helpers.ApiResponse{Success: true, Code: 200, Data: nil}
	if err := helpers.WriteToResponseBody(ctx, resp.Code, resp); err != nil {
		exceptions.ErrorHandler(ctx, err)
	}
}

// GetAll @Summary      Get All Product Tags
// @Tags         Product Tag
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path  string  true "Client ID"
// @Success      200  {object}  helpers.ApiResponse{data=[]product_tag.ProductTagResponse}
// @Router       /client/{client_id}/product-tags [get]
func (c *ProductTagContImpl) GetAll(ctx *gin.Context) {
	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
	result, err := c.ProductTagServ.GetAll(clientID)
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
	resp := helpers.ApiResponse{Success: true, Code: 200, Data: result}
	if err := helpers.WriteToResponseBody(ctx, resp.Code, resp); err != nil {
		exceptions.ErrorHandler(ctx, err)
	}
}

// GetByID @Summary      Get Product Tag By ID
// @Tags         Product Tag
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path  string  true "Client ID"
// @Param        tag_id     path  string  true "Tag ID"
// @Success      200  {object}  helpers.ApiResponse{data=product_tag.ProductTagResponse}
// @Router       /client/{client_id}/product-tags/{tag_id} [get]
func (c *ProductTagContImpl) GetByID(ctx *gin.Context) {
	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
	tagID, err := helpers.ParseUUID(ctx, "tag_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
	result, err := c.ProductTagServ.GetByID(clientID, tagID)
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
	resp := helpers.ApiResponse{Success: true, Code: 200, Data: result}
	if err := helpers.WriteToResponseBody(ctx, resp.Code, resp); err != nil {
		exceptions.ErrorHandler(ctx, err)
	}
}

// Delete @Summary      Delete Product Tag
// @Tags         Product Tag
// @Produce      json
// @Security     BearerAuth
// @Param        client_id  path  string  true "Client ID"
// @Param        tag_id     path  string  true "Tag ID"
// @Success      200  {object}  helpers.ApiResponse
// @Router       /client/{client_id}/product-tags/{tag_id} [delete]
func (c *ProductTagContImpl) Delete(ctx *gin.Context) {
	clientID, err := helpers.ParseUUID(ctx, "client_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
	tagID, err := helpers.ParseUUID(ctx, "tag_id")
	if err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
	if err := c.ProductTagServ.Delete(clientID, tagID); err != nil {
		exceptions.ErrorHandler(ctx, err)
		return
	}
	resp := helpers.ApiResponse{Success: true, Code: 200, Data: nil}
	if err := helpers.WriteToResponseBody(ctx, resp.Code, resp); err != nil {
		exceptions.ErrorHandler(ctx, err)
	}
}
