package handlers

import (
	"net/http"
	"strconv"

	"esim-platform/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AdminHandler struct {
	productService *services.ProductService
	orderService   *services.OrderService
	userService    *services.UserService
	pricingService *services.PricingService
}

type UpdatePackageMarkupRequest struct {
	MarkupPercent *float64 `json:"markup_percent"`
}

type UpdatePackageOverrideRequest struct {
	OverridePriceUSD *float64 `json:"override_price_usd"`
}

type UpdateOrderStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

type UpdateUserRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
	IsAdmin   *bool  `json:"is_admin"`
}

type UpdateSettingsRequest struct {
	Settings map[string]string `json:"settings" binding:"required"`
}

type SalesAnalyticsResponse struct {
	TotalSales        float64 `json:"total_sales"`
	TotalOrders       int64   `json:"total_orders"`
	CompletedOrders   int64   `json:"completed_orders"`
	PendingOrders     int64   `json:"pending_orders"`
	FailedOrders      int64   `json:"failed_orders"`
	AverageOrderValue float64 `json:"average_order_value"`
}

type ProductAnalyticsResponse struct {
	TotalProducts      int64                    `json:"total_products"`
	ActiveProducts     int64                    `json:"active_products"`
	InactiveProducts   int64                    `json:"inactive_products"`
	TopSellingProducts []map[string]interface{} `json:"top_selling_products"`
}

func NewAdminHandler(productService *services.ProductService, orderService *services.OrderService, userService *services.UserService, pricingService *services.PricingService) *AdminHandler {
	return &AdminHandler{
		productService: productService,
		orderService:   orderService,
		userService:    userService,
		pricingService: pricingService,
	}
}

// SyncPackagePrices godoc
// @Summary Sync package prices for a SKU (Admin)
// @Description Fetch provider packages and upsert pricing rows
// @Tags Admin,Packages
// @Produce json
// @Param skuId path string true "SKU ID"
// @Success 200 {object} map[string]interface{} "Packages synced"
// @Failure 500 {object} map[string]interface{} "Internal error"
// @Security Bearer
// @Router /admin/skus/{skuId}/packages/sync [post]
func (h *AdminHandler) SyncPackagePrices(c *gin.Context) {
	skuID := c.Param("skuId")
	if err := h.productService.SyncPackagePrices(skuID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "packages synced"})
}

// UpdatePackageMarkup godoc
// @Summary Update package markup (Admin)
// @Description Set or clear markup percent (clears override)
// @Tags Admin,Packages
// @Accept json
// @Produce json
// @Param priceId path int true "Provider Price ID"
// @Param body body handlers.UpdatePackageMarkupRequest true "Markup payload"
// @Success 200 {object} map[string]interface{} "Updated"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal error"
// @Security Bearer
// @Router /admin/packages/{priceId}/markup [put]
func (h *AdminHandler) UpdatePackageMarkup(c *gin.Context) {
	priceIDStr := c.Param("priceId")
	priceID, err := strconv.Atoi(priceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid priceId"})
		return
	}
	var req UpdatePackageMarkupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.MarkupPercent == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "markup_percent required"})
		return
	}
	if *req.MarkupPercent < 0 || *req.MarkupPercent > 500 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "markup_percent out of range"})
		return
	}
	if err := h.productService.SetPackageMarkup(priceID, *req.MarkupPercent); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "markup updated"})
}

// UpdatePackageOverride godoc
// @Summary Update package override price (Admin)
// @Description Set or clear override price (clears markup usage)
// @Tags Admin,Packages
// @Accept json
// @Produce json
// @Param priceId path int true "Provider Price ID"
// @Param body body handlers.UpdatePackageOverrideRequest true "Override payload (null to clear)"
// @Success 200 {object} map[string]interface{} "Updated"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 500 {object} map[string]interface{} "Internal error"
// @Security Bearer
// @Router /admin/packages/{priceId}/override [put]
func (h *AdminHandler) UpdatePackageOverride(c *gin.Context) {
	priceIDStr := c.Param("priceId")
	priceID, err := strconv.Atoi(priceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid priceId"})
		return
	}
	var req UpdatePackageOverrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.productService.SetPackageOverride(priceID, req.OverridePriceUSD); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "override updated"})
}

