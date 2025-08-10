package config

import (
	"os"
	"strconv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	QPay     QPayConfig
	RoamWiFi RoamWiFiConfig
	JWT      JWTConfig
}

type ServerConfig struct {
	Port string
	Host string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type QPayConfig struct {
	MerchantID       string
	MerchantPassword string
	Endpoint         string
	BaseURL          string
	Username         string
	Password         string
	InvoiceCode      string
	CallbackURL      string
}

type RoamWiFiConfig struct {
	APIKey      string
	APIURL      string
	PhoneNumber string
	Password    string
}

type JWTConfig struct {
	Secret     string
	Expiration int // in hours
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
			Host: getEnv("HOST", "0.0.0.0"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "esim_user"),
			Password: getEnv("DB_PASSWORD", "esim_password"),
			Name:     getEnv("DB_NAME", "esim_db"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		QPay: QPayConfig{
			MerchantID:       getEnv("QPAY_MERCHANT_ID", ""),
			MerchantPassword: getEnv("QPAY_MERCHANT_PASSWORD", ""),
			Endpoint:         getEnv("QPAY_ENDPOINT", "https://merchant.qpay.mn/v2"),
			BaseURL:          getEnv("QPAY_BASE_URL", "https://merchant.qpay.mn"),
			Username:         getEnv("QPAY_USERNAME", "DOKIND_MN"),
			Password:         getEnv("QPAY_PASSWORD", "xQF7fgDM"),
			InvoiceCode:      getEnv("QPAY_INVOICE_CODE", "DOKIND_MN_INVOICE"),
			CallbackURL:      getEnv("QPAY_CALLBACK_URL", ""),
		},
		RoamWiFi: RoamWiFiConfig{
			APIKey:      getEnv("ROAMWIFI_API_KEY", ""),
			APIURL:      getEnv("ROAMWIFI_API_URL", "http://bpm.roamwifi.com"),
			PhoneNumber: getEnv("ROAMWIFI_PHONENUMBER", ""),
			Password:    getEnv("ROAMWIFI_PASSWORD", ""),
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", "your-secret-key"),
			Expiration: getEnvAsInt("JWT_EXPIRATION", 24),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
