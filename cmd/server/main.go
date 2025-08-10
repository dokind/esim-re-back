package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "esim-platform/docs" // Import docs
	"esim-platform/internal/config"
	"esim-platform/internal/database"
	"esim-platform/internal/handlers"
	"esim-platform/internal/middleware"
	"esim-platform/internal/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title eSIM Platform API
// @version 1.0
// @description Complete eSIM platform with QPay payment integration and RoamWiFi API
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		logrus.Warn("No .env file found")
	}

	// Initialize configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.InitDB(cfg.Database)
	if err != nil {
		logrus.Fatal("Failed to connect to database:", err)
	}

	// Initialize Redis
	_, err = database.InitRedis(cfg.Redis)
	if err != nil {
		logrus.Fatal("Failed to connect to Redis:", err)
	}

	// Initialize services
	roamWiFiService := services.NewRoamWiFiService(cfg.RoamWiFi)
	qpayService := services.NewQPayService(cfg.QPay)
	pricingService := services.NewPricingService(db)
	orderService := services.NewOrderService(db, roamWiFiService, qpayService)
	productService := services.NewProductService(db, roamWiFiService)
	userService := services.NewUserService(db)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userService)
	productHandler := handlers.NewProductHandler(productService)
	orderHandler := handlers.NewOrderHandler(orderService)
	adminHandler := handlers.NewAdminHandler(productService, orderService, userService, pricingService)
	webhookHandler := handlers.NewWebhookHandler(orderService, qpayService)

	// Setup Gin router
	router := gin.Default()

	// CORS configuration
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Middleware
	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())

	// Public routes
	public := router.Group("/api/v1")
	{
		// Auth routes
		auth := public.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
		}

		// Product routes
		products := public.Group("/products")
		{
			products.GET("/", productHandler.GetProducts)
			products.GET("/continents", productHandler.GetProductsByContinent)
			products.GET("/skus", productHandler.GetSKUList)
			products.GET("/sku/:skuId", productHandler.GetSKU)
			products.GET("/sku/:skuId/packages", productHandler.GetPackagesBySKU)
			products.GET("/:id", productHandler.GetProduct)
		}

		// Order routes (public for creating orders)
		orders := public.Group("/orders")
		{
			orders.POST("/", orderHandler.CreateOrder)
			orders.GET("/:orderNumber", orderHandler.GetOrder)
			orders.POST("/:orderNumber/pay", orderHandler.InitiatePayment)
		}

		// Webhook routes
		webhooks := public.Group("/webhooks")
		{
			webhooks.POST("/qpay", webhookHandler.HandleQPayWebhook)
		}
	}

	// Protected routes
	protected := router.Group("/api/v1")
	protected.Use(middleware.AuthMiddleware(userService))
	{
		// User routes
		user := protected.Group("/user")
		{
			user.GET("/profile", authHandler.GetProfile)
			user.PUT("/profile", authHandler.UpdateProfile)
			user.GET("/orders", orderHandler.GetUserOrders)
		}
	}

	// Admin routes
	admin := router.Group("/api/v1/admin")
	admin.Use(middleware.AuthMiddleware(userService))
	admin.Use(middleware.AdminMiddleware(userService))
	{
		// Product management
		adminProducts := admin.Group("/products")
		{
			adminProducts.POST("/", adminHandler.CreateProduct)
			adminProducts.PUT("/:id", adminHandler.UpdateProduct)
			adminProducts.DELETE("/:id", adminHandler.DeleteProduct)
			adminProducts.POST("/sync", adminHandler.SyncProductsFromRoamWiFi)
		}

		// Order management
		adminOrders := admin.Group("/orders")
		{
			adminOrders.GET("/", adminHandler.GetAllOrders)
			adminOrders.GET("/:id", adminHandler.GetOrder)
			adminOrders.PUT("/:id/status", adminHandler.UpdateOrderStatus)
		}

		// User management
		adminUsers := admin.Group("/users")
		{
			adminUsers.GET("/", adminHandler.GetAllUsers)
			adminUsers.GET("/:id", adminHandler.GetUser)
			adminUsers.PUT("/:id", adminHandler.UpdateUser)
		}

		// Settings
		adminSettings := admin.Group("/settings")
		{
			adminSettings.GET("/", adminHandler.GetSettings)
			adminSettings.PUT("/", adminHandler.UpdateSettings)
		}

		// Pricing Management
		adminPricing := admin.Group("/pricing")
		{
			adminPricing.GET("/info", adminHandler.GetPricingInfo)
			adminPricing.PUT("/exchange-rate", adminHandler.UpdateExchangeRate)
			adminPricing.POST("/update-all", adminHandler.UpdateAllProductPricing)
		}

		// Product Pricing
		adminProductPricing := admin.Group("/products")
		{
			adminProductPricing.PUT("/:id/price", adminHandler.SetProductPrice)
		}

		// Analytics
		adminAnalytics := admin.Group("/analytics")
		{
			adminAnalytics.GET("/sales", adminHandler.GetSalesAnalytics)
			adminAnalytics.GET("/products", adminHandler.GetProductAnalytics)
		}
	}

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Start server
	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		logrus.Infof("Starting server on port %s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logrus.Info("Shutting down server...")

	// Give outstanding requests a deadline for completion
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logrus.Fatal("Server forced to shutdown:", err)
	}

	logrus.Info("Server exited")
}
