package handler

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/games-app/backend/internal/config"
	"github.com/games-app/backend/internal/database"
	"github.com/games-app/backend/internal/email"
)

// AuthHandler handles authentication requests
type AuthHandler struct {
	config      *config.Config
	userRepo    *database.UserRepository
	otpRepo     *database.OTPRepository
	emailClient email.EmailClient
	jwtSecret   []byte
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(cfg *config.Config) (*AuthHandler, error) {
	// Generate or use JWT secret
	jwtSecret := []byte(cfg.JWTSecret)
	if len(jwtSecret) == 0 {
		// Generate a random secret if not provided (for development only)
		jwtSecret = make([]byte, 32)
		rand.Read(jwtSecret)
	}

	// Initialize email client based on provider
	var emailClient email.EmailClient
	var err error

	switch cfg.EmailProvider {
	case "gmail":
		emailClient, err = email.NewGmailClient(cfg.GmailTokenPath, cfg.GmailFromEmail)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Gmail client: %w", err)
		}
	case "mailgun":
		emailClient = email.NewMailgunClient(cfg.MailgunAPIKey, cfg.MailgunDomain, cfg.MailgunBaseURL, cfg.MailgunFromEmail)
	default:
		// Default to Gmail
		emailClient, err = email.NewGmailClient(cfg.GmailTokenPath, cfg.GmailFromEmail)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Gmail client: %w", err)
		}
	}

	return &AuthHandler{
		config:      cfg,
		userRepo:    database.NewUserRepository(database.DB),
		otpRepo:     database.NewOTPRepository(database.DB),
		emailClient: emailClient,
		jwtSecret:   jwtSecret,
	}, nil
}

// RequestOtpRequest represents the request body for requesting OTP
type RequestOtpRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// RequestOtpResponse represents the response for requesting OTP
type RequestOtpResponse struct {
	Message string `json:"message"`
}

// RequestOtp handles OTP request
func (h *AuthHandler) RequestOtp(c *gin.Context) {
	var req RequestOtpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	email := req.Email

	// Rate limiting: max 3 OTPs per email per 10 minutes
	count, err := h.otpRepo.CountRecentOTPs(email, 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check rate limit"})
		return
	}
	if count >= 3 {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many OTP requests. Please try again later."})
		return
	}

	// Generate 4-digit OTP
	otpCode, err := generateOTP(4)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate OTP"})
		return
	}

	// Create OTP record
	otp := &database.OTP{
		Email:     email,
		Code:      otpCode,
		ExpiresAt: time.Now().Add(time.Duration(h.config.OTPExpiryMinutes) * time.Minute),
		Used:      false,
	}

	if err := h.otpRepo.Create(otp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create OTP: " + err.Error()})
		return
	}

	// Send OTP via email
	if err := h.emailClient.SendOTPEmail(email, otpCode); err != nil {
		// Log error but don't fail the request (OTP is still created)
		fmt.Printf("[AuthHandler] Failed to send email: %v\n", err)
		// In development, return the OTP in the response for testing
		if h.config.Environment == "development" {
			c.JSON(http.StatusOK, RequestOtpResponse{
				Message: fmt.Sprintf("OTP sent (dev mode - code: %s)", otpCode),
			})
			return
		}
	}

	c.JSON(http.StatusOK, RequestOtpResponse{
		Message: "OTP has been sent to your email",
	})
}

// VerifyOtpRequest represents the request body for verifying OTP
type VerifyOtpRequest struct {
	Email string `json:"email" binding:"required,email"`
	OTP   string `json:"otp" binding:"required,len=4"`
}

// VerifyOtpResponse represents the response for verifying OTP
type VerifyOtpResponse struct {
	Token string         `json:"token"`
	User  *database.User `json:"user"`
}

// VerifyOtp handles OTP verification
func (h *AuthHandler) VerifyOtp(c *gin.Context) {
	var req VerifyOtpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Find valid OTP
	otp, err := h.otpRepo.FindValidOTP(req.Email, req.OTP)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired OTP"})
		return
	}

	// Mark OTP as used
	if err := h.otpRepo.MarkAsUsed(otp.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark OTP as used"})
		return
	}

	// Get or create user
	user, err := h.userRepo.FindByEmail(req.Email)
	if err != nil {
		// User doesn't exist, create new one
		newUser := &database.User{
			Email:         req.Email,
			Name:          extractNameFromEmail(req.Email),
			EmailVerified: true,
		}
		user, err = h.userRepo.CreateOrUpdate(newUser)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user: " + err.Error()})
			return
		}
	} else {
		// Update existing user to mark email as verified
		user.EmailVerified = true
		user, err = h.userRepo.CreateOrUpdate(user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user: " + err.Error()})
			return
		}
	}

	// Generate JWT token
	token, err := h.generateJWT(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, VerifyOtpResponse{
		Token: token,
		User:  user,
	})
}

// GetCurrentUserResponse represents the response for getting current user
type GetCurrentUserResponse struct {
	User *database.User `json:"user"`
}

// GetCurrentUser returns the current authenticated user
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	user, err := h.userRepo.FindByID(userUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, GetCurrentUserResponse{
		User: user,
	})
}

// UpdateProfileRequest represents the request body for updating profile
type UpdateProfileRequest struct {
	DisplayName string `json:"display_name" binding:"required,min=1,max=100"`
}

// UpdateProfileResponse represents the response for updating profile
type UpdateProfileResponse struct {
	User *database.User `json:"user"`
}

// UpdateProfile updates the current user's profile
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	user, err := h.userRepo.FindByID(userUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	user.DisplayName = req.DisplayName
	if err := h.userRepo.Update(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, UpdateProfileResponse{
		User: user,
	})
}

// generateJWT generates a JWT token for the user
func (h *AuthHandler) generateJWT(userID uuid.UUID, email string) (string, error) {
	expiry := 24 * time.Hour
	if h.config.JWTExpiry != "" {
		var err error
		expiry, err = time.ParseDuration(h.config.JWTExpiry)
		if err != nil {
			expiry = 24 * time.Hour // Default to 24 hours if parsing fails
		}
	}

	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"email":   email,
		"exp":     time.Now().Add(expiry).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(h.jwtSecret)
}

// VerifyJWT verifies and parses a JWT token
func (h *AuthHandler) VerifyJWT(tokenString string) (uuid.UUID, string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return h.jwtSecret, nil
	})

	if err != nil {
		return uuid.Nil, "", err
	}

	if !token.Valid {
		return uuid.Nil, "", jwt.ErrSignatureInvalid
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return uuid.Nil, "", jwt.ErrSignatureInvalid
	}

	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return uuid.Nil, "", jwt.ErrSignatureInvalid
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, "", err
	}

	email, _ := claims["email"].(string)

	return userID, email, nil
}

// generateOTP generates a random N-digit OTP code
func generateOTP(length int) (string, error) {
	code := ""
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		code += fmt.Sprintf("%d", n.Int64())
	}
	return code, nil
}

// extractNameFromEmail extracts a name from an email address
func extractNameFromEmail(email string) string {
	// Extract the part before @ as a default name
	parts := email
	for idx := 0; idx < len(email); idx++ {
		if email[idx] == '@' {
			parts = email[:idx]
			break
		}
	}
	// Capitalize first letter
	if len(parts) > 0 {
		return string(parts[0]-32) + parts[1:]
	}
	return "User"
}
