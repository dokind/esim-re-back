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

type PackageInfo struct {
	PackageID   string  `json:"package_id"`
	PackageName string  `json:"package_name"`
	DataLimit   string  `json:"data_limit"`
	Validity    int     `json:"validity"`
	Price       float64 `json:"price"`
	Countries   string  `json:"countries"`
}

type SKUInfo struct {
	SKUID       int    `json:"sku_id"`
	Display     string `json:"display"`
	CountryCode string `json:"country_code"`
}

type OrderRequest struct {
	SKUID         string
	PackageID     string
	CustomerEmail string
	CustomerPhone string
	Quantity      int
}

type RoamWiFiOrderResponse struct {
	OrderID        string                 `json:"order_id"`
	Status         string                 `json:"status"`
	QRCode         string                 `json:"qr_code"`
	ActivationCode string                 `json:"activation_code"`
	ESIMData       map[string]interface{} `json:"esim_data"`
}

type OrderInfo struct {
	OrderID string `json:"order_id"`
	Status  string `json:"status"`
}

type RoamWiFiResponse struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func NewRoamWiFiService(cfg config.RoamWiFiConfig) *RoamWiFiService {
	client := &http.Client{Timeout: 30 * time.Second}
	return &RoamWiFiService{config: cfg, client: client}
}

// --- Detailed package response modeling (new) ---
type RoamWiFiPackageNetwork struct {
	Type     string `json:"type"`
	Operator string `json:"operator"`
	NameCN   string `json:"namecn"`
	NameEN   string `json:"nameen"`
}

type RoamWiFiPackage struct {
	APICode           string                   `json:"api_code"`
	Flows             float64                  `json:"flows"`
	Unit              string                   `json:"unit"`
	Days              int                      `json:"days"`
	Price             float64                  `json:"price"`
	PriceID           int                      `json:"price_id"`
	FlowType          int                      `json:"flow_type"`
	ShowName          string                   `json:"show_name"`
	PID               int                      `json:"pid"`
	Premark           string                   `json:"premark"`
	Overlay           int                      `json:"overlay"`
	ExpireDays        int                      `json:"expire_days"`
	Network           []RoamWiFiPackageNetwork `json:"network"`
	SupportDaypass    int                      `json:"support_daypass"`
	OpenCardFee       float64                  `json:"open_card_fee"`
	MinDay            int                      `json:"min_day"`
	SingleDiscountDay int                      `json:"single_discount_day"`
	SingleDiscount    int                      `json:"single_discount"`
	MaxDiscount       int                      `json:"max_discount"`
	MaxDay            int                      `json:"max_day"`
	MustDate          int                      `json:"must_date"`
	HadDaypassDetail  int                      `json:"had_daypass_detail"`
}

type RoamWiFiCountryImage struct {
	CountryCode int    `json:"country_code"`
	ImageURL    string `json:"image_url"`
	Name        string `json:"name"`
	NameEn      string `json:"name_en"`
}

type RoamWiFiPackagesResponse struct {
	SKUId          int                    `json:"sku_id"`
	Display        string                 `json:"display"`
	DisplayEn      string                 `json:"display_en"`
	CountryCode    string                 `json:"country_code"`
	SupportCountry []string               `json:"support_country"`
	ImageURL       string                 `json:"image_url"`
	CountryImages  []RoamWiFiCountryImage `json:"country_images"`
	Packages       []RoamWiFiPackage      `json:"packages"`
}

