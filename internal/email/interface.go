package email

// EmailClient interface for sending emails
type EmailClient interface {
	SendOTPEmail(toEmail, otpCode string) error
}

