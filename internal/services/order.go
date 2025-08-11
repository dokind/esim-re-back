package services

import (
	"encoding/json"
	"fmt"
	"time"

	"esim-platform/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrderService struct {
	db              *gorm.DB
	roamWiFiService *RoamWiFiService
	qpayService     *QPayService
}

type CreateOrderRequest struct {
	ProductID       uuid.UUID  `json:"product_id" binding:"required"`
	PackagePriceID  *uuid.UUID `json:"package_price_id"`
	ProviderPriceID *int       `json:"provider_price_id"`
	CustomerEmail   string     `json:"customer_email" binding:"required,email"`
	CustomerPhone   string     `json:"customer_phone"`
	UserID          *uuid.UUID `json:"user_id"`
	// CustomPriceUSD optional manual USD override of package effective price
	CustomPriceUSD *float64 `json:"custom_price_usd"`
}

type OrderResponse struct {
	ID            uuid.UUID            `json:"id"`
	OrderNumber   string               `json:"order_number"`
	Status        string               `json:"status"`
	Amount        float64              `json:"amount"`
	Currency      string               `json:"currency"`
	CustomerEmail string               `json:"customer_email"`
	CustomerPhone string               `json:"customer_phone"`
	Product       models.Product       `json:"product"`
	PackagePrice  *models.PackagePrice `json:"package_price,omitempty"`
	PaymentURL    string               `json:"payment_url,omitempty"`
	QRCode        string               `json:"qr_code,omitempty"`
	CreatedAt     time.Time            `json:"created_at"`
}

type PaymentInitiationRequest struct {
	OrderNumber string `json:"order_number" binding:"required"`
}

type PaymentInitiationResponse struct {
	OrderNumber string `json:"order_number"`
	PaymentURL  string `json:"payment_url"`
	QRCode      string `json:"qr_code"`
	InvoiceID   string `json:"invoice_id"`
}

func NewOrderService(db *gorm.DB, roamWiFiService *RoamWiFiService, qpayService *QPayService) *OrderService {
	return &OrderService{
		db:              db,
		roamWiFiService: roamWiFiService,
		qpayService:     qpayService,
	}
}

// CreateOrder creates a new order and initiates payment
func (o *OrderService) CreateOrder(req CreateOrderRequest) (*OrderResponse, error) {
	// Get product information
	var product models.Product
	if err := o.db.First(&product, req.ProductID).Error; err != nil {
		return nil, fmt.Errorf("product not found: %v", err)
	}

	// Determine package price selection
	var selectedPackage *models.PackagePrice
	if req.PackagePriceID != nil {
		var pp models.PackagePrice
		if err := o.db.First(&pp, *req.PackagePriceID).Error; err != nil {
			return nil, fmt.Errorf("package_price not found: %v", err)
		}
		selectedPackage = &pp
	} else if req.ProviderPriceID != nil {
		var pp models.PackagePrice
		if err := o.db.Where("provider_price_id = ?", *req.ProviderPriceID).First(&pp).Error; err != nil {
			return nil, fmt.Errorf("package_price not found for provider_price_id: %v", err)
		}
		selectedPackage = &pp
	} else {
		return nil, fmt.Errorf("package selection required")
	}

	// Validate selected package corresponds to product SKUID
	if selectedPackage != nil && selectedPackage.SKUID != product.SKUID {
		return nil, fmt.Errorf("selected package does not belong to product sku")
	}

	// Calculate final price: start from package effective USD price -> convert to MNT using current rate
	pricing := NewPricingService(o.db)
	usdToMnt, _ := pricing.GetUSDToMNTRate()
	finalPriceUSD := selectedPackage.EffectivePriceUSD
	if req.CustomPriceUSD != nil {
		finalPriceUSD = *req.CustomPriceUSD
	}
	finalPriceMNT := finalPriceUSD * usdToMnt

	// Generate order number
	orderNumber := o.qpayService.GenerateOrderNumber()

	// Create order in database
	order := models.Order{
		UserID:          req.UserID,
		ProductID:       req.ProductID,
		PackagePriceID:  &selectedPackage.ID,
		ProviderPriceID: &selectedPackage.ProviderPriceID,
		OrderNumber:     orderNumber,
		Status:          "pending",
		Amount:          finalPriceMNT,
		Currency:        "MNT",
		CustomerEmail:   req.CustomerEmail,
		CustomerPhone:   req.CustomerPhone,
	}

	if err := o.db.Create(&order).Error; err != nil {
		return nil, fmt.Errorf("failed to create order: %v", err)
	}

	// Create QPay invoice
	qpayAmount := o.qpayService.FormatAmount(finalPriceMNT)
	invoiceDescription := fmt.Sprintf("eSIM %s - %s (%s)", product.Name, product.DataLimit, selectedPackage.ShowName)

	qpayResponse, err := o.qpayService.CreateInvoice(
		orderNumber,
		invoiceDescription,
		req.CustomerEmail,
		qpayAmount,
	)
	if err != nil {
		// Update order status to failed
		o.db.Model(&order).Update("status", "failed")
		return nil, fmt.Errorf("failed to create QPay invoice: %v", err)
	}

	// Update order with QPay invoice ID
	o.db.Model(&order).Update("qpay_invoice_id", qpayResponse.Data.InvoiceID)

	// Create payment transaction record
	transactionData, _ := json.Marshal(map[string]interface{}{
		"qr_code": qpayResponse.Data.QRCode,
		"web_url": qpayResponse.Data.URLs.Web,
		"app_url": qpayResponse.Data.URLs.App,
	})

	paymentTransaction := models.PaymentTransaction{
		OrderID:           order.ID,
		QPayTransactionID: qpayResponse.Data.InvoiceID,
		Amount:            finalPriceMNT,
		Status:            "pending",
		PaymentMethod:     "qpay",
		TransactionData:   string(transactionData),
	}

	o.db.Create(&paymentTransaction)

	return &OrderResponse{
		ID:            order.ID,
		OrderNumber:   order.OrderNumber,
		Status:        order.Status,
		Amount:        order.Amount,
		Currency:      order.Currency,
		CustomerEmail: order.CustomerEmail,
		CustomerPhone: order.CustomerPhone,
		Product:       product,
		PackagePrice:  selectedPackage,
		PaymentURL:    qpayResponse.Data.URLs.Web,
		QRCode:        qpayResponse.Data.QRCode,
		CreatedAt:     order.CreatedAt,
	}, nil
}

