package services

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"esim-platform/internal/config"
)

type QPayService struct {
	config config.QPayConfig
	client *http.Client
}

type QPayInvoiceRequest struct {
	MerchantID         string  `json:"merchant_id"`
	InvoiceCode        string  `json:"invoice_code"`
	SenderInvoiceNo    string  `json:"sender_invoice_no"`
	InvoiceReceiver    string  `json:"invoice_receiver"`
	InvoiceDescription string  `json:"invoice_description"`
	Amount             float64 `json:"amount"`
	CallbackURL        string  `json:"callback_url"`
}

type QPayInvoiceResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		InvoiceID string `json:"invoice_id"`
		QRCode    string `json:"qr_code"`
		URLs      struct {
			Web string `json:"web"`
			App string `json:"app"`
		} `json:"urls"`
	} `json:"data"`
}

type QPayCheckPaymentRequest struct {
	MerchantID    string `json:"merchant_id"`
	InvoiceID     string `json:"invoice_id"`
	CheckPassword string `json:"check_password"`
}

type QPayCheckPaymentResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		InvoiceID       string  `json:"invoice_id"`
		SenderInvoiceNo string  `json:"sender_invoice_no"`
		TransactionID   string  `json:"transaction_id"`
		PaymentStatus   string  `json:"payment_status"`
		Amount          float64 `json:"amount"`
		PaidAmount      float64 `json:"paid_amount"`
		PaymentDate     string  `json:"payment_date"`
	} `json:"data"`
}

type QPayWebhookData struct {
	InvoiceID       string  `json:"invoice_id"`
	SenderInvoiceNo string  `json:"sender_invoice_no"`
	TransactionID   string  `json:"transaction_id"`
	PaymentStatus   string  `json:"payment_status"`
	Amount          float64 `json:"amount"`
	PaidAmount      float64 `json:"paid_amount"`
	PaymentDate     string  `json:"payment_date"`
}

func NewQPayService(cfg config.QPayConfig) *QPayService {
	return &QPayService{
		config: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateInvoice creates a new QPay invoice
func (q *QPayService) CreateInvoice(orderNumber, description, customerEmail string, amount float64) (*QPayInvoiceResponse, error) {
	url := fmt.Sprintf("%s/invoice", q.config.Endpoint)

	// Generate invoice code with prefix and timestamp
	invoiceCode := fmt.Sprintf("%s_%d", q.config.InvoiceCode, time.Now().Unix())

	reqBody := QPayInvoiceRequest{
		MerchantID:         q.config.MerchantID,
		InvoiceCode:        invoiceCode,
		SenderInvoiceNo:    orderNumber,
		InvoiceReceiver:    customerEmail,
		InvoiceDescription: description,
		Amount:             amount,
		CallbackURL:        q.config.CallbackURL,
	}

	reqBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := q.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	var response QPayInvoiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if response.Code != 0 {
		return nil, fmt.Errorf("QPay API error: %s", response.Message)
	}

	return &response, nil
}

// CheckPayment checks the payment status of an invoice
func (q *QPayService) CheckPayment(invoiceID string) (*QPayCheckPaymentResponse, error) {
	url := fmt.Sprintf("%s/payment/check", q.config.Endpoint)

	// Generate check password (MD5 hash of QPay password)
	checkPassword := fmt.Sprintf("%x", md5.Sum([]byte(q.config.Password)))

	reqBody := QPayCheckPaymentRequest{
		MerchantID:    q.config.MerchantID,
		InvoiceID:     invoiceID,
		CheckPassword: checkPassword,
	}

	reqBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := q.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	var response QPayCheckPaymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if response.Code != 0 {
		return nil, fmt.Errorf("QPay API error: %s", response.Message)
	}

	return &response, nil
}

// VerifyWebhookSignature verifies the webhook signature from QPay
func (q *QPayService) VerifyWebhookSignature(data map[string]interface{}, signature string) bool {
	// QPay webhook verification logic
	// This would typically involve checking a signature or hash
	// For now, we'll implement a basic verification

	// Extract required fields for signature verification
	invoiceID, ok1 := data["invoice_id"].(string)
	amount, ok2 := data["amount"].(float64)
	paymentStatus, ok3 := data["payment_status"].(string)

	if !ok1 || !ok2 || !ok3 {
		return false
	}

	// Create signature string (this is a simplified version)
	signatureString := fmt.Sprintf("%s%.2f%s%s",
		invoiceID,
		amount,
		paymentStatus,
		q.config.Password)

	expectedSignature := fmt.Sprintf("%x", md5.Sum([]byte(signatureString)))

	return expectedSignature == signature
}

// ParseWebhookData parses webhook data from QPay
func (q *QPayService) ParseWebhookData(data map[string]interface{}) (*QPayWebhookData, error) {
	webhookData := &QPayWebhookData{}

	// Parse invoice_id
	if invoiceID, ok := data["invoice_id"].(string); ok {
		webhookData.InvoiceID = invoiceID
	} else {
		return nil, fmt.Errorf("invalid invoice_id")
	}

	// Parse sender_invoice_no
	if senderInvoiceNo, ok := data["sender_invoice_no"].(string); ok {
		webhookData.SenderInvoiceNo = senderInvoiceNo
	} else {
		return nil, fmt.Errorf("invalid sender_invoice_no")
	}

	// Parse transaction_id
	if transactionID, ok := data["transaction_id"].(string); ok {
		webhookData.TransactionID = transactionID
	} else {
		return nil, fmt.Errorf("invalid transaction_id")
	}

	// Parse payment_status
	if paymentStatus, ok := data["payment_status"].(string); ok {
		webhookData.PaymentStatus = paymentStatus
	} else {
		return nil, fmt.Errorf("invalid payment_status")
	}

	// Parse amount
	if amount, ok := data["amount"].(float64); ok {
		webhookData.Amount = amount
	} else {
		return nil, fmt.Errorf("invalid amount")
	}

	// Parse paid_amount
	if paidAmount, ok := data["paid_amount"].(float64); ok {
		webhookData.PaidAmount = paidAmount
	} else {
		return nil, fmt.Errorf("invalid paid_amount")
	}

	// Parse payment_date
	if paymentDate, ok := data["payment_date"].(string); ok {
		webhookData.PaymentDate = paymentDate
	} else {
		return nil, fmt.Errorf("invalid payment_date")
	}

	return webhookData, nil
}

// GetPaymentStatus returns the payment status based on QPay status
func (q *QPayService) GetPaymentStatus(qpayStatus string) string {
	switch qpayStatus {
	case "PAID":
		return "paid"
	case "PENDING":
		return "pending"
	case "FAILED":
		return "failed"
	case "CANCELLED":
		return "cancelled"
	default:
		return "unknown"
	}
}

// FormatAmount formats amount for QPay (in MNT, no decimals)
func (q *QPayService) FormatAmount(amount float64) float64 {
	// QPay expects amounts in MNT without decimals
	return float64(int(amount))
}

// GenerateOrderNumber generates a unique order number
func (q *QPayService) GenerateOrderNumber() string {
	timestamp := time.Now().Unix()
	random := time.Now().UnixNano() % 1000
	return fmt.Sprintf("ESIM%d%d", timestamp, random)
}
