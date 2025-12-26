package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds application configuration
type Config struct {
	Port        string
	Environment string
	LogLevel    string
	APIBaseURL  string

	// Database
	DatabaseURL string

	// Email Provider (gmail or mailgun)
	EmailProvider string

	// Mailgun Email
	MailgunAPIKey    string
	MailgunDomain    string
	MailgunBaseURL   string
	MailgunFromEmail string

	// Gmail Email
	GmailTokenPath string
	GmailTokenJSON string // Token JSON as environment variable (alternative to file)
	GmailFromEmail string

	OTPExpiryMinutes int

	// JWT
	JWTSecret string
	JWTExpiry string
}

// Load reads configuration from environment variables
func Load() *Config {
	// Load .env file if it exists (ignore error if file doesn't exist)
	_ = godotenv.Load()

	otpExpiryMinutes := 5 // Default 5 minutes
	if expiryStr := getEnv("OTP_EXPIRY_MINUTES", "5"); expiryStr != "" {
		if parsed, err := fmt.Sscanf(expiryStr, "%d", &otpExpiryMinutes); err != nil || parsed != 1 {
			otpExpiryMinutes = 5
		}
	}

	cfg := &Config{
		Port:             getEnv("PORT", "8080"),
		Environment:      getEnv("ENVIRONMENT", "development"),
		LogLevel:         getEnv("LOG_LEVEL", "info"),
		APIBaseURL:       getEnv("API_BASE_URL", "/api/v1"),
		DatabaseURL:      getEnv("DATABASE_URL", ""),
		EmailProvider:    getEnv("EMAIL_PROVIDER", "gmail"), // Default to gmail
		MailgunAPIKey:    getEnv("MAILGUN_API_KEY", ""),
		MailgunDomain:    getEnv("MAILGUN_DOMAIN", ""),
		MailgunBaseURL:   getEnv("MAILGUN_BASE_URL", "https://api.mailgun.net"),
		MailgunFromEmail: getEnv("MAILGUN_FROM_EMAIL", "noreply@gamesapp.com"),
		GmailTokenPath:   getEnv("GMAIL_TOKEN_PATH", "config/token.json"),
		GmailTokenJSON:   getEnv("GMAIL_TOKEN_JSON", ""), // Token JSON as env var (alternative to file)
		GmailFromEmail:   getEnv("GMAIL_FROM_EMAIL", "me"),
		OTPExpiryMinutes: otpExpiryMinutes,
		JWTSecret:        getEnv("JWT_SECRET", ""),
		JWTExpiry:        getEnv("JWT_EXPIRY", "24h"),
	}

	return cfg
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