// GetOrder retrieves order information
func (o *OrderService) GetOrder(orderNumber string) (*OrderResponse, error) {
	var order models.Order
	if err := o.db.Preload("Product").Preload("PackagePrice").Preload("PaymentTransactions").Where("order_number = ?", orderNumber).First(&order).Error; err != nil {
		return nil, fmt.Errorf("order not found: %v", err)
	}

	response := &OrderResponse{
		ID:            order.ID,
		OrderNumber:   order.OrderNumber,
		Status:        order.Status,
		Amount:        order.Amount,
		Currency:      order.Currency,
		CustomerEmail: order.CustomerEmail,
		CustomerPhone: order.CustomerPhone,
		Product:       order.Product,
		PackagePrice:  order.PackagePrice,
		CreatedAt:     order.CreatedAt,
	}

	// Add payment information if available
	if len(order.PaymentTransactions) > 0 {
		lastTransaction := order.PaymentTransactions[len(order.PaymentTransactions)-1]
		var transactionData map[string]interface{}
		if err := json.Unmarshal([]byte(lastTransaction.TransactionData), &transactionData); err == nil {
			if webURL, exists := transactionData["web_url"].(string); exists {
				response.PaymentURL = webURL
			}
			if qrCode, exists := transactionData["qr_code"].(string); exists {
				response.QRCode = qrCode
			}
		}
	}

	return response, nil
}

