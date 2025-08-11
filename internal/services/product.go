package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"esim-platform/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProductService struct {
	db              *gorm.DB
	roamWiFiService *RoamWiFiService
}

// EnrichedRoamWiFiPackage extends provider package data with pricing fields
type EnrichedRoamWiFiPackage struct {
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
	// Pricing enrichment
	EffectivePriceUSD float64  `json:"effective_price_usd"`
	EffectivePriceMNT *float64 `json:"effective_price_mnt,omitempty"`
	PriceSource       string   `json:"price_source"`
	MarkupPercent     *float64 `json:"markup_percent,omitempty"`
	OverridePriceUSD  *float64 `json:"override_price_usd,omitempty"`
}

// EnrichedRoamWiFiPackagesResponse top-level enriched response
type EnrichedRoamWiFiPackagesResponse struct {
	SKUId          int                       `json:"sku_id"`
	Display        string                    `json:"display"`
	DisplayEn      string                    `json:"display_en"`
	CountryCode    string                    `json:"country_code"`
	SupportCountry []string                  `json:"support_country"`
	ImageURL       string                    `json:"image_url"`
	CountryImages  []RoamWiFiCountryImage    `json:"country_images"`
	Packages       []EnrichedRoamWiFiPackage `json:"packages"`
}

type CreateProductRequest struct {
	SKUID          string   `json:"sku_id" binding:"required"`
	Name           string   `json:"name" binding:"required"`
	Description    string   `json:"description"`
	DataLimit      string   `json:"data_limit"`
	ValidityDays   int      `json:"validity_days"`
	Countries      []string `json:"countries"`
	Continent      string   `json:"continent"`
	BasePrice      float64  `json:"base_price" binding:"required"`
	CustomPriceUSD *float64 `json:"custom_price_usd"`
}

// SyncPackagePrices fetches provider packages for a SKU and upserts pricing rows
func (p *ProductService) SyncPackagePrices(skuID string) error {
	detailed, err := p.roamWiFiService.GetPackagesDetailed(skuID)
	if err != nil {
		return fmt.Errorf("fetch detailed packages: %w", err)
	}
	if detailed == nil {
		return fmt.Errorf("no data returned for sku %s", skuID)
	}
	pricing := NewPricingService(p.db)
	rate, _ := pricing.GetUSDToMNTRate()
	now := time.Now()
	for _, pkg := range detailed.Packages {
		effective := pkg.Price
		priceSource := "base"
		var existing models.PackagePrice
		tx := p.db.Where("provider_price_id = ?", pkg.PriceID).First(&existing)
		if tx.Error != nil && !errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return fmt.Errorf("read existing price: %w", tx.Error)
		}
		if existing.ID == uuid.Nil {
			var effectiveMNT *float64
			if rate > 0 {
				mnt := effective * rate
				effectiveMNT = &mnt
			}
			rec := models.PackagePrice{SKUID: skuID, ProviderPriceID: pkg.PriceID, APICode: pkg.APICode, ShowName: pkg.ShowName, Flows: pkg.Flows, Unit: pkg.Unit, Days: pkg.Days, RawProviderPrice: pkg.Price, EffectivePriceUSD: effective, EffectivePriceMNT: effectiveMNT, ExchangeRate: &rate, PriceSource: priceSource, Active: true, LastSyncedAt: &now}
			if err := p.db.Create(&rec).Error; err != nil {
				return fmt.Errorf("create package price: %w", err)
			}
		} else {
			existing.SKUID = skuID
			existing.APICode = pkg.APICode
			existing.ShowName = pkg.ShowName
			existing.Flows = pkg.Flows
			existing.Unit = pkg.Unit
			existing.Days = pkg.Days
			existing.RawProviderPrice = pkg.Price
			if existing.OverridePriceUSD != nil {
				existing.EffectivePriceUSD = *existing.OverridePriceUSD
				priceSource = "override"
			} else if existing.MarkupPercent != nil {
				existing.EffectivePriceUSD = pkg.Price * (1 + *existing.MarkupPercent/100)
				priceSource = "markup"
			} else {
				existing.EffectivePriceUSD = pkg.Price
				priceSource = "base"
			}
			existing.PriceSource = priceSource
			existing.ExchangeRate = &rate
			if rate > 0 {
				mnt := existing.EffectivePriceUSD * rate
				existing.EffectivePriceMNT = &mnt
			}
			existing.LastSyncedAt = &now
			existing.Active = true
			if err := p.db.Save(&existing).Error; err != nil {
				return fmt.Errorf("update package price: %w", err)
			}
		}
	}
	var providerIDs []int
	for _, pkg := range detailed.Packages {
		providerIDs = append(providerIDs, pkg.PriceID)
	}
	if err := p.db.Model(&models.PackagePrice{}).Where("sku_id = ? AND provider_price_id NOT IN ?", skuID, providerIDs).Updates(map[string]interface{}{"active": false}).Error; err != nil {
		return fmt.Errorf("deactivate missing packages: %w", err)
	}
	return nil
}

