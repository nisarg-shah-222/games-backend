package email

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// GmailClient handles email sending via Gmail API
type GmailClient struct {
	service   *gmail.Service
	fromEmail string
}

// TokenData represents the token.json structure
type TokenData struct {
	Token        string   `json:"token"`
	RefreshToken string   `json:"refresh_token"`
	TokenURI     string   `json:"token_uri"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	Scopes       []string `json:"scopes"`
	Expiry       string   `json:"expiry"`
}

// NewGmailClient creates a new Gmail client using token.json or token JSON from env var
func NewGmailClient(tokenPath string, tokenJSON string, fromEmail string) (*GmailClient, error) {
	// Read token from env var first, then fall back to file
	var tokenData *TokenData
	var err error

	if tokenJSON != "" {
		// Parse token from environment variable
		tokenData, err = parseTokenJSON(tokenJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to parse token from environment variable: %w", err)
		}
	} else {
		// Read token from file
		tokenData, err = loadToken(tokenPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load token: %w", err)
		}
	}

	// Create OAuth2 config
	config := &oauth2.Config{
		ClientID:     tokenData.ClientID,
		ClientSecret: tokenData.ClientSecret,
		Scopes:       tokenData.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.com/o/oauth2/auth",
			TokenURL: tokenData.TokenURI,
		},
	}

	// Parse expiry time
	var expiry time.Time
	if tokenData.Expiry != "" {
		expiry, err = time.Parse(time.RFC3339, tokenData.Expiry)
		if err != nil {
			expiry = time.Now().Add(1 * time.Hour) // Default to 1 hour if parsing fails
		}
	} else {
		expiry = time.Now().Add(1 * time.Hour)
	}

	// Create token
	token := &oauth2.Token{
		AccessToken:  tokenData.Token,
		RefreshToken: tokenData.RefreshToken,
		TokenType:    "Bearer",
		Expiry:       expiry,
	}

	// Create token source with auto-refresh
	ctx := context.Background()
	tokenSource := config.TokenSource(ctx, token)

	// Create Gmail service
	service, err := gmail.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gmail service: %w", err)
	}

	// Get the authenticated user's email address if fromEmail is "me" or empty
	if fromEmail == "" || fromEmail == "me" {
		// Get user profile to get the email address
		profile, err := service.Users.GetProfile("me").Do()
		if err != nil {
			// If we can't get profile, use a default
			fromEmail = "noreply@gmail.com"
		} else {
			fromEmail = profile.EmailAddress
		}
	}

	return &GmailClient{
		service:   service,
		fromEmail: fromEmail,
	}, nil
}

// SendOTPEmail sends an OTP code to the specified email via Gmail API
func (c *GmailClient) SendOTPEmail(toEmail, otpCode string) error {
	// Create email message in RFC 2822 format
	message := fmt.Sprintf("From: %s\r\n", c.fromEmail)
	message += fmt.Sprintf("To: %s\r\n", toEmail)
	message += "Subject: Your Games Verification Code\r\n"
	message += "MIME-Version: 1.0\r\n"
	message += "Content-Type: text/html; charset=UTF-8\r\n"
	message += "\r\n"
	message += fmt.Sprintf(`<h2>Your Verification Code</h2><p>Your verification code is: <strong>%s</strong></p><p>This code will expire in 5 minutes.</p>`, otpCode)

	// Encode message in base64url format (URL-safe, no padding)
	encodedMessage := base64.RawURLEncoding.EncodeToString([]byte(message))

	// Create the message
	msg := &gmail.Message{
		Raw: encodedMessage,
	}

	// Send the message
	ctx := context.Background()
	_, err := c.service.Users.Messages.Send("me", msg).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to send email via Gmail API: %w", err)
	}

	return nil
}

// loadToken loads token data from token.json file
func loadToken(tokenPath string) (*TokenData, error) {
	// If path is relative, make it relative to backend directory
	if !filepath.IsAbs(tokenPath) {
		// Try to find it relative to current working directory or backend folder
		if _, err := os.Stat(tokenPath); os.IsNotExist(err) {
			// Try config/token.json
			tokenPath = filepath.Join("config", "token.json")
		}
	}

	// Read file
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	// Parse JSON
	var tokenData TokenData
	if err := json.Unmarshal(data, &tokenData); err != nil {
		return nil, fmt.Errorf("failed to parse token file: %w", err)
	}

	// Validate required fields
	if tokenData.ClientID == "" {
		return nil, fmt.Errorf("client_id is missing from token file")
	}
	if tokenData.ClientSecret == "" {
		return nil, fmt.Errorf("client_secret is missing from token file")
	}
	if tokenData.Token == "" {
		return nil, fmt.Errorf("token is missing from token file")
	}

	return &tokenData, nil
}

// parseTokenJSON parses token data from JSON string
func parseTokenJSON(jsonStr string) (*TokenData, error) {
	var tokenData TokenData
	if err := json.Unmarshal([]byte(jsonStr), &tokenData); err != nil {
		return nil, fmt.Errorf("failed to parse token JSON: %w", err)
	}

	// Validate required fields
	if tokenData.ClientID == "" {
		return nil, fmt.Errorf("client_id is missing from token JSON")
	}
	if tokenData.ClientSecret == "" {
		return nil, fmt.Errorf("client_secret is missing from token JSON")
	}
	if tokenData.Token == "" {
		return nil, fmt.Errorf("token is missing from token JSON")
	}

	return &tokenData, nil
}
