package handlers

import (
	"net/http"
	"strconv"

	"esim-platform/internal/models"
	"esim-platform/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ProductHandler struct {
	productService *services.ProductService
}

type CreateProductRequest struct {
	SKUID        string   `json:"sku_id" binding:"required"`
	Name         string   `json:"name" binding:"required"`
	Description  string   `json:"description"`
	DataLimit    string   `json:"data_limit"`
	ValidityDays int      `json:"validity_days"`
	Countries    []string `json:"countries"`
	Continent    string   `json:"continent"`
	BasePrice    float64  `json:"base_price" binding:"required"`
	CustomPrice  *float64 `json:"custom_price"`
}

type UpdateProductRequest struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	DataLimit    string   `json:"data_limit"`
	ValidityDays int      `json:"validity_days"`
	Countries    []string `json:"countries"`
	Continent    string   `json:"continent"`
	BasePrice    float64  `json:"base_price"`
	CustomPrice  *float64 `json:"custom_price"`
	IsActive     *bool    `json:"is_active"`
}

type ProductResponse struct {
	ID           string   `json:"id"`
	SKUID        string   `json:"sku_id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	DataLimit    string   `json:"data_limit"`
	ValidityDays int      `json:"validity_days"`
	Countries    []string `json:"countries"`
	Continent    string   `json:"continent"`
	BasePrice    float64  `json:"base_price"`
	CustomPrice  *float64 `json:"custom_price"`
	PriceMNT     *float64 `json:"price_mnt"`
	DisplayPrice float64  `json:"display_price"`
	Currency     string   `json:"currency"`
	ExchangeRate *float64 `json:"exchange_rate,omitempty"`
	ProfitMargin *float64 `json:"profit_margin,omitempty"`
	IsActive     bool     `json:"is_active"`
	LastSyncedAt *string  `json:"last_synced_at,omitempty"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
}

func NewProductHandler(productService *services.ProductService) *ProductHandler {
	return &ProductHandler{
		productService: productService,
	}
}

// convertToProductResponse converts a Product model to ProductResponse
func (h *ProductHandler) convertToProductResponse(product models.Product) ProductResponse {
	response := ProductResponse{
		ID:           product.ID.String(),
		SKUID:        product.SKUID,
		Name:         product.Name,
		Description:  product.Description,
		DataLimit:    product.DataLimit,
		ValidityDays: product.ValidityDays,
		Countries:    product.Countries,
		Continent:    product.Continent,
		BasePrice:    product.BasePrice,
		CustomPrice:  product.CustomPrice,
		PriceMNT:     product.PriceMNT,
		DisplayPrice: product.GetDisplayPrice(),
		Currency:     "MNT", // Default to MNT for Mongolian users
		ExchangeRate: product.ExchangeRate,
		ProfitMargin: product.ProfitMargin,
		IsActive:     product.IsActive,
		CreatedAt:    product.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:    product.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if product.LastSyncedAt != nil {
		syncTime := product.LastSyncedAt.Format("2006-01-02T15:04:05Z")
		response.LastSyncedAt = &syncTime
	}

	return response
}

// GetProducts godoc
// @Summary Get all products
// @Description Retrieve list of all available eSIM products with MNT pricing
// @Tags Products
// @Produce json
// @Param continent query string false "Filter by continent" Enums(Asia, Europe, Africa, Americas, Oceania)
// @Param active query string false "Filter by active status" Enums(true, false)
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Number of items per page" default(20)
// @Success 200 {object} map[string]interface{} "List of products"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /products [get]
func (h *ProductHandler) GetProducts(c *gin.Context) {
	// Parse query parameters
	continent := c.Query("continent")
	active := c.Query("active")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	// Get products from service
	products, total, err := h.productService.GetProducts(page, limit, continent, active)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to response format with MNT pricing
	var productResponses []ProductResponse
	for _, product := range products {
		productResponses = append(productResponses, h.convertToProductResponse(product))
	}

	c.JSON(http.StatusOK, gin.H{
		"products": productResponses,
		"total":    total,
		"page":     page,
		"limit":    limit,
	})
}

// GetProductsByContinent retrieves products grouped by continent
func (h *ProductHandler) GetProductsByContinent(c *gin.Context) {
	products, err := h.productService.GetProductsByContinent()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert each continent's products to response format
	responseMap := make(map[string][]ProductResponse)
	for continent, continentProducts := range products {
		var productResponses []ProductResponse
		for _, product := range continentProducts {
			productResponses = append(productResponses, h.convertToProductResponse(product))
		}
		responseMap[continent] = productResponses
	}

	c.JSON(http.StatusOK, responseMap)
}

// GetProduct retrieves a specific product by ID
func (h *ProductHandler) GetProduct(c *gin.Context) {
	productID := c.Param("id")

	// Parse UUID
	id, err := uuid.Parse(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	product, err := h.productService.GetProduct(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	// Convert to response format with MNT pricing
	response := h.convertToProductResponse(*product)
	c.JSON(http.StatusOK, response)
}

// GetPackagesBySKU retrieves packages for a specific SKU
func (h *ProductHandler) GetPackagesBySKU(c *gin.Context) {
	skuID := c.Param("skuId")

	packages, err := h.productService.GetPackagesBySKU(skuID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, packages)
}

// CreateProduct creates a new product (admin only)
func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	serviceReq := services.CreateProductRequest{
		SKUID:        req.SKUID,
		Name:         req.Name,
		Description:  req.Description,
		DataLimit:    req.DataLimit,
		ValidityDays: req.ValidityDays,
		Countries:    req.Countries,
		Continent:    req.Continent,
		BasePrice:    req.BasePrice,
		CustomPrice:  req.CustomPrice,
	}

	product, err := h.productService.CreateProduct(serviceReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, product)
}

// UpdateProduct updates an existing product (admin only)
func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	productID := c.Param("id")

	// Parse UUID
	id, err := uuid.Parse(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	serviceReq := services.UpdateProductRequest{
		Name:         req.Name,
		Description:  req.Description,
		DataLimit:    req.DataLimit,
		ValidityDays: req.ValidityDays,
		Countries:    req.Countries,
		Continent:    req.Continent,
		BasePrice:    req.BasePrice,
		CustomPrice:  req.CustomPrice,
		IsActive:     req.IsActive,
	}

	product, err := h.productService.UpdateProduct(id, serviceReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, product)
}

// DeleteProduct deletes a product (admin only)
func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	productID := c.Param("id")

	// Parse UUID
	id, err := uuid.Parse(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	if err := h.productService.DeleteProduct(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product deleted successfully"})
}

// SyncProductsFromRoamWiFi syncs products from RoamWiFi API (admin only)
func (h *ProductHandler) SyncProductsFromRoamWiFi(c *gin.Context) {
	count, err := h.productService.SyncProductsFromRoamWiFi()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Products synced successfully",
		"count":   count,
	})
}
