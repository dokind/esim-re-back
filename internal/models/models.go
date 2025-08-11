package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// StringArray is a type for PostgreSQL string arrays
type StringArray []string

// Scan implements the sql.Scanner interface
func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}

	switch v := value.(type) {
	case pq.StringArray:
		*s = StringArray(v)
		return nil
	case []string:
		*s = StringArray(v)
		return nil
	case string:
		// Handle PostgreSQL array format like {item1,item2,item3}
		if len(v) == 0 {
			*s = StringArray{}
			return nil
		}
		if v[0] == '{' && v[len(v)-1] == '}' {
			// Remove braces and split by comma
			content := v[1 : len(v)-1]
			if content == "" {
				*s = StringArray{}
				return nil
			}
			parts := strings.Split(content, ",")
			result := make(StringArray, len(parts))
			for i, part := range parts {
				// Remove quotes if present
				part = strings.Trim(part, "\"")
				result[i] = part
			}
			*s = result
			return nil
		}
		// Single item
		*s = StringArray{v}
		return nil
	default:
		return fmt.Errorf("cannot scan %T into StringArray", value)
	}
}

// Value implements the driver.Valuer interface
func (s StringArray) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return pq.Array(s).Value()
}

type User struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Email        string    `json:"email" gorm:"uniqueIndex;not null"`
	PasswordHash string    `json:"-" gorm:"not null"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Phone        string    `json:"phone"`
	IsAdmin      bool      `json:"is_admin" gorm:"default:false"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Product struct {
	ID           uuid.UUID   `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	SKUID        string      `json:"sku_id" gorm:"column:sku_id;not null"`
	Name         string      `json:"name" gorm:"not null"`
	Description  string      `json:"description"`
	DataLimit    string      `json:"data_limit"`
	ValidityDays int         `json:"validity_days"`
	Countries    StringArray `json:"countries" gorm:"type:text[]"`
	Continent    string      `json:"continent"`
	BasePrice    float64     `json:"base_price" gorm:"not null"`
	// CustomPriceUSD optional product-level USD override used for display if set
	CustomPriceUSD     *float64   `json:"custom_price_usd"`
	PriceMNT           *float64   `json:"price_mnt"`            // Price in Mongolian Tugrik
	ExchangeRate       *float64   `json:"exchange_rate"`        // USD to MNT exchange rate used
	ProfitMargin       *float64   `json:"profit_margin"`        // Profit margin percentage
	AdminPriceOverride *float64   `json:"admin_price_override"` // Manual price override by admin
	IsActive           bool       `json:"is_active" gorm:"default:true"`
	LastSyncedAt       *time.Time `json:"last_synced_at"` // When last synced from RoamWiFi
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type Order struct {
	ID        uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID    *uuid.UUID `json:"user_id"`
	User      *User      `json:"user,omitempty"`
	ProductID uuid.UUID  `json:"product_id"`
	Product   Product    `json:"product"`
	// Package pricing (new)
	PackagePriceID      *uuid.UUID           `json:"package_price_id" gorm:"index"`
	PackagePrice        *PackagePrice        `json:"package_price"`
	ProviderPriceID     *int                 `json:"provider_price_id" gorm:"index"`
	OrderNumber         string               `json:"order_number" gorm:"uniqueIndex;not null"`
	QPayInvoiceID       string               `json:"qpay_invoice_id"`
	Status              string               `json:"status" gorm:"default:'pending'"`
	Amount              float64              `json:"amount" gorm:"not null"`
	Currency            string               `json:"currency" gorm:"default:'MNT'"`
	CustomerEmail       string               `json:"customer_email"`
	CustomerPhone       string               `json:"customer_phone"`
	RoamWiFiOrderID     string               `json:"roamwifi_order_id"`
	ESIMData            *string              `json:"esim_data" gorm:"type:jsonb"`
	PaymentTransactions []PaymentTransaction `json:"payment_transactions,omitempty"`
	CreatedAt           time.Time            `json:"created_at"`
	UpdatedAt           time.Time            `json:"updated_at"`
}

type PaymentTransaction struct {
	ID                uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	OrderID           uuid.UUID `json:"order_id"`
	Order             Order     `json:"order,omitempty"`
	QPayTransactionID string    `json:"qpay_transaction_id"`
	Amount            float64   `json:"amount" gorm:"not null"`
	Status            string    `json:"status" gorm:"not null"`
	PaymentMethod     string    `json:"payment_method"`
	TransactionData   string    `json:"transaction_data" gorm:"type:jsonb"`
	CreatedAt         time.Time `json:"created_at"`
}

type AdminSetting struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	SettingKey   string    `json:"setting_key" gorm:"uniqueIndex;not null"`
	SettingValue string    `json:"setting_value"`
	Description  string    `json:"description"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type AuditLog struct {
	ID           uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID       *uuid.UUID `json:"user_id"`
	User         *User      `json:"user,omitempty"`
	Action       string     `json:"action" gorm:"not null"`
	ResourceType string     `json:"resource_type"`
	ResourceID   *uuid.UUID `json:"resource_id"`
	Details      string     `json:"details" gorm:"type:jsonb"`
	IPAddress    string     `json:"ip_address"`
	UserAgent    string     `json:"user_agent"`
	CreatedAt    time.Time  `json:"created_at"`
}

// Package represents an eSIM package offered by RoamWiFi
type Package struct {
	ID                 uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	SKUID              string     `json:"sku_id" gorm:"column:sku_id;not null"`
	PackageID          string     `json:"package_id" gorm:"not null"`
	PackageName        string     `json:"package_name" gorm:"not null"`
	DataLimit          string     `json:"data_limit"`
	ValidityDays       int        `json:"validity_days"`
	Countries          []string   `json:"countries" gorm:"type:text[]"`
	BasePrice          float64    `json:"base_price" gorm:"not null"`
	PriceMNT           *float64   `json:"price_mnt"`
	ExchangeRate       *float64   `json:"exchange_rate"`
	ProfitMargin       *float64   `json:"profit_margin"`
	AdminPriceOverride *float64   `json:"admin_price_override"`
	IsActive           bool       `json:"is_active" gorm:"default:true"`
	LastSyncedAt       *time.Time `json:"last_synced_at"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// PackagePrice stores pricing & override data for provider package (using provider price_id)
type PackagePrice struct {
	ID                uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	SKUID             string     `json:"sku_id" gorm:"column:sku_id;index;not null"`
	ProviderPriceID   int        `json:"provider_price_id" gorm:"uniqueIndex:uniq_provider_price"`
	APICode           string     `json:"api_code" gorm:"index"`
	ShowName          string     `json:"show_name"`
	Flows             float64    `json:"flows"`
	Unit              string     `json:"unit"`
	Days              int        `json:"days"`
	RawProviderPrice  float64    `json:"raw_provider_price"`
	MarkupPercent     *float64   `json:"markup_percent"`
	OverridePriceUSD  *float64   `json:"override_price_usd"`
	EffectivePriceUSD float64    `json:"effective_price_usd"`
	EffectivePriceMNT *float64   `json:"effective_price_mnt"`
	ExchangeRate      *float64   `json:"exchange_rate"`
	PriceSource       string     `json:"price_source"` // base|markup|override
	Active            bool       `json:"active" gorm:"default:true"`
	LastSyncedAt      *time.Time `json:"last_synced_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (pp *PackagePrice) BeforeCreate(tx *gorm.DB) error {
	if pp.ID == uuid.Nil {
		pp.ID = uuid.New()
	}
	return nil
}

// CurrencyRate represents exchange rates for different currencies
type CurrencyRate struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	FromCurrency string    `json:"from_currency" gorm:"not null"` // e.g., "USD"
	ToCurrency   string    `json:"to_currency" gorm:"not null"`   // e.g., "MNT"
	Rate         float64   `json:"rate" gorm:"not null"`
	Source       string    `json:"source"` // e.g., "manual", "api", etc.
	LastUpdated  time.Time `json:"last_updated"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// JSONB is a custom type for PostgreSQL JSONB
type JSONB map[string]interface{}

// Value implements driver.Valuer interface
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner interface
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, j)
	case string:
		return json.Unmarshal([]byte(v), j)
	default:
		return fmt.Errorf("cannot scan %T into JSONB", value)
	}
}

// GormDataType returns the GORM data type
func (JSONB) GormDataType() string {
	return "jsonb"
}

// BeforeCreate hook for User
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// BeforeCreate hook for Product
func (p *Product) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// BeforeCreate hook for Order
func (o *Order) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return nil
}

// BeforeCreate hook for PaymentTransaction
func (pt *PaymentTransaction) BeforeCreate(tx *gorm.DB) error {
	if pt.ID == uuid.Nil {
		pt.ID = uuid.New()
	}
	return nil
}

// BeforeCreate hook for AdminSetting
func (as *AdminSetting) BeforeCreate(tx *gorm.DB) error {
	if as.ID == uuid.Nil {
		as.ID = uuid.New()
	}
	return nil
}

// BeforeCreate hook for AuditLog
func (al *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if al.ID == uuid.Nil {
		al.ID = uuid.New()
	}
	return nil
}

// BeforeCreate hook for Package
func (p *Package) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// BeforeCreate hook for CurrencyRate
func (cr *CurrencyRate) BeforeCreate(tx *gorm.DB) error {
	if cr.ID == uuid.Nil {
		cr.ID = uuid.New()
	}
	return nil
}

// GetDisplayPrice returns the price to display to customers in MNT
func (p *Product) GetDisplayPrice() float64 {
	// If admin has set a manual override, use that
	if p.AdminPriceOverride != nil {
		return *p.AdminPriceOverride
	}

	// If we have a calculated MNT price, use that
	if p.PriceMNT != nil {
		return *p.PriceMNT
	}

	// If we have a custom USD override (legacy), convert not stored -> treat as already MNT? Here we assume override is MNT if PriceMNT absent.
	if p.CustomPriceUSD != nil {
		return *p.CustomPriceUSD
	}

	// Default to base price
	return p.BasePrice
}

// GetDisplayPrice returns the price to display to customers in MNT
func (pkg *Package) GetDisplayPrice() float64 {
	// If admin has set a manual override, use that
	if pkg.AdminPriceOverride != nil {
		return *pkg.AdminPriceOverride
	}

	// If we have a calculated MNT price, use that
	if pkg.PriceMNT != nil {
		return *pkg.PriceMNT
	}

	// Default to base price
	return pkg.BasePrice
}

// CalculateMNTPrice calculates the MNT price based on USD base price and exchange rate
func (p *Product) CalculateMNTPrice(usdToMntRate float64, profitMarginPercent float64) {
	mntPrice := p.BasePrice * usdToMntRate

	// Apply profit margin if specified
	if profitMarginPercent > 0 {
		mntPrice = mntPrice * (1 + profitMarginPercent/100)
	}

	p.PriceMNT = &mntPrice
	p.ExchangeRate = &usdToMntRate
	p.ProfitMargin = &profitMarginPercent
}

// CalculateMNTPrice calculates the MNT price based on USD base price and exchange rate
func (pkg *Package) CalculateMNTPrice(usdToMntRate float64, profitMarginPercent float64) {
	mntPrice := pkg.BasePrice * usdToMntRate

	// Apply profit margin if specified
	if profitMarginPercent > 0 {
		mntPrice = mntPrice * (1 + profitMarginPercent/100)
	}

	pkg.PriceMNT = &mntPrice
	pkg.ExchangeRate = &usdToMntRate
	pkg.ProfitMargin = &profitMarginPercent
}
