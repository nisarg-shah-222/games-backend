package database

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OTP represents an OTP record in the database
type OTP struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Email     string    `gorm:"type:varchar(255);not null;index" json:"email"`
	Code      string    `gorm:"type:varchar(4);not null" json:"code"`
	ExpiresAt time.Time `gorm:"not null;index" json:"expires_at"`
	Used      bool      `gorm:"default:false" json:"used"`
	CreatedAt time.Time `json:"created_at"`
}

// BeforeCreate hook to generate UUID if not set
func (o *OTP) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return nil
}

// IsExpired checks if the OTP has expired
func (o *OTP) IsExpired() bool {
	return time.Now().After(o.ExpiresAt)
}

// OTPRepository handles OTP database operations
type OTPRepository struct {
	db *gorm.DB
}

// NewOTPRepository creates a new OTP repository
func NewOTPRepository(db *gorm.DB) *OTPRepository {
	return &OTPRepository{db: db}
}

// Create creates a new OTP record
func (r *OTPRepository) Create(otp *OTP) error {
	return r.db.Create(otp).Error
}

// FindValidOTP finds a valid (not used, not expired) OTP for the given email and code
func (r *OTPRepository) FindValidOTP(email, code string) (*OTP, error) {
	var otp OTP
	err := r.db.Where("email = ? AND code = ? AND used = ? AND expires_at > ?",
		email, code, false, time.Now()).
		Order("created_at DESC").
		First(&otp).Error
	if err != nil {
		return nil, err
	}
	return &otp, nil
}

// MarkAsUsed marks an OTP as used
func (r *OTPRepository) MarkAsUsed(id uuid.UUID) error {
	return r.db.Model(&OTP{}).Where("id = ?", id).Update("used", true).Error
}

// CountRecentOTPs counts OTPs created for an email in the last N minutes
func (r *OTPRepository) CountRecentOTPs(email string, minutes int) (int64, error) {
	var count int64
	since := time.Now().Add(-time.Duration(minutes) * time.Minute)
	err := r.db.Model(&OTP{}).
		Where("email = ? AND created_at > ?", email, since).
		Count(&count).Error
	return count, err
}