// SetPackageMarkup sets markup percent and recomputes effective price (clears override)
func (p *ProductService) SetPackageMarkup(providerPriceID int, markup float64) error {
	var pp models.PackagePrice
	if err := p.db.Where("provider_price_id = ?", providerPriceID).First(&pp).Error; err != nil {
		return err
	}
	pp.MarkupPercent = &markup
	pp.OverridePriceUSD = nil
	// recompute
	base := pp.RawProviderPrice
	pp.EffectivePriceUSD = base * (1 + markup/100)
	pp.PriceSource = "markup"
	rateSvc := NewPricingService(p.db)
	if rate, err := rateSvc.GetUSDToMNTRate(); err == nil {
		pp.ExchangeRate = &rate
		mnt := pp.EffectivePriceUSD * rate
		pp.EffectivePriceMNT = &mnt
	} else {
		pp.ExchangeRate = nil
	}
	return p.db.Save(&pp).Error
}

// SetPackageOverride sets or clears override price (if nil passed clears override and falls back to markup/base)
func (p *ProductService) SetPackageOverride(providerPriceID int, override *float64) error {
	var pp models.PackagePrice
	if err := p.db.Where("provider_price_id = ?", providerPriceID).First(&pp).Error; err != nil {
		return err
	}
	if override == nil {
		pp.OverridePriceUSD = nil
		// fallback recompute
		if pp.MarkupPercent != nil {
			pp.EffectivePriceUSD = pp.RawProviderPrice * (1 + *pp.MarkupPercent/100)
			pp.PriceSource = "markup"
		} else {
			pp.EffectivePriceUSD = pp.RawProviderPrice
			pp.PriceSource = "base"
		}
	} else {
		if *override <= 0 {
			return fmt.Errorf("override must be > 0")
		}
		pp.OverridePriceUSD = override
		pp.EffectivePriceUSD = *override
		pp.PriceSource = "override"
	}
	rateSvc := NewPricingService(p.db)
	if rate, err := rateSvc.GetUSDToMNTRate(); err == nil {
		pp.ExchangeRate = &rate
		mnt := pp.EffectivePriceUSD * rate
		pp.EffectivePriceMNT = &mnt
	} else {
		pp.ExchangeRate = nil
	}
	return p.db.Save(&pp).Error
}

type UpdateProductRequest struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	DataLimit      string   `json:"data_limit"`
	ValidityDays   int      `json:"validity_days"`
	Countries      []string `json:"countries"`
	Continent      string   `json:"continent"`
	BasePrice      float64  `json:"base_price"`
	CustomPriceUSD *float64 `json:"custom_price_usd"`
	IsActive       *bool    `json:"is_active"`
}

func NewProductService(db *gorm.DB, roamWiFiService *RoamWiFiService) *ProductService {
	return &ProductService{
		db:              db,
		roamWiFiService: roamWiFiService,
	}
}

// GetProducts retrieves products with filtering and pagination
func (p *ProductService) GetProducts(page, limit int, continent, active string) ([]models.Product, int64, error) {
	var products []models.Product
	var total int64

	offset := (page - 1) * limit
	query := p.db.Model(&models.Product{})

	// Apply filters
	if continent != "" {
		query = query.Where("continent = ?", continent)
	}
	if active != "" {
		if active == "true" {
			query = query.Where("is_active = ?", true)
		} else if active == "false" {
			query = query.Where("is_active = ?", false)
		}
	}

	// Get total count
	query.Count(&total)

	// Get products with pagination
	if err := query.Offset(offset).Limit(limit).Find(&products).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get products: %v", err)
	}

	return products, total, nil
}

// GetProductsByContinent retrieves products grouped by continent
func (p *ProductService) GetProductsByContinent() (map[string][]models.Product, error) {
	var products []models.Product
	if err := p.db.Where("is_active = ?", true).Find(&products).Error; err != nil {
		return nil, fmt.Errorf("failed to get products: %v", err)
	}

	// Group products by continent
	continentMap := make(map[string][]models.Product)
	for _, product := range products {
		continentMap[product.Continent] = append(continentMap[product.Continent], product)
	}

	return continentMap, nil
}

// GetProduct retrieves a specific product by ID
func (p *ProductService) GetProduct(productID uuid.UUID) (*models.Product, error) {
	var product models.Product
	if err := p.db.Where("id = ?", productID).First(&product).Error; err != nil {
		return nil, fmt.Errorf("product not found: %v", err)
	}
	return &product, nil
}