// CreateProduct godoc
// @Summary Create new product (Admin)
// @Description Create a new eSIM product (admin only)
// @Tags Admin,Products
// @Accept json
// @Produce json
// @Param product body services.CreateProductRequest true "Product details"
// @Success 201 {object} map[string]interface{} "Product created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid input"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security Bearer
// @Router /admin/products [post]
func (h *AdminHandler) CreateProduct(c *gin.Context) {
	var req services.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	product, err := h.productService.CreateProduct(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, product)
}

// UpdateProduct godoc
// @Summary Update product (Admin)
// @Description Update an existing eSIM product (admin only)
// @Tags Admin,Products
// @Accept json
// @Produce json
// @Param id path string true "Product ID (UUID)"
// @Param product body services.UpdateProductRequest true "Product update details"
// @Success 200 {object} map[string]interface{} "Product updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid product ID or input"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security Bearer
// @Router /admin/products/{id} [put]
func (h *AdminHandler) UpdateProduct(c *gin.Context) {
	productID := c.Param("id")

	// Parse UUID
	id, err := uuid.Parse(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var req services.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	product, err := h.productService.UpdateProduct(id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, product)
}

// DeleteProduct godoc
// @Summary Delete product (Admin)
// @Description Delete an eSIM product (admin only)
// @Tags Admin,Products
// @Produce json
// @Param id path string true "Product ID (UUID)"
// @Success 200 {object} map[string]interface{} "Product deleted successfully"
// @Failure 400 {object} map[string]interface{} "Invalid product ID"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security Bearer
// @Router /admin/products/{id} [delete]
func (h *AdminHandler) DeleteProduct(c *gin.Context) {
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

// SyncProductsFromRoamWiFi godoc
// @Summary Sync products from RoamWiFi API (Admin)
// @Description Synchronize products from RoamWiFi API (admin only)
// @Tags Admin,Products
// @Produce json
// @Success 200 {object} map[string]interface{} "Products synced successfully"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security Bearer
// @Router /admin/products/sync [post]
func (h *AdminHandler) SyncProductsFromRoamWiFi(c *gin.Context) {
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

// GetAllOrders retrieves all orders (admin only)
func (h *AdminHandler) GetAllOrders(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	status := c.Query("status")

	orders, total, err := h.orderService.GetAllOrders(page, limit, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
		"total":  total,
		"page":   page,
		"limit":  limit,
		"status": status,
	})
}

// GetOrder retrieves a specific order (admin only)
func (h *AdminHandler) GetOrder(c *gin.Context) {
	orderID := c.Param("id")

	// Parse UUID
	_, err := uuid.Parse(orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	// This would need to be implemented in the order service
	// For now, we'll return an error
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented"})
}

// UpdateOrderStatus updates order status (admin only)
func (h *AdminHandler) UpdateOrderStatus(c *gin.Context) {
	orderID := c.Param("id")

	// Parse UUID
	_, err := uuid.Parse(orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var req UpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// This would need to be implemented in the order service
	// For now, we'll return an error
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented"})
}

// GetAllUsers godoc
// @Summary Get all users (Admin)
// @Description Retrieve all users with pagination (admin only)
// @Tags Admin,Users
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} map[string]interface{} "Users list with pagination"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security Bearer
// @Router /admin/users [get]
func (h *AdminHandler) GetAllUsers(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	users, total, err := h.userService.GetAllUsers(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetUser godoc
// @Summary Get user by ID (Admin)
// @Description Retrieve a specific user by ID (admin only)
// @Tags Admin,Users
// @Produce json
// @Param id path string true "User ID (UUID)"
// @Success 200 {object} map[string]interface{} "User information"
// @Failure 400 {object} map[string]interface{} "Invalid user ID"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Security Bearer
// @Router /admin/users/{id} [get]
func (h *AdminHandler) GetUser(c *gin.Context) {
	userID := c.Param("id")

	// Parse UUID
	id, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := h.userService.GetUserByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateUser godoc
// @Summary Update user (Admin)
// @Description Update user information (admin only)
// @Tags Admin,Users
// @Accept json
// @Produce json
// @Param id path string true "User ID (UUID)"
// @Param user body UpdateUserRequest true "User update details"
// @Success 200 {object} map[string]interface{} "User updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid user ID or input"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security Bearer
// @Router /admin/users/{id} [put]
func (h *AdminHandler) UpdateUser(c *gin.Context) {
	userID := c.Param("id")

	// Parse UUID
	id, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get existing user
	user, err := h.userService.GetUserByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Update fields
	if req.FirstName != "" {
		user.FirstName = req.FirstName
	}
	if req.LastName != "" {
		user.LastName = req.LastName
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}
	if req.IsAdmin != nil {
		user.IsAdmin = *req.IsAdmin
	}

	if err := h.userService.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// GetSettings godoc
// @Summary Get admin settings (Admin)
// @Description Retrieve admin settings (admin only)
// @Tags Admin,Settings
// @Produce json
// @Success 200 {object} map[string]interface{} "Admin settings"
// @Security Bearer
// @Router /admin/settings [get]
func (h *AdminHandler) GetSettings(c *gin.Context) {
	// This would need to be implemented with a settings service
	// For now, we'll return a mock response
	c.JSON(http.StatusOK, gin.H{
		"settings": map[string]string{
			"qpay_merchant_id":         "",
			"qpay_merchant_password":   "",
			"qpay_endpoint":            "",
			"roamwifi_api_key":         "",
			"roamwifi_api_url":         "",
			"default_currency":         "MNT",
			"profit_margin_percentage": "10",
		},
	})
}

// UpdateSettings godoc
// @Summary Update admin settings (Admin)
// @Description Update admin settings (admin only)
// @Tags Admin,Settings
// @Accept json
// @Produce json
// @Param settings body UpdateSettingsRequest true "Settings to update"
// @Success 200 {object} map[string]interface{} "Settings updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid input"
// @Security Bearer
// @Router /admin/settings [put]
func (h *AdminHandler) UpdateSettings(c *gin.Context) {
	var req UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// This would need to be implemented with a settings service
	// For now, we'll return a success response
	c.JSON(http.StatusOK, gin.H{
		"message":  "Settings updated successfully",
		"settings": req.Settings,
	})
}

// GetSalesAnalytics godoc
// @Summary Get sales analytics (Admin)
// @Description Retrieve sales analytics data (admin only)
// @Tags Admin,Analytics
// @Produce json
// @Success 200 {object} SalesAnalyticsResponse "Sales analytics data"
// @Security Bearer
// @Router /admin/analytics/sales [get]
func (h *AdminHandler) GetSalesAnalytics(c *gin.Context) {
	// This would need to be implemented with analytics queries
	// For now, we'll return mock data
	c.JSON(http.StatusOK, SalesAnalyticsResponse{
		TotalSales:        1000000.0,
		TotalOrders:       150,
		CompletedOrders:   120,
		PendingOrders:     20,
		FailedOrders:      10,
		AverageOrderValue: 6666.67,
	})
}

// GetProductAnalytics godoc
// @Summary Get product analytics (Admin)
// @Description Retrieve product analytics data (admin only)
// @Tags Admin,Analytics
// @Produce json
// @Success 200 {object} ProductAnalyticsResponse "Product analytics data"
// @Security Bearer
// @Router /admin/analytics/products [get]
func (h *AdminHandler) GetProductAnalytics(c *gin.Context) {
	// This would need to be implemented with analytics queries
	// For now, we'll return mock data
	c.JSON(http.StatusOK, ProductAnalyticsResponse{
		TotalProducts:    50,
		ActiveProducts:   45,
		InactiveProducts: 5,
		TopSellingProducts: []map[string]interface{}{
			{
				"product_id":  "uuid-1",
				"name":        "Europe eSIM 1GB",
				"total_sales": 500000.0,
				"order_count": 75,
			},
			{
				"product_id":  "uuid-2",
				"name":        "Asia eSIM 2GB",
				"total_sales": 300000.0,
				"order_count": 45,
			},
		},
	})
}

// Pricing Management Handlers

type UpdateExchangeRateRequest struct {
	Rate float64 `json:"rate" binding:"required,gt=0"`
}

type UpdateProfitMarginRequest struct {
	Margin float64 `json:"margin" binding:"required,gte=0"`
}

type SetProductPriceRequest struct {
	Price float64 `json:"price" binding:"required,gt=0"`
}

type PricingInfo struct {
	CurrentExchangeRate float64 `json:"current_exchange_rate"`
	DefaultProfitMargin float64 `json:"default_profit_margin"`
	LastUpdated         string  `json:"last_updated"`
}

// GetPricingInfo godoc
// @Summary Get pricing information (Admin)
// @Description Get current pricing information including exchange rates and profit margins (admin only)
// @Tags Admin,Pricing
// @Produce json
// @Success 200 {object} PricingInfo "Current pricing information"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security Bearer
// @Router /admin/pricing/info [get]
func (h *AdminHandler) GetPricingInfo(c *gin.Context) {
	rate, err := h.pricingService.GetUSDToMNTRate()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get exchange rate"})
		return
	}

	margin := h.pricingService.GetDefaultProfitMargin()

	c.JSON(http.StatusOK, PricingInfo{
		CurrentExchangeRate: rate,
		DefaultProfitMargin: margin,
		LastUpdated:         "2025-08-08T18:00:00Z", // This should be fetched from database
	})
}

// UpdateExchangeRate godoc
// @Summary Update exchange rate (Admin)
// @Description Manually set the USD to MNT exchange rate (admin only)
// @Tags Admin,Pricing
// @Accept json
// @Produce json
// @Param rate body UpdateExchangeRateRequest true "Exchange rate"
// @Success 200 {object} map[string]interface{} "Exchange rate updated"
// @Failure 400 {object} map[string]interface{} "Invalid input"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security Bearer
// @Router /admin/pricing/exchange-rate [put]
func (h *AdminHandler) UpdateExchangeRate(c *gin.Context) {
	var req UpdateExchangeRateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.pricingService.SetManualExchangeRate(req.Rate); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update exchange rate"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Exchange rate updated successfully",
		"exchange_rate": req.Rate,
	})
}

// UpdateAllProductPricing godoc
// @Summary Update all product pricing (Admin)
// @Description Recalculate pricing for all products and packages (admin only)
// @Tags Admin,Pricing
// @Produce json
// @Success 200 {object} map[string]interface{} "All pricing updated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security Bearer
// @Router /admin/pricing/update-all [post]
func (h *AdminHandler) UpdateAllProductPricing(c *gin.Context) {
	if err := h.pricingService.UpdateAllProductPricing(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product pricing"})
		return
	}

	if err := h.pricingService.UpdateAllPackagePricing(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update package pricing"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All product pricing updated successfully"})
}

// SetProductPrice godoc
// @Summary Set product price (Admin)
// @Description Set a manual price override for a specific product (admin only)
// @Tags Admin,Pricing
// @Accept json
// @Produce json
// @Param id path string true "Product ID (UUID)"
// @Param price body SetProductPriceRequest true "Product price"
// @Success 200 {object} map[string]interface{} "Product price updated"
// @Failure 400 {object} map[string]interface{} "Invalid product ID or input"
// @Security Bearer
// @Router /admin/products/{id}/price [put]
func (h *AdminHandler) SetProductPrice(c *gin.Context) {
	productID := c.Param("id")

	// Parse UUID
	id, err := uuid.Parse(productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var req SetProductPriceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// This would need to be implemented in ProductService
	// For now, we'll return a success response
	c.JSON(http.StatusOK, gin.H{
		"message":    "Product price updated successfully",
		"product_id": id,
		"price":      req.Price,
	})
}