// GetPackagesDetailed returns rich provider data mapped into internal structs
func (r *RoamWiFiService) GetPackagesDetailed(skuID string) (*RoamWiFiPackagesResponse, error) {
	if err := r.ensureAuthenticated(); err != nil {
		return nil, fmt.Errorf("authentication failed: %v", err)
	}
	apiURL := fmt.Sprintf("%s/api_esim/getPackages", r.config.APIURL)
	params := map[string]string{"token": r.token, "skuId": skuID}
	params["sign"] = r.generateSignature(params)
	values := url.Values{}
	for k, v := range params {
		values.Add(k, v)
	}
	fullURL := apiURL + "?" + values.Encode()
	resp, err := http.Post(fullURL, "application/x-www-form-urlencoded", nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read failed: %v", err)
	}
	fmt.Printf("GetPackagesDetailed URL=%s RAW=%s\n", fullURL, string(body))
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode failed: %v", err)
	}
	code := fmt.Sprint(raw["code"])
	if code != "0" && code != "200" {
		return nil, fmt.Errorf("API error code=%s", code)
	}
	data, ok := raw["data"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected data structure")
	}
	respObj := &RoamWiFiPackagesResponse{}
	if v, ok := data["skuid"].(float64); ok {
		respObj.SKUId = int(v)
	}
	if v, ok := data["display"].(string); ok {
		respObj.Display = v
	}
	if v, ok := data["displayEn"].(string); ok {
		respObj.DisplayEn = v
	}
	if v, ok := data["countrycode"].(string); ok {
		respObj.CountryCode = v
	} else if v2, ok2 := data["countrycode"].(float64); ok2 {
		respObj.CountryCode = strconv.Itoa(int(v2))
	}
	if v, ok := data["imageUrl"].(string); ok {
		respObj.ImageURL = v
	}
	if arr, ok := data["supportCountry"].([]interface{}); ok {
		for _, c := range arr {
			if s, ok := c.(string); ok {
				respObj.SupportCountry = append(respObj.SupportCountry, s)
			}
		}
	}
	if imgs, ok := data["countryImageUrlDtoList"].([]interface{}); ok {
		for _, im := range imgs {
			if m, ok := im.(map[string]interface{}); ok {
				ci := RoamWiFiCountryImage{}
				if v, ok := m["countryCode"].(float64); ok {
					ci.CountryCode = int(v)
				}
				if v, ok := m["imageUrl"].(string); ok {
					ci.ImageURL = v
				}
				if v, ok := m["name"].(string); ok {
					ci.Name = v
				}
				if v, ok := m["nameEn"].(string); ok {
					ci.NameEn = v
				}
				respObj.CountryImages = append(respObj.CountryImages, ci)
			}
		}
	}
	// packages
	if list, ok := data["esimPackageDtoList"].([]interface{}); ok {
		for _, item := range list {
			if m, ok := item.(map[string]interface{}); ok {
				p := RoamWiFiPackage{}
				if v, ok := m["apiCode"].(string); ok {
					p.APICode = v
				}
				if v, ok := m["flows"].(float64); ok {
					p.Flows = v
				}
				if v, ok := m["unit"].(string); ok {
					p.Unit = v
				}
				if v, ok := m["days"].(float64); ok {
					p.Days = int(v)
				}
				if v, ok := m["price"].(float64); ok {
					p.Price = v
				}
				if v, ok := m["priceid"].(float64); ok {
					p.PriceID = int(v)
				}
				if v, ok := m["flowType"].(float64); ok {
					p.FlowType = int(v)
				}
				if v, ok := m["showName"].(string); ok {
					p.ShowName = v
				}
				if v, ok := m["pid"].(float64); ok {
					p.PID = int(v)
				}
				if v, ok := m["premark"].(string); ok {
					p.Premark = v
				}
				if v, ok := m["overlay"].(float64); ok {
					p.Overlay = int(v)
				}
				if v, ok := m["expireDays"].(float64); ok {
					p.ExpireDays = int(v)
				}
				if v, ok := m["supportDaypass"].(float64); ok {
					p.SupportDaypass = int(v)
				}
				if v, ok := m["openCardFee"].(float64); ok {
					p.OpenCardFee = v
				}
				if v, ok := m["minDay"].(float64); ok {
					p.MinDay = int(v)
				}
				if v, ok := m["singleDiscountDay"].(float64); ok {
					p.SingleDiscountDay = int(v)
				}
				if v, ok := m["singleDiscount"].(float64); ok {
					p.SingleDiscount = int(v)
				}
				if v, ok := m["maxDiscount"].(float64); ok {
					p.MaxDiscount = int(v)
				}
				if v, ok := m["maxDay"].(float64); ok {
					p.MaxDay = int(v)
				}
				if v, ok := m["mustDate"].(float64); ok {
					p.MustDate = int(v)
				}
				if v, ok := m["hadDaypassDetail"].(float64); ok {
					p.HadDaypassDetail = int(v)
				}
				if nets, ok := m["networkDtoList"].([]interface{}); ok {
					for _, n := range nets {
						if nm, ok := n.(map[string]interface{}); ok {
							nw := RoamWiFiPackageNetwork{}
							if v, ok := nm["type"].(string); ok {
								nw.Type = v
							}
							if v, ok := nm["operator"].(string); ok {
								nw.Operator = v
							}
							if v, ok := nm["namecn"].(string); ok {
								nw.NameCN = v
							}
							if v, ok := nm["nameen"].(string); ok {
								nw.NameEN = v
							}
							p.Network = append(p.Network, nw)
						}
					}
				}
				respObj.Packages = append(respObj.Packages, p)
			}
		}
	}
	return respObj, nil
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

	apiURL := fmt.Sprintf("%s/api_esim/getSkus", r.config.APIURL)
	params := map[string]string{"token": r.token}
	params["sign"] = r.generateSignature(params)
	values := url.Values{}
	for k, v := range params {
		values.Add(k, v)
	}
	fullURL := apiURL + "?" + values.Encode()
	fmt.Printf("GetSkus URL: %s\n", fullURL)
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
	var codeStr string
	switch v := result["code"].(type) {
	case float64:
		codeStr = fmt.Sprintf("%.0f", v)
	case string:
		codeStr = v
	default:
		return nil, fmt.Errorf("unexpected code type: %T", v)
	}
	if codeStr != "0" {
		if msg, ok := result["message"].(string); ok {
			return nil, fmt.Errorf("API error: %s", msg)
		}
		return nil, fmt.Errorf("API error code=%s body=%v", codeStr, result)
	}
	arr, ok := result["data"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected data format")
	}
	var skuList []SKUInfo
	for _, item := range arr {
		if m, ok := item.(map[string]interface{}); ok {
			sku := SKUInfo{}
			if v, ok := m["skuid"].(float64); ok {
				sku.SKUID = int(v)
			}
			if v, ok := m["display"].(string); ok {
				sku.Display = v
			}
			if v, ok := m["countryCode"].(string); ok {
				sku.CountryCode = v
			}
			skuList = append(skuList, sku)
		}
	}
	return skuList, nil
}