// GetPackagesBySKU retrieves packages for a specific SKU from RoamWiFi
func (p *ProductService) GetPackagesBySKU(skuID string) ([]PackageInfo, error) {
	return p.roamWiFiService.GetPackagesBySKU(skuID)
}

// CreateProduct creates a new product
func (p *ProductService) CreateProduct(req CreateProductRequest) (*models.Product, error) {
	product := models.Product{
		SKUID:          req.SKUID,
		Name:           req.Name,
		Description:    req.Description,
		DataLimit:      req.DataLimit,
		ValidityDays:   req.ValidityDays,
		Countries:      req.Countries,
		Continent:      req.Continent,
		BasePrice:      req.BasePrice,
		CustomPriceUSD: req.CustomPriceUSD,
		IsActive:       true,
	}

	if err := p.db.Create(&product).Error; err != nil {
		return nil, fmt.Errorf("failed to create product: %v", err)
	}

	return &product, nil
}

// UpdateProduct updates an existing product
func (p *ProductService) UpdateProduct(productID uuid.UUID, req UpdateProductRequest) (*models.Product, error) {
	var product models.Product
	if err := p.db.Where("id = ?", productID).First(&product).Error; err != nil {
		return nil, fmt.Errorf("product not found: %v", err)
	}

	// Update fields
	if req.Name != "" {
		product.Name = req.Name
	}
	if req.Description != "" {
		product.Description = req.Description
	}
	if req.DataLimit != "" {
		product.DataLimit = req.DataLimit
	}
	if req.ValidityDays > 0 {
		product.ValidityDays = req.ValidityDays
	}
	if req.Countries != nil {
		product.Countries = req.Countries
	}
	if req.Continent != "" {
		product.Continent = req.Continent
	}
	if req.BasePrice > 0 {
		product.BasePrice = req.BasePrice
	}
	if req.CustomPriceUSD != nil {
		product.CustomPriceUSD = req.CustomPriceUSD
	}
	if req.IsActive != nil {
		product.IsActive = *req.IsActive
	}

	if err := p.db.Save(&product).Error; err != nil {
		return nil, fmt.Errorf("failed to update product: %v", err)
	}

	return &product, nil
}

// DeleteProduct deletes a product
func (p *ProductService) DeleteProduct(productID uuid.UUID) error {
	return p.db.Where("id = ?", productID).Delete(&models.Product{}).Error
}

// SyncProductsFromRoamWiFi syncs products from RoamWiFi API
func (p *ProductService) SyncProductsFromRoamWiFi() (int, error) {
	// Get SKU list from RoamWiFi
	skuList, err := p.roamWiFiService.GetSKUList()
	if err != nil {
		return 0, fmt.Errorf("failed to get SKU list from RoamWiFi: %v", err)
	}

	count := 0
	for _, sku := range skuList {
		// Convert SKUID int to string
		skuIDStr := fmt.Sprintf("%d", sku.SKUID)

		// Check if product already exists
		var existingProduct models.Product
		if err := p.db.Where("sku_id = ?", skuIDStr).First(&existingProduct).Error; err == nil {
			// Product exists, update it
			existingProduct.Name = sku.Display
			existingProduct.Continent = p.inferContinentFromDisplay(sku.Display)
			// Set default values since API doesn't provide these
			existingProduct.DataLimit = "Varies"
			existingProduct.ValidityDays = 30 // Default validity
			existingProduct.BasePrice = 25.0  // Default price, admin can update later

			// Parse country code - this might be a region code, we'll store it
			existingProduct.Countries = []string{sku.CountryCode}

			if err := p.db.Save(&existingProduct).Error; err != nil {
				continue // Skip this product if update fails
			}
		} else {
			// Product doesn't exist, create it
			product := models.Product{
				SKUID:        skuIDStr,
				Name:         sku.Display,
				Continent:    p.inferContinentFromDisplay(sku.Display),
				DataLimit:    "Varies",
				ValidityDays: 30,   // Default validity
				BasePrice:    25.0, // Default price, admin can update later
				Countries:    []string{sku.CountryCode},
				IsActive:     true,
			}

			if err := p.db.Create(&product).Error; err != nil {
				continue // Skip this product if creation fails
			}
		}
		count++
	}

	return count, nil
}

// inferContinentFromDisplay tries to infer continent from the display name
func (p *ProductService) inferContinentFromDisplay(display string) string {
	displayLower := strings.ToLower(display)

	if strings.Contains(displayLower, "africa") {
		return "Africa"
	}
	if strings.Contains(displayLower, "asia") {
		return "Asia"
	}
	if strings.Contains(displayLower, "europe") {
		return "Europe"
	}
	if strings.Contains(displayLower, "america") || strings.Contains(displayLower, "usa") {
		return "North America"
	}
	if strings.Contains(displayLower, "oceania") || strings.Contains(displayLower, "australia") {
		return "Oceania"
	}

	// Default to Global if we can't determine
	return "Global"
}

