package database

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// JSONB is a custom type for PostgreSQL JSONB fields
type JSONB map[string]interface{}

// Value implements the driver.Valuer interface
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// Game represents a game in the database
type Game struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name        string    `gorm:"type:varchar(255);not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	Icon        string    `gorm:"type:varchar(10)" json:"icon"`
	Details     JSONB     `gorm:"type:jsonb;not null;default:'{}'" json:"details"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// BeforeCreate hook to generate UUID if not set
func (g *Game) BeforeCreate(tx *gorm.DB) error {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	return nil
}

// GameRequest represents a game request in the database
type GameRequest struct {
	ID         uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	GameID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"game_id"`
	RequesterID uuid.UUID `gorm:"type:uuid;not null;index" json:"requester_id"`
	PartnerID  uuid.UUID  `gorm:"type:uuid;not null;index" json:"partner_id"`
	Status     string     `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"` // pending, accepted, rejected, expired
	ExpiresAt  time.Time  `gorm:"not null;index" json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`

	// Relations
	Game      Game `gorm:"foreignKey:GameID" json:"game,omitempty"`
	Requester User `gorm:"foreignKey:RequesterID" json:"requester,omitempty"`
	Partner   User `gorm:"foreignKey:PartnerID" json:"partner,omitempty"`
}

// BeforeCreate hook to generate UUID if not set
func (gr *GameRequest) BeforeCreate(tx *gorm.DB) error {
	if gr.ID == uuid.Nil {
		gr.ID = uuid.New()
	}
	return nil
}

// IsExpired checks if the request has expired
func (gr *GameRequest) IsExpired() bool {
	return time.Now().After(gr.ExpiresAt)
}