// InitiatePayment initiates payment for an existing order
func (o *OrderService) InitiatePayment(orderNumber string) (*PaymentInitiationResponse, error) {
	var order models.Order
	if err := o.db.Preload("Product").Preload("PackagePrice").Where("order_number = ?", orderNumber).First(&order).Error; err != nil {
		return nil, fmt.Errorf("order not found: %v", err)
	}

	if order.Status != "pending" {
		return nil, fmt.Errorf("order is not in pending status")
	}

	// Check if QPay invoice already exists
	if order.QPayInvoiceID != "" {
		// Check payment status
		paymentStatus, err := o.qpayService.CheckPayment(order.QPayInvoiceID)
		if err == nil && paymentStatus.Data.PaymentStatus == "PAID" {
			// Update order status
			o.db.Model(&order).Update("status", "paid")
			return nil, fmt.Errorf("payment already completed")
		}
	}

	// Create new QPay invoice
	qpayAmount := o.qpayService.FormatAmount(order.Amount)
	invoiceDescription := fmt.Sprintf("eSIM %s - %s", order.Product.Name, order.Product.DataLimit)

	qpayResponse, err := o.qpayService.CreateInvoice(
		orderNumber,
		invoiceDescription,
		order.CustomerEmail,
		qpayAmount,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create QPay invoice: %v", err)
	}

	// Update order with QPay invoice ID
	o.db.Model(&order).Update("qpay_invoice_id", qpayResponse.Data.InvoiceID)

	// Create or update payment transaction
	var paymentTransaction models.PaymentTransaction
	if err := o.db.Where("order_id = ?", order.ID).First(&paymentTransaction).Error; err != nil {
		// Create new transaction
		transactionData, _ := json.Marshal(map[string]interface{}{
			"qr_code": qpayResponse.Data.QRCode,
			"web_url": qpayResponse.Data.URLs.Web,
			"app_url": qpayResponse.Data.URLs.App,
		})
		paymentTransaction = models.PaymentTransaction{
			OrderID:           order.ID,
			QPayTransactionID: qpayResponse.Data.InvoiceID,
			Amount:            order.Amount,
			Status:            "pending",
			PaymentMethod:     "qpay",
			TransactionData:   string(transactionData),
		}
		o.db.Create(&paymentTransaction)
	} else {
		// Update existing transaction
		transactionData, _ := json.Marshal(map[string]interface{}{
			"qr_code": qpayResponse.Data.QRCode,
			"web_url": qpayResponse.Data.URLs.Web,
			"app_url": qpayResponse.Data.URLs.App,
		})
		paymentTransaction.QPayTransactionID = qpayResponse.Data.InvoiceID
		paymentTransaction.TransactionData = string(transactionData)
		o.db.Save(&paymentTransaction)
	}

	return &PaymentInitiationResponse{
		OrderNumber: orderNumber,
		PaymentURL:  qpayResponse.Data.URLs.Web,
		QRCode:      qpayResponse.Data.QRCode,
		InvoiceID:   qpayResponse.Data.InvoiceID,
	}, nil
}

// ProcessPaymentWebhook processes QPay webhook
func (o *OrderService) ProcessPaymentWebhook(webhookData *QPayWebhookData) error {
	// Find order by order number
	var order models.Order
	if err := o.db.Where("order_number = ?", webhookData.SenderInvoiceNo).First(&order).Error; err != nil {
		return fmt.Errorf("order not found: %v", err)
	}

	// Update order status
	paymentStatus := o.qpayService.GetPaymentStatus(webhookData.PaymentStatus)
	o.db.Model(&order).Update("status", paymentStatus)

	// Update or create payment transaction
	var paymentTransaction models.PaymentTransaction
	if err := o.db.Where("order_id = ?", order.ID).First(&paymentTransaction).Error; err != nil {
		// Create new transaction
		transactionData, _ := json.Marshal(map[string]interface{}{
			"invoice_id":     webhookData.InvoiceID,
			"payment_date":   webhookData.PaymentDate,
			"paid_amount":    webhookData.PaidAmount,
			"payment_status": webhookData.PaymentStatus,
		})
		paymentTransaction = models.PaymentTransaction{
			OrderID:           order.ID,
			QPayTransactionID: webhookData.TransactionID,
			Amount:            webhookData.Amount,
			Status:            paymentStatus,
			PaymentMethod:     "qpay",
			TransactionData:   string(transactionData),
		}
		o.db.Create(&paymentTransaction)
	} else {
		// Update existing transaction
		transactionData, _ := json.Marshal(map[string]interface{}{
			"invoice_id":     webhookData.InvoiceID,
			"payment_date":   webhookData.PaymentDate,
			"paid_amount":    webhookData.PaidAmount,
			"payment_status": webhookData.PaymentStatus,
		})
		paymentTransaction.QPayTransactionID = webhookData.TransactionID
		paymentTransaction.Status = paymentStatus
		paymentTransaction.TransactionData = string(transactionData)
		o.db.Save(&paymentTransaction)
	}

	// If payment is successful, create eSIM order with RoamWiFi
	if paymentStatus == "paid" {
		return o.createESIMOrder(&order)
	}

	return nil
}