// SearchProducts searches products by name or description
func (p *ProductService) SearchProducts(query string, page, limit int) ([]models.Product, int64, error) {
	var products []models.Product
	var total int64

	offset := (page - 1) * limit

	// Build search query
	searchQuery := p.db.Where("name ILIKE ? OR description ILIKE ?",
		"%"+query+"%", "%"+query+"%")

	// Get total count
	searchQuery.Model(&models.Product{}).Count(&total)

	// Get products with pagination
	if err := searchQuery.Offset(offset).Limit(limit).Find(&products).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to search products: %v", err)
	}

	return products, total, nil
}

// GetProductsByPriceRange retrieves products within a price range
func (p *ProductService) GetProductsByPriceRange(minPrice, maxPrice float64, page, limit int) ([]models.Product, int64, error) {
	var products []models.Product
	var total int64

	offset := (page - 1) * limit

	// Build query
	query := p.db.Where("base_price BETWEEN ? AND ?", minPrice, maxPrice)

	// Get total count
	query.Model(&models.Product{}).Count(&total)

	// Get products with pagination
	if err := query.Offset(offset).Limit(limit).Find(&products).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get products by price range: %v", err)
	}

	return products, total, nil
}

// GetSKUList proxies to RoamWiFiService to fetch live SKU list
func (p *ProductService) GetSKUList() ([]SKUInfo, error) {
	return p.roamWiFiService.GetSKUList()
}

// GetSKUByID proxies to RoamWiFiService to fetch a single SKU
func (p *ProductService) GetSKUByID(skuID string) (*SKUInfo, error) {
	return p.roamWiFiService.GetSKUByID(skuID)
}

// GetPackagesRaw proxies to RoamWiFiService to fetch raw packages data
func (p *ProductService) GetPackagesRaw(skuID string) (map[string]interface{}, error) {
	return p.roamWiFiService.GetPackagesRaw(skuID)
}

// GetPackagesDetailed proxies to RoamWiFiService detailed response
func (p *ProductService) GetPackagesDetailed(skuID string) (*EnrichedRoamWiFiPackagesResponse, error) {
	base, err := p.roamWiFiService.GetPackagesDetailed(skuID)
	if err != nil {
		return nil, err
	}
	enriched := &EnrichedRoamWiFiPackagesResponse{
		SKUId:          base.SKUId,
		Display:        base.Display,
		DisplayEn:      base.DisplayEn,
		CountryCode:    base.CountryCode,
		SupportCountry: base.SupportCountry,
		ImageURL:       base.ImageURL,
		CountryImages:  base.CountryImages,
		Packages:       make([]EnrichedRoamWiFiPackage, 0, len(base.Packages)),
	}
	// Load pricing map
	var prices []models.PackagePrice
	priceMap := map[int]models.PackagePrice{}
	if err := p.db.Where("sku_id = ? AND active = ?", skuID, true).Find(&prices).Error; err == nil {
		for _, pr := range prices {
			priceMap[pr.ProviderPriceID] = pr
		}
	}
	// Merge
	for _, pkg := range base.Packages {
		merged := EnrichedRoamWiFiPackage{
			APICode: pkg.APICode, Flows: pkg.Flows, Unit: pkg.Unit, Days: pkg.Days, Price: pkg.Price, PriceID: pkg.PriceID, FlowType: pkg.FlowType, ShowName: pkg.ShowName, PID: pkg.PID, Premark: pkg.Premark, Overlay: pkg.Overlay, ExpireDays: pkg.ExpireDays, Network: pkg.Network, SupportDaypass: pkg.SupportDaypass, OpenCardFee: pkg.OpenCardFee, MinDay: pkg.MinDay, SingleDiscountDay: pkg.SingleDiscountDay, SingleDiscount: pkg.SingleDiscount, MaxDiscount: pkg.MaxDiscount, MaxDay: pkg.MaxDay, MustDate: pkg.MustDate, HadDaypassDetail: pkg.HadDaypassDetail,
			EffectivePriceUSD: pkg.Price, PriceSource: "base",
		}
		if pr, ok := priceMap[pkg.PriceID]; ok {
			merged.EffectivePriceUSD = pr.EffectivePriceUSD
			merged.EffectivePriceMNT = pr.EffectivePriceMNT
			merged.PriceSource = pr.PriceSource
			merged.MarkupPercent = pr.MarkupPercent
			merged.OverridePriceUSD = pr.OverridePriceUSD
		}
		enriched.Packages = append(enriched.Packages, merged)
	}
	return enriched, nil
}