// Play represents a game play in the database
type Play struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	GameID     uuid.UUID `gorm:"type:uuid;not null;index" json:"game_id"`
	Partner1ID uuid.UUID `gorm:"type:uuid;not null;index" json:"partner1_id"`
	Partner2ID uuid.UUID `gorm:"type:uuid;not null;index" json:"partner2_id"`
	PlayData   JSONB     `gorm:"type:jsonb;not null;default:'{}'" json:"play_data"`
	IsLive     bool      `gorm:"not null;default:true;index" json:"is_live"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	// Relations
	Game     Game `gorm:"foreignKey:GameID" json:"game,omitempty"`
	Partner1 User `gorm:"foreignKey:Partner1ID" json:"partner1,omitempty"`
	Partner2 User `gorm:"foreignKey:Partner2ID" json:"partner2,omitempty"`
}

// BeforeCreate hook to generate UUID if not set
func (p *Play) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// GameRepository handles game database operations
type GameRepository struct {
	db *gorm.DB
}

// NewGameRepository creates a new game repository
func NewGameRepository(db *gorm.DB) *GameRepository {
	return &GameRepository{db: db}
}

// FindAll finds all games
func (r *GameRepository) FindAll() ([]Game, error) {
	var games []Game
	err := r.db.Order("name ASC").Find(&games).Error
	return games, err
}

// FindByID finds a game by ID
func (r *GameRepository) FindByID(id uuid.UUID) (*Game, error) {
	var game Game
	err := r.db.Where("id = ?", id).First(&game).Error
	if err != nil {
		return nil, err
	}
	return &game, nil
}

// GameRequestRepository handles game request database operations
type GameRequestRepository struct {
	db *gorm.DB
}

// NewGameRequestRepository creates a new game request repository
func NewGameRequestRepository(db *gorm.DB) *GameRequestRepository {
	return &GameRequestRepository{db: db}
}

// CreateRequest creates a new game request
func (r *GameRequestRepository) CreateRequest(request *GameRequest) error {
	return r.db.Create(request).Error
}

// FindRequestByID finds a game request by ID
func (r *GameRequestRepository) FindRequestByID(id uuid.UUID) (*GameRequest, error) {
	var request GameRequest
	err := r.db.Where("id = ?", id).
		Preload("Game").
		Preload("Requester").
		Preload("Partner").
		First(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

// FindPendingRequestsByPartner finds all pending requests for a partner
func (r *GameRequestRepository) FindPendingRequestsByPartner(partnerID uuid.UUID) ([]GameRequest, error) {
	var requests []GameRequest
	err := r.db.Where("partner_id = ? AND status = ? AND expires_at > ?", partnerID, "pending", time.Now()).
		Preload("Game").
		Preload("Requester").
		Order("created_at DESC").
		Find(&requests).Error
	return requests, err
}

// FindPendingRequestsByRequester finds all pending requests sent by a requester
func (r *GameRequestRepository) FindPendingRequestsByRequester(requesterID uuid.UUID) ([]GameRequest, error) {
	var requests []GameRequest
	err := r.db.Where("requester_id = ? AND status = ? AND expires_at > ?", requesterID, "pending", time.Now()).
		Preload("Game").
		Preload("Partner").
		Order("created_at DESC").
		Find(&requests).Error
	return requests, err
}

// UpdateRequest updates a game request
func (r *GameRequestRepository) UpdateRequest(request *GameRequest) error {
	return r.db.Save(request).Error
}

// ExpireOldRequests marks expired requests as expired
func (r *GameRequestRepository) ExpireOldRequests() error {
	return r.db.Model(&GameRequest{}).
		Where("status = ? AND expires_at <= ?", "pending", time.Now()).
		Update("status", "expired").Error
}

// PlayRepository handles play database operations
type PlayRepository struct {
	db *gorm.DB
}

// NewPlayRepository creates a new play repository
func NewPlayRepository(db *gorm.DB) *PlayRepository {
	return &PlayRepository{db: db}
}

// CreatePlay creates a new play
func (r *PlayRepository) CreatePlay(play *Play) error {
	return r.db.Create(play).Error
}

// FindPlayByID finds a play by ID
func (r *PlayRepository) FindPlayByID(id uuid.UUID) (*Play, error) {
	var play Play
	err := r.db.Where("id = ?", id).
		Preload("Game").
		Preload("Partner1").
		Preload("Partner2").
		First(&play).Error
	if err != nil {
		return nil, err
	}
	return &play, err
}

// FindLivePlayByPartners finds the live play for a partner combination
func (r *PlayRepository) FindLivePlayByPartners(partner1ID, partner2ID uuid.UUID, gameID uuid.UUID) (*Play, error) {
	var play Play
	// Normalize partner IDs (smaller first)
	smallerID := partner1ID
	largerID := partner2ID
	if partner1ID.String() > partner2ID.String() {
		smallerID = partner2ID
		largerID = partner1ID
	}

	err := r.db.Where("((partner1_id = ? AND partner2_id = ?) OR (partner1_id = ? AND partner2_id = ?)) AND game_id = ? AND is_live = ?",
		smallerID, largerID, largerID, smallerID, gameID, true).
		Preload("Game").
		Preload("Partner1").
		Preload("Partner2").
		First(&play).Error
	if err != nil {
		return nil, err
	}
	return &play, nil
}

// UpdatePlay updates a play
func (r *PlayRepository) UpdatePlay(play *Play) error {
	return r.db.Save(play).Error
}

// EndLivePlay marks a play as not live
func (r *PlayRepository) EndLivePlay(playID uuid.UUID) error {
	return r.db.Model(&Play{}).
		Where("id = ?", playID).
		Update("is_live", false).Error
}

// EndAllLivePlaysByPartners ends all live plays for a partner combination
func (r *PlayRepository) EndAllLivePlaysByPartners(partner1ID, partner2ID uuid.UUID) error {
	// Normalize partner IDs
	smallerID := partner1ID
	largerID := partner2ID
	if partner1ID.String() > partner2ID.String() {
		smallerID = partner2ID
		largerID = partner1ID
	}

	return r.db.Model(&Play{}).
		Where("((partner1_id = ? AND partner2_id = ?) OR (partner1_id = ? AND partner2_id = ?)) AND is_live = ?",
			smallerID, largerID, largerID, smallerID, true).
		Update("is_live", false).Error
}

