package database

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PartnerRequest represents a partner request in the database
type PartnerRequest struct {
	ID             uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	SenderID       uuid.UUID  `gorm:"type:uuid;not null;index" json:"sender_id"`
	RecipientEmail string     `gorm:"type:varchar(255);not null;index" json:"recipient_email"`
	RecipientID    *uuid.UUID `gorm:"type:uuid;index" json:"recipient_id"`
	Status         string     `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"` // pending, accepted, rejected, cancelled
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	// Relations
	Sender    User  `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
	Recipient *User `gorm:"foreignKey:RecipientID" json:"recipient,omitempty"`
}

// BeforeCreate hook to generate UUID if not set
func (pr *PartnerRequest) BeforeCreate(tx *gorm.DB) error {
	if pr.ID == uuid.Nil {
		pr.ID = uuid.New()
	}
	return nil
}

// Partnership represents an active partnership between two users
type Partnership struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	User1ID   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex;index" json:"user1_id"`
	User2ID   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex;index" json:"user2_id"`
	CreatedAt time.Time `json:"created_at"`

	// Relations
	User1 User `gorm:"foreignKey:User1ID" json:"user1,omitempty"`
	User2 User `gorm:"foreignKey:User2ID" json:"user2,omitempty"`
}

// BeforeCreate hook to generate UUID if not set
func (p *Partnership) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// PartnershipRepository handles partnership database operations
type PartnershipRepository struct {
	db *gorm.DB
}

// NewPartnershipRepository creates a new partnership repository
func NewPartnershipRepository(db *gorm.DB) *PartnershipRepository {
	return &PartnershipRepository{db: db}
}

// CreateRequest creates a new partner request
func (r *PartnershipRepository) CreateRequest(request *PartnerRequest) error {
	return r.db.Create(request).Error
}

// FindRequestByID finds a partner request by ID
func (r *PartnershipRepository) FindRequestByID(id uuid.UUID) (*PartnerRequest, error) {
	var request PartnerRequest
	err := r.db.Where("id = ?", id).Preload("Sender").Preload("Recipient").First(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

// FindRequestBySenderAndEmail finds a request by sender and recipient email
func (r *PartnershipRepository) FindRequestBySenderAndEmail(senderID uuid.UUID, recipientEmail string) (*PartnerRequest, error) {
	var request PartnerRequest
	err := r.db.Where("sender_id = ? AND recipient_email = ?", senderID, recipientEmail).First(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

// FindPendingRequestsBySender finds all pending requests sent by a user
func (r *PartnershipRepository) FindPendingRequestsBySender(senderID uuid.UUID) ([]PartnerRequest, error) {
	var requests []PartnerRequest
	err := r.db.Where("sender_id = ? AND status = ?", senderID, "pending").
		Preload("Recipient").
		Order("created_at DESC").
		Find(&requests).Error
	return requests, err
}

// FindPendingRequestsByRecipient finds all pending requests received by a user
// Queries by both recipient_id and recipient_email to handle cases where user didn't exist when request was sent
func (r *PartnershipRepository) FindPendingRequestsByRecipient(recipientID uuid.UUID, recipientEmail string) ([]PartnerRequest, error) {
	var requests []PartnerRequest
	err := r.db.Where("(recipient_id = ? OR recipient_email = ?) AND status = ?", recipientID, recipientEmail, "pending").
		Preload("Sender").
		Order("created_at DESC").
		Find(&requests).Error
	return requests, err
}

// UpdateRequest updates a partner request
func (r *PartnershipRepository) UpdateRequest(request *PartnerRequest) error {
	return r.db.Save(request).Error
}

// CancelPendingRequestsByUser cancels all pending requests for a user (both sent and received)
func (r *PartnershipRepository) CancelPendingRequestsByUser(userID uuid.UUID) error {
	return r.db.Model(&PartnerRequest{}).
		Where("(sender_id = ? OR recipient_id = ?) AND status = ?", userID, userID, "pending").
		Update("status", "cancelled").Error
}

// CreatePartnership creates a new partnership
func (r *PartnershipRepository) CreatePartnership(partnership *Partnership) error {
	return r.db.Create(partnership).Error
}

// FindPartnershipByUser finds a partnership for a given user
func (r *PartnershipRepository) FindPartnershipByUser(userID uuid.UUID) (*Partnership, error) {
	var partnership Partnership
	err := r.db.Where("user1_id = ? OR user2_id = ?", userID, userID).
		Preload("User1").
		Preload("User2").
		First(&partnership).Error
	if err != nil {
		return nil, err
	}
	return &partnership, nil
}

// DeletePartnership deletes a partnership
func (r *PartnershipRepository) DeletePartnership(partnershipID uuid.UUID) error {
	return r.db.Delete(&Partnership{}, partnershipID).Error
}

// DeletePartnershipByUser deletes a partnership by user ID
func (r *PartnershipRepository) DeletePartnershipByUser(userID uuid.UUID) error {
	return r.db.Where("user1_id = ? OR user2_id = ?", userID, userID).Delete(&Partnership{}).Error
}

// UserHasPartnership checks if a user has an active partnership
func (r *PartnershipRepository) UserHasPartnership(userID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&Partnership{}).
		Where("user1_id = ? OR user2_id = ?", userID, userID).
		Count(&count).Error
	return count > 0, err
}