// GetPackagesBySKU retrieves available packages for a specific SKU (legacy signed API)
func (r *RoamWiFiService) GetPackagesBySKU(skuID string) ([]PackageInfo, error) {
	if err := r.ensureAuthenticated(); err != nil {
		return nil, fmt.Errorf("authentication failed: %v", err)
	}
	apiURL := fmt.Sprintf("%s/api_esim/getPackages", r.config.APIURL)
	params := map[string]string{"token": r.token, "skuId": skuID}
	params["sign"] = r.generateSignature(params)
	values := url.Values{}
	for k, v := range params {
		values.Add(k, v)
	}
	fullURL := apiURL + "?" + values.Encode()
	resp, err := http.Post(fullURL, "application/x-www-form-urlencoded", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}
	fmt.Printf("GetPackages URL=%s RAW=%s\n", fullURL, string(body))
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}
	codeVal := fmt.Sprint(result["code"])
	if codeVal != "0" && codeVal != "200" {
		if msg, ok := result["message"].(string); ok {
			return nil, fmt.Errorf("API error: %s", msg)
		}
		return nil, fmt.Errorf("API error code=%s body=%s", codeVal, string(body))
	}
	dataObj, ok := result["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected data format: data not object keys=%v", keysOf(result))
	}
	list, ok := dataObj["esimPackageDtoList"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("missing esimPackageDtoList in data keys=%v", keysOf(dataObj))
	}
	var countries string
	if sc, ok := dataObj["supportCountry"].([]interface{}); ok {
		var cs []string
		for _, c := range sc {
			if s, ok := c.(string); ok {
				cs = append(cs, s)
			}
		}
		countries = strings.Join(cs, ",")
	}
	pkgs := make([]PackageInfo, 0, len(list))
	for _, raw := range list {
		pm, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		pkg := PackageInfo{}
		if v, ok := pm["apiCode"].(string); ok {
			pkg.PackageID = v
		}
		if pkg.PackageID == "" {
			if v, ok := pm["priceid"].(float64); ok {
				pkg.PackageID = fmt.Sprintf("priceid-%d", int(v))
			}
		}
		if pkg.PackageID == "" {
			if v, ok := pm["pid"].(float64); ok {
				pkg.PackageID = fmt.Sprintf("pid-%d", int(v))
			}
		}
		if v, ok := pm["showName"].(string); ok && v != "" {
			pkg.PackageName = v
		}
		if pkg.PackageName == "" {
			if v, ok := pm["premark"].(string); ok {
				pkg.PackageName = truncatePremark(v)
			}
		}
		if flows, ok := pm["flows"].(float64); ok {
			if unit, ok := pm["unit"].(string); ok {
				pkg.DataLimit = fmt.Sprintf("%d%s", int(flows), unit)
			}
		}
		if v, ok := pm["days"].(float64); ok {
			pkg.Validity = int(v)
		}
		if v, ok := pm["price"].(float64); ok {
			pkg.Price = v
		}
		pkg.Countries = countries
		pkgs = append(pkgs, pkg)
	}
	return pkgs, nil
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

