package services

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"esim-platform/internal/config"
)

type RoamWiFiService struct {
	config      config.RoamWiFiConfig
	client      *http.Client
	token       string
	tokenExpiry time.Time
}

type RoamWiFiResponse struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// Login response structure
type LoginResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Token string `json:"token"`
	} `json:"data"`
}

// SKU response structures
type SKUInfo struct {
	SKUID       int    `json:"skuid"`
	Display     string `json:"display"`
	CountryCode string `json:"countryCode"`
}

type SKUListResponse struct {
	Code    string    `json:"code"`
	Message string    `json:"message"`
	Data    []SKUInfo `json:"data"`
}

type PackageInfo struct {
	PackageID   string  `json:"package_id"`
	PackageName string  `json:"package_name"`
	DataLimit   string  `json:"data_limit"`
	Validity    int     `json:"validity"`
	Price       float64 `json:"price"`
	Countries   string  `json:"countries"`
}

type OrderRequest struct {
	SKUID         string `json:"sku_id"`
	PackageID     string `json:"package_id"`
	CustomerEmail string `json:"customer_email"`
	CustomerPhone string `json:"customer_phone"`
	Quantity      int    `json:"quantity"`
}

type RoamWiFiOrderResponse struct {
	OrderID        string                 `json:"order_id"`
	Status         string                 `json:"status"`
	ESIMData       map[string]interface{} `json:"esim_data"`
	QRCode         string                 `json:"qr_code"`
	ActivationCode string                 `json:"activation_code"`
}

type OrderInfo struct {
	OrderID   string                 `json:"order_id"`
	Status    string                 `json:"status"`
	Amount    float64                `json:"amount"`
	CreatedAt string                 `json:"created_at"`
	ESIMData  map[string]interface{} `json:"esim_data"`
}

func NewRoamWiFiService(cfg config.RoamWiFiConfig) *RoamWiFiService {
	// Create a more basic HTTP client similar to curl/wget
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives:   true, // Disable connection pooling
			DisableCompression:  true, // Disable compression
			MaxIdleConns:        0,    // No idle connections
			MaxIdleConnsPerHost: 0,    // No idle connections per host
		},
	}

	return &RoamWiFiService{
		config: cfg,
		client: client,
	}
}

// generateSignature creates MD5 signature for authentication using RoamWiFi's method
func (r *RoamWiFiService) generateSignature(params map[string]string) string {
	signKey := "ro@mw1f1-bpm-ap1"
	var arrayList []string

	// Remove sign from params if it exists (create a copy to avoid modifying original)
	data := make(map[string]string)
	for k, v := range params {
		if k != "sign" {
			data[k] = v
		}
	}

	// Sort keys
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build array exactly like working code
	for _, key := range keys {
		arrayList = append(arrayList, key+"="+data[key])
	}

	// Concatenate and add sign key - exactly like working code
	var buffer strings.Builder
	buffer.WriteString(strings.Join(arrayList, "") + signKey)
	content := buffer.String()

	// Generate MD5 hash
	hash := md5.Sum([]byte(content))
	return hex.EncodeToString(hash[:])
}

// login authenticates with RoamWiFi API and gets token
func (r *RoamWiFiService) login() error {
	// Use the exact same URL pattern as working code
	loginURL := fmt.Sprintf("%s/api_order/login", r.config.APIURL)

	// Create parameters exactly as in the working code
	params := map[string]string{
		"phonenumber": r.config.PhoneNumber,
		"password":    r.config.Password,
	}

	// If credentials are empty, return an error immediately
	if r.config.PhoneNumber == "" || r.config.Password == "" {
		return fmt.Errorf("missing credentials: phonenumber='%s', password='%s'", r.config.PhoneNumber, r.config.Password)
	}

	// Generate signature exactly like working code
	signature := r.generateSignature(params)
	params["sign"] = signature

	// Build query string exactly like working code
	values := url.Values{}
	for k, v := range params {
		values.Add(k, v)
	}
	fullURL := loginURL + "?" + values.Encode()

	fmt.Printf("Login URL: %s\n", fullURL)

	// Make POST request exactly like working code
	resp, err := http.Post(fullURL, "application/x-www-form-urlencoded", nil)
	if err != nil {
		return fmt.Errorf("failed to make login request: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read login response body: %v", err)
	}

	fmt.Printf("Login Response: %s\n", string(body))

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to decode login response: %v", err)
	}

	// Check for successful login and extract token exactly like working code
	if dataField, ok := result["data"].(map[string]interface{}); ok {
		if token, exists := dataField["token"].(string); exists {
			r.token = token
			r.tokenExpiry = time.Now().Add(24 * time.Hour)
			return nil
		}
	}

	return fmt.Errorf("token not found in response: %v", result)
}

