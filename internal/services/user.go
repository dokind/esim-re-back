package services

import (
	"fmt"

	"esim-platform/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserService struct {
	DB *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{
		DB: db,
	}
}

// GetUserByID retrieves a user by ID
func (u *UserService) GetUserByID(userID uuid.UUID) (*models.User, error) {
	var user models.User
	if err := u.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, fmt.Errorf("user not found: %v", err)
	}
	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (u *UserService) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	if err := u.DB.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, fmt.Errorf("user not found: %v", err)
	}
	return &user, nil
}

// CreateUser creates a new user
func (u *UserService) CreateUser(user *models.User) error {
	return u.DB.Create(user).Error
}

// UpdateUser updates an existing user
func (u *UserService) UpdateUser(user *models.User) error {
	return u.DB.Save(user).Error
}

// DeleteUser deletes a user
func (u *UserService) DeleteUser(userID uuid.UUID) error {
	return u.DB.Where("id = ?", userID).Delete(&models.User{}).Error
}

// GetAllUsers retrieves all users with pagination
func (u *UserService) GetAllUsers(page, limit int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	offset := (page - 1) * limit

	// Get total count
	u.DB.Model(&models.User{}).Count(&total)

	// Get users with pagination
	if err := u.DB.Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get users: %v", err)
	}

	return users, total, nil
}

// GetUsersByRole retrieves users by role (admin/non-admin)
func (u *UserService) GetUsersByRole(isAdmin bool, page, limit int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	offset := (page - 1) * limit

	// Get total count
	u.DB.Model(&models.User{}).Where("is_admin = ?", isAdmin).Count(&total)

	// Get users with pagination
	if err := u.DB.Where("is_admin = ?", isAdmin).Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get users: %v", err)
	}

	return users, total, nil
}

// UpdateUserRole updates user's admin status
func (u *UserService) UpdateUserRole(userID uuid.UUID, isAdmin bool) error {
	return u.DB.Model(&models.User{}).Where("id = ?", userID).Update("is_admin", isAdmin).Error
}

// SearchUsers searches users by email or name
func (u *UserService) SearchUsers(query string, page, limit int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	offset := (page - 1) * limit

	// Build search query
	searchQuery := u.DB.Where("email ILIKE ? OR first_name ILIKE ? OR last_name ILIKE ?", 
		"%"+query+"%", "%"+query+"%", "%"+query+"%")

	// Get total count
	searchQuery.Model(&models.User{}).Count(&total)

	// Get users with pagination
	if err := searchQuery.Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to search users: %v", err)
	}

	return users, total, nil
} 