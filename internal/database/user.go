package database

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents a user in the database
type User struct {
	ID            uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Email         string    `gorm:"type:varchar(255);unique;not null;index" json:"email"`
	Name          string    `gorm:"type:varchar(255);not null" json:"name"`
	DisplayName   string    `gorm:"type:varchar(100)" json:"display_name"`
	EmailVerified bool      `gorm:"default:false" json:"email_verified"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// BeforeCreate hook to generate UUID if not set
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// UserRepository handles user database operations
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// FindByEmail finds a user by their email
func (r *UserRepository) FindByEmail(email string) (*User, error) {
	var user User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// CreateOrUpdate creates a new user or updates an existing one based on email
func (r *UserRepository) CreateOrUpdate(user *User) (*User, error) {
	var existingUser User
	err := r.db.Where("email = ?", user.Email).First(&existingUser).Error

	if err == gorm.ErrRecordNotFound {
		// Create new user
		if err := r.db.Create(user).Error; err != nil {
			return nil, err
		}
		return user, nil
	} else if err != nil {
		return nil, err
	}

	// Update existing user
	existingUser.Name = user.Name
	existingUser.EmailVerified = user.EmailVerified
	existingUser.UpdatedAt = time.Now()

	if err := r.db.Save(&existingUser).Error; err != nil {
		return nil, err
	}

	return &existingUser, nil
}

// FindByID finds a user by their ID
func (r *UserRepository) FindByID(id uuid.UUID) (*User, error) {
	var user User
	err := r.db.Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Update updates a user's information
func (r *UserRepository) Update(user *User) error {
	return r.db.Save(user).Error
}
