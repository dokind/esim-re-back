package handlers

import (
	"encoding/json"
	"net/http"

	"esim-platform/internal/services"

	"github.com/gin-gonic/gin"
)

type WebhookHandler struct {
	orderService *services.OrderService
	qpayService  *services.QPayService
}

func NewWebhookHandler(orderService *services.OrderService, qpayService *services.QPayService) *WebhookHandler {
	return &WebhookHandler{
		orderService: orderService,
		qpayService:  qpayService,
	}
}

// HandleQPayWebhook godoc
// @Summary Handle QPay webhook
// @Description Process QPay webhook notifications for payment status updates
// @Tags Webhooks
// @Accept json
// @Produce json
// @Param webhook body map[string]interface{} true "QPay webhook data"
// @Success 200 {object} map[string]interface{} "Webhook processed successfully"
// @Failure 400 {object} map[string]interface{} "Invalid webhook data"
// @Failure 401 {object} map[string]interface{} "Invalid webhook signature"
// @Failure 500 {object} map[string]interface{} "Failed to process webhook"
// @Router /webhooks/qpay [post]
func (h *WebhookHandler) HandleQPayWebhook(c *gin.Context) {
	// Read the request body
	var webhookData map[string]interface{}
	if err := c.ShouldBindJSON(&webhookData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook data"})
		return
	}

	// Log webhook data for debugging
	// In production, you might want to log this to a file or monitoring service
	webhookBytes, _ := json.Marshal(webhookData)
	c.Header("X-Webhook-Data", string(webhookBytes))

	// Verify webhook signature (optional but recommended)
	signature := c.GetHeader("X-QPay-Signature")
	if signature != "" {
		if !h.qpayService.VerifyWebhookSignature(webhookData, signature) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid webhook signature"})
			return
		}
	}

	// Parse webhook data
	qpayWebhookData, err := h.qpayService.ParseWebhookData(webhookData)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse webhook data"})
		return
	}

	// Process the payment webhook
	if err := h.orderService.ProcessPaymentWebhook(qpayWebhookData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process payment webhook"})
		return
	}

	// Return success response
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Webhook processed successfully",
		"data": gin.H{
			"invoice_id":     qpayWebhookData.InvoiceID,
			"order_number":   qpayWebhookData.SenderInvoiceNo,
			"payment_status": qpayWebhookData.PaymentStatus,
		},
	})
}

// HandleRoamWiFiWebhook godoc
// @Summary Handle RoamWiFi webhook
// @Description Process RoamWiFi webhook notifications (not implemented)
// @Tags Webhooks
// @Accept json
// @Produce json
// @Param webhook body map[string]interface{} true "RoamWiFi webhook data"
// @Success 501 {object} map[string]interface{} "Not implemented"
// @Failure 400 {object} map[string]interface{} "Invalid webhook data"
// @Router /webhooks/roamwifi [post]
func (h *WebhookHandler) HandleRoamWiFiWebhook(c *gin.Context) {
	// This would handle webhooks from RoamWiFi if they provide them
	// For now, we'll return a not implemented response

	var webhookData map[string]interface{}
	if err := c.ShouldBindJSON(&webhookData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook data"})
		return
	}

	// Log webhook data
	webhookBytes, _ := json.Marshal(webhookData)
	c.Header("X-RoamWiFi-Webhook-Data", string(webhookBytes))

	// Process RoamWiFi webhook
	// This would need to be implemented based on RoamWiFi's webhook format
	c.JSON(http.StatusNotImplemented, gin.H{
		"status":  "not_implemented",
		"message": "RoamWiFi webhook processing not implemented",
		"data":    webhookData,
	})
}

// HealthCheck for webhook endpoints
func (h *WebhookHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "webhook_handler",
		"endpoints": gin.H{
			"qpay":     "/api/v1/webhooks/qpay",
			"roamwifi": "/api/v1/webhooks/roamwifi",
		},
	})
}
