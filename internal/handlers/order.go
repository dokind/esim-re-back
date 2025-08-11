package handlers

import (
	"net/http"
	"strconv"

	"esim-platform/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type OrderHandler struct {
	orderService *services.OrderService
}

type CreateOrderRequest struct {
	ProductID string `json:"product_id" binding:"required"`
	// One of PackagePriceID (internal) or ProviderPriceID (upstream price_id) must be supplied to select package pricing
	PackagePriceID  *string  `json:"package_price_id"`
	ProviderPriceID *int     `json:"provider_price_id"`
	CustomerEmail   string   `json:"customer_email" binding:"required,email"`
	CustomerPhone   string   `json:"customer_phone"`
	UserID          *string  `json:"user_id"`
	CustomPriceUSD  *float64 `json:"custom_price_usd"`
}

type PaymentInitiationRequest struct {
	OrderNumber string `json:"order_number" binding:"required"`
}

func NewOrderHandler(orderService *services.OrderService) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
	}
}

// CreateOrder godoc
// @Summary Create new eSIM order
// @Description Create a new order for eSIM purchase
// @Tags Orders
// @Accept json
// @Produce json
// @Param order body CreateOrderRequest true "Order details (include package_price_id or provider_price_id)"
// @Success 201 {object} map[string]interface{} "Order created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid input"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /orders [post]
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Basic validation: require one of package identifiers
	if (req.PackagePriceID == nil || *req.PackagePriceID == "") && req.ProviderPriceID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "package_price_id or provider_price_id is required"})
		return
	}

	// Parse product ID
	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	// Parse user ID if provided
	var userID *uuid.UUID
	if req.UserID != nil {
		parsedUserID, err := uuid.Parse(*req.UserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}
		userID = &parsedUserID
	}

	// Create order request
	var packagePriceUUID *uuid.UUID
	if req.PackagePriceID != nil && *req.PackagePriceID != "" {
		ppid, err := uuid.Parse(*req.PackagePriceID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid package_price_id"})
			return
		}
		packagePriceUUID = &ppid
	}

	orderReq := services.CreateOrderRequest{ProductID: productID, PackagePriceID: packagePriceUUID, ProviderPriceID: req.ProviderPriceID, CustomerEmail: req.CustomerEmail, CustomerPhone: req.CustomerPhone, UserID: userID, CustomPriceUSD: req.CustomPriceUSD}

	order, err := h.orderService.CreateOrder(orderReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, order)
}

// GetOrder godoc
// @Summary Get order by order number
// @Description Retrieve order information by order number
// @Tags Orders
// @Produce json
// @Param orderNumber path string true "Order Number"
// @Success 200 {object} map[string]interface{} "Order information"
// @Failure 404 {object} map[string]interface{} "Order not found"
// @Router /orders/{orderNumber} [get]
func (h *OrderHandler) GetOrder(c *gin.Context) {
	orderNumber := c.Param("orderNumber")

	order, err := h.orderService.GetOrder(orderNumber)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	c.JSON(http.StatusOK, order)
}

// InitiatePayment godoc
// @Summary Initiate payment for order
// @Description Start payment process for an existing order
// @Tags Orders
// @Produce json
// @Param orderNumber path string true "Order Number"
// @Success 200 {object} map[string]interface{} "Payment initiation response"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /orders/{orderNumber}/payment [post]
func (h *OrderHandler) InitiatePayment(c *gin.Context) {
	orderNumber := c.Param("orderNumber")

	response, err := h.orderService.InitiatePayment(orderNumber)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetUserOrders godoc
// @Summary Get user orders
// @Description Retrieve orders for the authenticated user with pagination
// @Tags Orders
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} map[string]interface{} "User orders with pagination"
// @Failure 400 {object} map[string]interface{} "Invalid user ID"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security Bearer
// @Router /user/orders [get]
func (h *OrderHandler) GetUserOrders(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Parse user ID
	parsedUserID, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	orders, total, err := h.orderService.GetUserOrders(parsedUserID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
		"total":  total,
		"page":   page,
		"limit":  limit,
	})
}

// GetAllOrders godoc
// @Summary Get all orders (Admin)
// @Description Retrieve all orders with pagination and optional status filter (admin only)
// @Tags Orders,Admin
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param status query string false "Order status filter"
// @Success 200 {object} map[string]interface{} "All orders with pagination"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Security Bearer
// @Router /admin/orders [get]
func (h *OrderHandler) GetAllOrders(c *gin.Context) {
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

// GetOrderByID godoc
// @Summary Get order by ID (Admin)
// @Description Retrieve a specific order by its ID (admin only)
// @Tags Orders,Admin
// @Produce json
// @Param id path string true "Order ID (UUID)"
// @Success 200 {object} map[string]interface{} "Order information"
// @Failure 400 {object} map[string]interface{} "Invalid order ID"
// @Failure 501 {object} map[string]interface{} "Not implemented"
// @Security Bearer
// @Router /admin/orders/{id} [get]
func (h *OrderHandler) GetOrderByID(c *gin.Context) {
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

// UpdateOrderStatus godoc
// @Summary Update order status (Admin)
// @Description Update the status of a specific order (admin only)
// @Tags Orders,Admin
// @Accept json
// @Produce json
// @Param id path string true "Order ID (UUID)"
// @Param status body map[string]string true "Status update"
// @Success 200 {object} map[string]interface{} "Status updated"
// @Failure 400 {object} map[string]interface{} "Invalid order ID or request"
// @Failure 501 {object} map[string]interface{} "Not implemented"
// @Security Bearer
// @Router /admin/orders/{id}/status [put]
func (h *OrderHandler) UpdateOrderStatus(c *gin.Context) {
	orderID := c.Param("id")

	// Parse UUID
	_, err := uuid.Parse(orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// This would need to be implemented in the order service
	// For now, we'll return an error
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented"})
}
