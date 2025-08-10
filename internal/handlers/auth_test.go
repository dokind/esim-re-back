package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRegister(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a new router
	router := gin.New()

	// Create a mock user service (you would need to implement this)
	// userService := &MockUserService{}
	// authHandler := NewAuthHandler(userService)

	// Add the route
	// router.POST("/register", authHandler.Register)

	// Test data
	registerData := RegisterRequest{
		Email:     "test@example.com",
		Password:  "password123",
		FirstName: "Test",
		LastName:  "User",
		Phone:     "+97612345678",
	}

	jsonData, _ := json.Marshal(registerData)

	// Create request
	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	w := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLogin(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a new router
	router := gin.New()

	// Create a mock user service (you would need to implement this)
	// userService := &MockUserService{}
	// authHandler := NewAuthHandler(userService)

	// Add the route
	// router.POST("/login", authHandler.Login)

	// Test data
	loginData := LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	jsonData, _ := json.Marshal(loginData)

	// Create request
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	w := httptest.NewRecorder()

	// Serve the request
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
}
