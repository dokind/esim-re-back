package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"esim-platform/internal/models"

	"gorm.io/gorm"
)

type PricingService struct {
	db *gorm.DB
}

type ExchangeRateAPIResponse struct {
	Result          string             `json:"result"`
	BaseCode        string             `json:"base_code"`
	ConversionRates map[string]float64 `json:"conversion_rates"`
}

func NewPricingService(db *gorm.DB) *PricingService {
	return &PricingService{db: db}
}

// GetUSDToMNTRate gets the current USD to MNT exchange rate
func (p *PricingService) GetUSDToMNTRate() (float64, error) {
	// First try to get from database (cache)
	var rate models.CurrencyRate
	if err := p.db.Where("from_currency = ? AND to_currency = ?", "USD", "MNT").
		Order("last_updated DESC").First(&rate).Error; err == nil {
		// Check if the rate is not older than 24 hours
		if time.Since(rate.LastUpdated) < 24*time.Hour {
			return rate.Rate, nil
		}
	}

	// If no recent rate found, fetch from external API or use default
	newRate, err := p.fetchExchangeRateFromAPI()
	if err != nil {
		// If API fails, use a default rate or the last known rate
		if rate.Rate > 0 {
			return rate.Rate, nil
		}
		// Default fallback rate (approximate USD to MNT)
		return 2850.0, nil
	}

	// Save the new rate to database
	currencyRate := models.CurrencyRate{
		FromCurrency: "USD",
		ToCurrency:   "MNT",
		Rate:         newRate,
		Source:       "api",
		LastUpdated:  time.Now(),
	}
	p.db.Create(&currencyRate)

	return newRate, nil
}

// fetchExchangeRateFromAPI fetches exchange rate from external API
func (p *PricingService) fetchExchangeRateFromAPI() (float64, error) {
	// Using a free exchange rate API (you can replace with your preferred provider)
	url := "https://api.exchangerate-api.com/v4/latest/USD"

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var apiResp ExchangeRateAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return 0, err
	}

	if mntRate, exists := apiResp.ConversionRates["MNT"]; exists {
		return mntRate, nil
	}

	return 0, fmt.Errorf("MNT rate not found in API response")
}

// GetDefaultProfitMargin gets the default profit margin from settings
func (p *PricingService) GetDefaultProfitMargin() float64 {
	var setting models.AdminSetting
	if err := p.db.Where("setting_key = ?", "default_profit_margin").First(&setting).Error; err == nil {
		if margin, err := strconv.ParseFloat(setting.SettingValue, 64); err == nil {
			return margin
		}
	}
	// Default profit margin if not set
	return 10.0
}

// UpdateProductPricing updates the MNT pricing for a product
func (p *PricingService) UpdateProductPricing(productID string) error {
	var product models.Product
	if err := p.db.First(&product, "id = ?", productID).Error; err != nil {
		return err
	}

	usdToMntRate, err := p.GetUSDToMNTRate()
	if err != nil {
		return err
	}

	profitMargin := p.GetDefaultProfitMargin()
	product.CalculateMNTPrice(usdToMntRate, profitMargin)

	return p.db.Save(&product).Error
}

// UpdatePackagePricing updates the MNT pricing for a package
func (p *PricingService) UpdatePackagePricing(packageID string) error {
	var pkg models.Package
	if err := p.db.First(&pkg, "id = ?", packageID).Error; err != nil {
		return err
	}

	usdToMntRate, err := p.GetUSDToMNTRate()
	if err != nil {
		return err
	}

	profitMargin := p.GetDefaultProfitMargin()
	pkg.CalculateMNTPrice(usdToMntRate, profitMargin)

	return p.db.Save(&pkg).Error
}

// UpdateAllProductPricing updates pricing for all active products
func (p *PricingService) UpdateAllProductPricing() error {
	var products []models.Product
	if err := p.db.Where("is_active = ?", true).Find(&products).Error; err != nil {
		return err
	}

	usdToMntRate, err := p.GetUSDToMNTRate()
	if err != nil {
		return err
	}

	profitMargin := p.GetDefaultProfitMargin()

	for i := range products {
		products[i].CalculateMNTPrice(usdToMntRate, profitMargin)
	}

	return p.db.Save(&products).Error
}

// UpdateAllPackagePricing updates pricing for all active packages
func (p *PricingService) UpdateAllPackagePricing() error {
	var packages []models.Package
	if err := p.db.Where("is_active = ?", true).Find(&packages).Error; err != nil {
		return err
	}

	usdToMntRate, err := p.GetUSDToMNTRate()
	if err != nil {
		return err
	}

	profitMargin := p.GetDefaultProfitMargin()

	for i := range packages {
		packages[i].CalculateMNTPrice(usdToMntRate, profitMargin)
	}

	return p.db.Save(&packages).Error
}

// SetManualExchangeRate allows admin to set a manual exchange rate
func (p *PricingService) SetManualExchangeRate(rate float64) error {
	currencyRate := models.CurrencyRate{
		FromCurrency: "USD",
		ToCurrency:   "MNT",
		Rate:         rate,
		Source:       "manual",
		LastUpdated:  time.Now(),
	}
	return p.db.Create(&currencyRate).Error
}