// createESIMOrder creates eSIM order with RoamWiFi after successful payment
func (o *OrderService) createESIMOrder(order *models.Order) error {
	// Get product information
	var product models.Product
	if err := o.db.First(&product, order.ProductID).Error; err != nil {
		return fmt.Errorf("product not found: %v", err)
	}

	// Load package price if present
	var packagePrice models.PackagePrice
	if order.PackagePriceID != nil {
		if err := o.db.First(&packagePrice, *order.PackagePriceID).Error; err != nil {
			return fmt.Errorf("package_price not found: %v", err)
		}
	}

	// Create order request for RoamWiFi
	packageID := product.SKUID
	if order.ProviderPriceID != nil {
		packageID = fmt.Sprintf("%d", *order.ProviderPriceID)
	}
	orderReq := OrderRequest{SKUID: product.SKUID, PackageID: packageID, CustomerEmail: order.CustomerEmail, CustomerPhone: order.CustomerPhone, Quantity: 1}

	// Create order with RoamWiFi
	roamWiFiResponse, err := o.roamWiFiService.CreateOrder(orderReq)
	if err != nil {
		// Update order status to failed
		o.db.Model(order).Update("status", "failed")
		return fmt.Errorf("failed to create RoamWiFi order: %v", err)
	}

	// Update order with RoamWiFi order ID and eSIM data
	esimData, _ := json.Marshal(map[string]interface{}{
		"roamwifi_order_id": roamWiFiResponse.OrderID,
		"qr_code":           roamWiFiResponse.QRCode,
		"activation_code":   roamWiFiResponse.ActivationCode,
		"esim_data":         roamWiFiResponse.ESIMData,
	})

	o.db.Model(order).Updates(map[string]interface{}{
		"roamwifi_order_id": roamWiFiResponse.OrderID,
		"esim_data":         string(esimData),
		"status":            "completed",
	})

	// Send PDF email if email is provided
	if order.CustomerEmail != "" {
		go func() {
			o.roamWiFiService.SendPDFEmail(roamWiFiResponse.OrderID, order.CustomerEmail)
		}()
	}

	return nil
}

// GetUserOrders retrieves orders for a specific user
func (o *OrderService) GetUserOrders(userID uuid.UUID, page, limit int) ([]OrderResponse, int64, error) {
	var orders []models.Order
	var total int64

	offset := (page - 1) * limit

	// Get total count
	o.db.Model(&models.Order{}).Where("user_id = ?", userID).Count(&total)

	// Get orders with pagination
	if err := o.db.Preload("Product").Preload("PackagePrice").Where("user_id = ?", userID).
		Order("created_at DESC").Offset(offset).Limit(limit).Find(&orders).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get user orders: %v", err)
	}

	var responses []OrderResponse
	for _, order := range orders {
		response := OrderResponse{ID: order.ID, OrderNumber: order.OrderNumber, Status: order.Status, Amount: order.Amount, Currency: order.Currency, CustomerEmail: order.CustomerEmail, CustomerPhone: order.CustomerPhone, Product: order.Product, PackagePrice: order.PackagePrice, CreatedAt: order.CreatedAt}
		responses = append(responses, response)
	}

	return responses, total, nil
}

// GetAllOrders retrieves all orders with pagination (for admin)
func (o *OrderService) GetAllOrders(page, limit int, status string) ([]OrderResponse, int64, error) {
	var orders []models.Order
	var total int64

	offset := (page - 1) * limit
	query := o.db.Model(&models.Order{})

	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Get total count
	query.Count(&total)

	// Get orders with pagination
	if err := query.Preload("Product").Preload("PackagePrice").Preload("User").
		Order("created_at DESC").Offset(offset).Limit(limit).Find(&orders).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get orders: %v", err)
	}

	var responses []OrderResponse
	for _, order := range orders {
		response := OrderResponse{ID: order.ID, OrderNumber: order.OrderNumber, Status: order.Status, Amount: order.Amount, Currency: order.Currency, CustomerEmail: order.CustomerEmail, CustomerPhone: order.CustomerPhone, Product: order.Product, PackagePrice: order.PackagePrice, CreatedAt: order.CreatedAt}
		responses = append(responses, response)
	}

	return responses, total, nil
}
