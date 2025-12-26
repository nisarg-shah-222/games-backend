package email

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// MailgunClient handles email sending via Mailgun API
type MailgunClient struct {
	APIKey    string
	Domain    string
	BaseURL   string
	FromEmail string
}

// NewMailgunClient creates a new Mailgun client
func NewMailgunClient(apiKey, domain, baseURL, fromEmail string) *MailgunClient {
	return &MailgunClient{
		APIKey:    apiKey,
		Domain:    domain,
		BaseURL:   baseURL,
		FromEmail: fromEmail,
	}
}

// SendOTPEmail sends an OTP code to the specified email
func (c *MailgunClient) SendOTPEmail(toEmail, otpCode string) error {
	if c.APIKey == "" {
		// In development, just log the OTP instead of sending
		fmt.Printf("[Mailgun] OTP for %s: %s\n", toEmail, otpCode)
		return nil
	}

	// Validate configuration
	if c.Domain == "" {
		return fmt.Errorf("mailgun domain is not configured")
	}

	// Ensure from email matches the Mailgun domain
	// Extract email address from "Display Name <email@domain>" format if present
	fromEmail := c.FromEmail
	if fromEmail == "" {
		// Default to postmaster@domain if not configured
		fromEmail = fmt.Sprintf("Games <postmaster@%s>", c.Domain)
	} else {
		// Check if from email domain matches Mailgun domain
		emailDomain := extractDomainFromEmail(fromEmail)
		if emailDomain != c.Domain {
			// Extract display name if present, otherwise use default
			displayName := extractDisplayName(fromEmail)
			if displayName == "" {
				displayName = "Games"
			}
			fromEmail = fmt.Sprintf("%s <postmaster@%s>", displayName, c.Domain)
			fmt.Printf("[Mailgun] From email domain doesn't match Mailgun domain. Using '%s' instead\n", fromEmail)
		}
	}

	// Mailgun API endpoint: https://api.mailgun.net/v3/{domain}/messages
	apiURL := fmt.Sprintf("%s/v3/%s/messages", c.BaseURL, c.Domain)

	// Prepare form data
	data := url.Values{}
	data.Set("from", fromEmail)
	data.Set("to", toEmail)
	data.Set("subject", "Your Games Verification Code")
	data.Set("text", fmt.Sprintf("Your verification code is: %s\n\nThis code will expire in 5 minutes.", otpCode))
	data.Set("html", fmt.Sprintf("<h2>Your Verification Code</h2><p>Your verification code is: <strong>%s</strong></p><p>This code will expire in 5 minutes.</p>", otpCode))

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Mailgun uses Basic Auth with api:key format
	req.SetBasicAuth("api", c.APIKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Read response body for detailed error message
		bodyBytes, readErr := io.ReadAll(resp.Body)
		bodyStr := ""
		if readErr == nil {
			bodyStr = string(bodyBytes)
		}

		// Provide helpful error messages for common issues
		if resp.StatusCode == 403 && strings.Contains(bodyStr, "authorized recipients") {
			return fmt.Errorf("mailgun sandbox restriction: recipient '%s' must be added to authorized recipients in Mailgun dashboard. Error: %s", toEmail, bodyStr)
		}

		// Include response body in error if available
		if bodyStr != "" {
			return fmt.Errorf("mailgun API returned status %d: %s", resp.StatusCode, bodyStr)
		}
		return fmt.Errorf("mailgun API returned status %d", resp.StatusCode)
	}

	return nil
}

// extractDomainFromEmail extracts the domain from an email address
// Handles both "email@domain.com" and "Display Name <email@domain.com>" formats
func extractDomainFromEmail(email string) string {
	// Extract email from "Display Name <email@domain.com>" format if present
	if strings.Contains(email, "<") && strings.Contains(email, ">") {
		start := strings.Index(email, "<")
		end := strings.Index(email, ">")
		if start >= 0 && end > start {
			email = email[start+1 : end]
		}
	}

	// Extract domain part (after @)
	parts := strings.Split(strings.TrimSpace(email), "@")
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}

// extractDisplayName extracts the display name from "Display Name <email@domain.com>" format
func extractDisplayName(email string) string {
	if strings.Contains(email, "<") && strings.Contains(email, ">") {
		start := strings.Index(email, "<")
		if start > 0 {
			return strings.TrimSpace(email[:start])
		}
	}
	return ""
}
