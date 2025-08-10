package services

import (
	"fmt"
	"strings"

	"esim-platform/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProductService struct {
	db              *gorm.DB
	roamWiFiService *RoamWiFiService
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
		SKUID:        req.SKUID,
		Name:         req.Name,
		Description:  req.Description,
		DataLimit:    req.DataLimit,
		ValidityDays: req.ValidityDays,
		Countries:    req.Countries,
		Continent:    req.Continent,
		BasePrice:    req.BasePrice,
		CustomPrice:  req.CustomPrice,
		IsActive:     true,
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
	if req.CustomPrice != nil {
		product.CustomPrice = req.CustomPrice
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