// GetPackagesBySKUBearer retains the newer bearer-based implementation for potential future use
func (r *RoamWiFiService) GetPackagesBySKUBearer(skuID string) ([]PackageInfo, error) {
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
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}
	// Expect code 200 here
	var codeStr string
	if code, ok := result["code"].(float64); ok {
		codeStr = fmt.Sprintf("%.0f", code)
	} else if code, ok := result["code"].(string); ok {
		codeStr = code
	}
	if codeStr != "200" {
		return nil, fmt.Errorf("API error code=%s body=%s", codeStr, string(body))
	}
	dataBytes, _ := json.Marshal(result["data"])
	var packages []PackageInfo
	_ = json.Unmarshal(dataBytes, &packages)
	return packages, nil
}

// CreateOrder creates an order (legacy signed endpoint)
func (r *RoamWiFiService) CreateOrder(req OrderRequest) (*RoamWiFiOrderResponse, error) {
	if err := r.ensureAuthenticated(); err != nil {
		return nil, fmt.Errorf("authentication failed: %v", err)
	}

	apiURL := fmt.Sprintf("%s/api_order/createOrder", r.config.APIURL)
	params := map[string]string{
		"token":          r.token,
		"sku_id":         req.SKUID,
		"package_id":     req.PackageID,
		"customer_email": req.CustomerEmail,
		"customer_phone": req.CustomerPhone,
		"quantity":       strconv.Itoa(req.Quantity),
	}
	// remove empty optional params to match signing expectations
	for k, v := range params {
		if v == "" {
			delete(params, k)
		}
	}
	params["sign"] = r.generateSignature(params)

	values := url.Values{}
	for k, v := range params {
		values.Add(k, v)
	}
	fullURL := apiURL + "?" + values.Encode()
	resp, err := http.Post(fullURL, "application/x-www-form-urlencoded", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}
	fmt.Printf("CreateOrder URL=%s RAW=%s\n", fullURL, string(body))

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	var codeStr string
	switch v := result["code"].(type) {
	case float64:
		codeStr = fmt.Sprintf("%.0f", v)
	case string:
		codeStr = v
	}
	if codeStr != "0" && codeStr != "200" {
		if msg, ok := result["message"].(string); ok {
			return nil, fmt.Errorf("API error: %s", msg)
		}
		return nil, fmt.Errorf("API error code=%s body=%s", codeStr, string(body))
	}

	data, _ := result["data"].(map[string]interface{})
	if data == nil {
		return nil, fmt.Errorf("missing data field body=%s", string(body))
	}
	respObj := &RoamWiFiOrderResponse{}
	if v, ok := data["order_id"].(string); ok {
		respObj.OrderID = v
	} else if v, ok := data["orderId"].(string); ok {
		respObj.OrderID = v
	}
	if v, ok := data["status"].(string); ok {
		respObj.Status = v
	}
	if v, ok := data["qr_code"].(string); ok {
		respObj.QRCode = v
	} else if v, ok := data["qrcode"].(string); ok {
		respObj.QRCode = v
	}
	if v, ok := data["activation_code"].(string); ok {
		respObj.ActivationCode = v
	}
	if esim, ok := data["esim_data"].(map[string]interface{}); ok {
		respObj.ESIMData = esim
	}
	return respObj, nil
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

// parsePackageArray converts generic interface array into domain packages
func parsePackageArray(items []interface{}) []PackageInfo {
	var packages []PackageInfo
	for _, item := range items {
		if pkgMap, ok := item.(map[string]interface{}); ok {
			pkg := PackageInfo{}
			if v, ok := pkgMap["package_id"].(string); ok {
				pkg.PackageID = v
			}
			if pkg.PackageID == "" {
				if v, ok := pkgMap["packageId"].(string); ok {
					pkg.PackageID = v
				}
			}
			if v, ok := pkgMap["package_name"].(string); ok {
				pkg.PackageName = v
			}
			if pkg.PackageName == "" {
				if v, ok := pkgMap["packageName"].(string); ok {
					pkg.PackageName = v
				}
			}
			if v, ok := pkgMap["data_limit"].(string); ok {
				pkg.DataLimit = v
			}
			if pkg.DataLimit == "" {
				if v, ok := pkgMap["dataLimit"].(string); ok {
					pkg.DataLimit = v
				}
			}
			if v, ok := pkgMap["validity"]; ok {
				switch vv := v.(type) {
				case float64:
					pkg.Validity = int(vv)
				case string:
					if iv, err := strconv.Atoi(vv); err == nil {
						pkg.Validity = iv
					}
				}
			}
			if v, ok := pkgMap["price"]; ok {
				switch vv := v.(type) {
				case float64:
					pkg.Price = vv
				case string:
					if fv, err := strconv.ParseFloat(vv, 64); err == nil {
						pkg.Price = fv
					}
				}
			}
			if v, ok := pkgMap["countries"].(string); ok {
				pkg.Countries = v
			}
			packages = append(packages, pkg)
		}
	}
	return packages
}

// truncatePremark provides a concise fallback package name extracted from premark HTML/long text
func truncatePremark(s string) string {
	if s == "" {
		return ""
	}
	// strip rudimentary HTML tags
	cleaned := s
	for _, tag := range []string{"<p>", "</p>", "<br>", "<br/>", "<br />"} {
		cleaned = strings.ReplaceAll(cleaned, tag, " ")
	}
	cleaned = strings.TrimSpace(cleaned)
	if len(cleaned) > 60 {
		cleaned = cleaned[:60] + "…"
	}
	return cleaned
}

func keysOf(m map[string]interface{}) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}

// GetPackagesRaw mirrors legacy GetPackages returning raw decoded map
func (r *RoamWiFiService) GetPackagesRaw(skuID string) (map[string]interface{}, error) {
	if err := r.ensureAuthenticated(); err != nil {
		return nil, fmt.Errorf("authentication failed: %v", err)
	}
	apiURL := fmt.Sprintf("%s/api_esim/getPackages", r.config.APIURL)
	params := map[string]string{"token": r.token, "skuId": skuID}
	params["sign"] = r.generateSignature(params)
	values := url.Values{}
	for k, v := range params {
		values.Add(k, v)
	}
	fullURL := apiURL + "?" + values.Encode()
	resp, err := http.Post(fullURL, "application/x-www-form-urlencoded", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}
	fmt.Printf("GetPackagesRaw URL=%s RAW=%s\n", fullURL, string(body))
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}
	return result, nil
}