// ensureAuthenticated ensures we have a valid token
func (r *RoamWiFiService) ensureAuthenticated() error {
	// Always force a fresh login for debugging
	return r.login()
}

// GetSKUList retrieves the list of available eSIM SKUs from production API
func (r *RoamWiFiService) GetSKUList() ([]SKUInfo, error) {
	if err := r.ensureAuthenticated(); err != nil {
		return nil, fmt.Errorf("authentication failed: %v", err)
	}

	// Use the exact same URL pattern as working code
	apiURL := fmt.Sprintf("%s/api_esim/getSkus", r.config.APIURL)

	params := map[string]string{
		"token": r.token,
	}
	signature := r.generateSignature(params)
	params["sign"] = signature

	// Build query string exactly like working code
	values := url.Values{}
	for k, v := range params {
		values.Add(k, v)
	}
	fullURL := apiURL + "?" + values.Encode()

	fmt.Printf("GetSkus URL: %s\n", fullURL)

	// Make POST request exactly like working code
	resp, err := http.Post(fullURL, "application/x-www-form-urlencoded", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	fmt.Printf("SKU List API Response: %s\n", string(body))

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	// Check for successful response (code should be 0 for success)
	var codeStr string
	if code, ok := result["code"].(float64); ok {
		codeStr = fmt.Sprintf("%.0f", code)
	} else if code, ok := result["code"].(string); ok {
		codeStr = code
	} else {
		return nil, fmt.Errorf("unexpected code type: %T", result["code"])
	}

	if codeStr != "0" {
		if message, exists := result["message"].(string); exists {
			return nil, fmt.Errorf("API error: %s", message)
		}
		return nil, fmt.Errorf("API error with code: %v", result["code"])
	}

	// Parse data field
	dataField, ok := result["data"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected data format")
	}

	var skuList []SKUInfo
	for _, item := range dataField {
		if skuMap, ok := item.(map[string]interface{}); ok {
			sku := SKUInfo{}
			if skuID, exists := skuMap["skuid"]; exists {
				if skuIDFloat, ok := skuID.(float64); ok {
					sku.SKUID = int(skuIDFloat)
				}
			}
			if display, exists := skuMap["display"]; exists {
				if displayStr, ok := display.(string); ok {
					sku.Display = displayStr
				}
			}
			if countryCode, exists := skuMap["countryCode"]; exists {
				if countryCodeStr, ok := countryCode.(string); ok {
					sku.CountryCode = countryCodeStr
				}
			}
			skuList = append(skuList, sku)
		}
	}

	return skuList, nil
}

// GetSKUsByContinent retrieves SKUs grouped by continent from production API
func (r *RoamWiFiService) GetSKUsByContinent() ([]SKUInfo, error) {
	if err := r.ensureAuthenticated(); err != nil {
		return nil, fmt.Errorf("authentication failed: %v", err)
	}

	// Use the exact same URL pattern as working code
	apiURL := fmt.Sprintf("%s/api_esim/getSkuByGroup", r.config.APIURL)

	params := map[string]string{
		"token": r.token,
	}
	signature := r.generateSignature(params)
	params["sign"] = signature

	// Build URL with query parameters - POST request like working code
	values := url.Values{}
	for k, v := range params {
		values.Add(k, v)
	}
	fullURL := apiURL + "?" + values.Encode()

	req, err := http.NewRequest("POST", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	fmt.Printf("SKU By Continent API Response: %s\n", string(body))

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	// Check for successful response (code should be 0 for success)
	var codeStr string
	if code, ok := result["code"].(float64); ok {
		codeStr = fmt.Sprintf("%.0f", code)
	} else if code, ok := result["code"].(string); ok {
		codeStr = code
	} else {
		return nil, fmt.Errorf("unexpected code type: %T", result["code"])
	}

	if codeStr != "0" {
		if message, exists := result["message"].(string); exists {
			return nil, fmt.Errorf("API error: %s", message)
		}
		return nil, fmt.Errorf("API error with code: %v", result["code"])
	}

	// Parse data field
	dataField, ok := result["data"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected data format")
	}

	var skuList []SKUInfo
	for _, item := range dataField {
		if skuMap, ok := item.(map[string]interface{}); ok {
			sku := SKUInfo{}
			if skuID, exists := skuMap["skuid"]; exists {
				if skuIDFloat, ok := skuID.(float64); ok {
					sku.SKUID = int(skuIDFloat)
				}
			}
			if display, exists := skuMap["display"]; exists {
				if displayStr, ok := display.(string); ok {
					sku.Display = displayStr
				}
			}
			if countryCode, exists := skuMap["countryCode"]; exists {
				if countryCodeStr, ok := countryCode.(string); ok {
					sku.CountryCode = countryCodeStr
				}
			}
			skuList = append(skuList, sku)
		}
	}

	return skuList, nil
}

// GetPackagesBySKU retrieves available packages for a specific SKU
func (r *RoamWiFiService) GetPackagesBySKU(skuID string) ([]PackageInfo, error) {
	url := fmt.Sprintf("%s/sku/%s/packages", r.config.APIURL, skuID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.config.APIKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	var response RoamWiFiResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if response.Code != "200" {
		return nil, fmt.Errorf("API error: %s", response.Message)
	}

	// Parse the data field
	dataBytes, err := json.Marshal(response.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %v", err)
	}

	var packageList []PackageInfo
	if err := json.Unmarshal(dataBytes, &packageList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal package list: %v", err)
	}

	return packageList, nil
}

// CreateOrder creates a new eSIM order
func (r *RoamWiFiService) CreateOrder(orderReq OrderRequest) (*RoamWiFiOrderResponse, error) {
	if err := r.ensureAuthenticated(); err != nil {
		return nil, fmt.Errorf("authentication failed: %v", err)
	}

	// Use the exact same URL pattern as working code
	apiURL := fmt.Sprintf("%s/api_order/createOrder", r.config.APIURL)

	// Use the fields from OrderRequest struct
	quantityStr := strconv.Itoa(orderReq.Quantity)

	params := map[string]string{
		"token":    r.token,
		"skuid":    orderReq.SKUID,
		"quantity": quantityStr,
	}
	signature := r.generateSignature(params)
	params["sign"] = signature

	// Build URL with query parameters - POST request like working code
	values := url.Values{}
	for k, v := range params {
		values.Add(k, v)
	}
	fullURL := apiURL + "?" + values.Encode()

	req, err := http.NewRequest("POST", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	fmt.Printf("Create Order API Response: %s\n", string(body))

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	// Check for successful response (code should be 0 for success)
	var codeStr string
	if code, ok := result["code"].(float64); ok {
		codeStr = fmt.Sprintf("%.0f", code)
	} else if code, ok := result["code"].(string); ok {
		codeStr = code
	} else {
		return nil, fmt.Errorf("unexpected code type: %T", result["code"])
	}

	if codeStr != "0" {
		if message, exists := result["message"].(string); exists {
			return nil, fmt.Errorf("API error: %s", message)
		}
		return nil, fmt.Errorf("API error with code: %v", result["code"])
	}

	// Parse the response data
	orderResponse := &RoamWiFiOrderResponse{}
	if dataField, exists := result["data"].(map[string]interface{}); exists {
		if orderID, ok := dataField["order_id"].(string); ok {
			orderResponse.OrderID = orderID
		}
		if status, ok := dataField["status"].(string); ok {
			orderResponse.Status = status
		}
		if qrCode, ok := dataField["qr_code"].(string); ok {
			orderResponse.QRCode = qrCode
		}
		if activationCode, ok := dataField["activation_code"].(string); ok {
			orderResponse.ActivationCode = activationCode
		}
		if esimData, ok := dataField["esim_data"].(map[string]interface{}); ok {
			orderResponse.ESIMData = esimData
		}
	}

	return orderResponse, nil
}

// GetOrderInfo retrieves order information by order ID
func (r *RoamWiFiService) GetOrderInfo(orderID string) (*OrderInfo, error) {
	url := fmt.Sprintf("%s/order/%s", r.config.APIURL, orderID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.config.APIKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	var response RoamWiFiResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if response.Code != "200" {
		return nil, fmt.Errorf("API error: %s", response.Message)
	}

	// Parse the data field
	dataBytes, err := json.Marshal(response.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %v", err)
	}

	var orderInfo OrderInfo
	if err := json.Unmarshal(dataBytes, &orderInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal order info: %v", err)
	}

	return &orderInfo, nil
}

// GetOrderList retrieves the list of orders
func (r *RoamWiFiService) GetOrderList(page, limit int) ([]OrderInfo, error) {
	url := fmt.Sprintf("%s/orders?page=%d&limit=%d", r.config.APIURL, page, limit)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.config.APIKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	var response RoamWiFiResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if response.Code != "200" {
		return nil, fmt.Errorf("API error: %s", response.Message)
	}

	// Parse the data field
	dataBytes, err := json.Marshal(response.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %v", err)
	}

	var orderList []OrderInfo
	if err := json.Unmarshal(dataBytes, &orderList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal order list: %v", err)
	}

	return orderList, nil
}

// VerifyResources verifies if resources are available
func (r *RoamWiFiService) VerifyResources(skuID, packageID string) (bool, error) {
	url := fmt.Sprintf("%s/verify/resources", r.config.APIURL)

	reqBody := map[string]string{
		"sku_id":     skuID,
		"package_id": packageID,
	}

	reqBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return false, fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		return false, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.config.APIKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	var response RoamWiFiResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return false, fmt.Errorf("failed to decode response: %v", err)
	}

	if response.Code != "200" {
		return false, fmt.Errorf("API error: %s", response.Message)
	}

	// Parse the data field to check availability
	dataBytes, err := json.Marshal(response.Data)
	if err != nil {
		return false, fmt.Errorf("failed to marshal data: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(dataBytes, &result); err != nil {
		return false, fmt.Errorf("failed to unmarshal verification result: %v", err)
	}

	// Check if available field exists and is true
	if available, ok := result["available"].(bool); ok {
		return available, nil
	}

	return false, fmt.Errorf("invalid response format")
}

// SendPDFEmail sends PDF email with eSIM details
func (r *RoamWiFiService) SendPDFEmail(orderID, email string) error {
	url := fmt.Sprintf("%s/order/%s/send-pdf", r.config.APIURL, orderID)

	reqBody := map[string]string{
		"email": email,
	}

	reqBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.config.APIKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	var response RoamWiFiResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	if response.Code != "200" {
		return fmt.Errorf("API error: %s", response.Message)
	}

	return nil
}

// GetSKUByID retrieves a specific SKU by ID
func (r *RoamWiFiService) GetSKUByID(skuID string) (*SKUInfo, error) {
	url := fmt.Sprintf("%s/sku/%s", r.config.APIURL, skuID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.config.APIKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	var response RoamWiFiResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if response.Code != "200" {
		return nil, fmt.Errorf("API error: %s", response.Message)
	}

	// Convert response data to SKUInfo
	dataBytes, err := json.Marshal(response.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response data: %v", err)
	}

	var sku SKUInfo
	if err := json.Unmarshal(dataBytes, &sku); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SKU data: %v", err)
	}

	return &sku, nil
}
