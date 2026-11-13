package email

import (
	"bytes"
	"net/smtp"
	"text/template"
	"time"
)

// SMTPConfig holds Brevo SMTP connection settings.
type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
}

// EmailService handles sending emails via Brevo SMTP.
type EmailService struct {
	config SMTPConfig
}

// NewEmailService creates a new EmailService with the given SMTP configuration.
func NewEmailService(config SMTPConfig) *EmailService {
	return &EmailService{config: config}
}

// sendEmail sends an email using Brevo SMTP.
func (s *EmailService) sendEmail(to string, subject string, htmlTemplate string, data interface{}) error {
	tmpl, err := template.ParseFiles(htmlTemplate)
	if err != nil {
		return err
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return err
	}

	headers := make(map[string]string)
	headers["From"] = s.config.Username
	headers["To"] = to
	headers["Subject"] = subject
	headers["Content-Type"] = "text/html; charset=UTF-8"

	message := ""
	for k, v := range headers {
		message += k + ": " + v + "\r\n"
	}
	message += "\r\n" + body.String()

	addr := s.config.Host + ":" + s.config.Port

	return smtp.SendMail(addr, smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host), s.config.Username, []string{to}, []byte(message))
}

// SendVerifyEmail sends the email verification template.
func (s *EmailService) SendVerifyEmail(to string, name string, verificationURL string) error {
	data := map[string]interface{}{
		"name":        name,
		"verificationURL": verificationURL,
		"currentYear": time.Now().Year(),
	}

	return s.sendEmail(to, "Verify your email", "internal/email/templates/verify-email.html", data)
}

// SendResetPassword sends the reset password template.
func (s *EmailService) SendResetPassword(to string, name string, resetURL string) error {
	data := map[string]interface{}{
		"name":        name,
		"resetURL":    resetURL,
		"currentYear": time.Now().Year(),
	}

	return s.sendEmail(to, "Reset your password", "internal/email/templates/reset-password.html", data)
}

// SendWelcome sends the welcome email template.
func (s *EmailService) SendWelcome(to string, name string, loginURL string) error {
	data := map[string]interface{}{
		"name":      name,
		"loginURL":  loginURL,
		"currentYear": time.Now().Year(),
	}

	return s.sendEmail(to, "Welcome to Pikyon!", "internal/email/templates/welcome.html", data)
}

// SendShareMemory sends the share memory email template.
func (s *EmailService) SendShareMemory(to string, name string, memoryTitle string, memoryID string) error {
	data := map[string]interface{}{
		"name":        name,
		"memoryTitle": memoryTitle,
		"memoryID":    memoryID,
		"currentYear": time.Now().Year(),
	}

	return s.sendEmail(to, "You've been shared a memory", "internal/email/templates/share-memory.html", data)
}

// SendMemoryViewed sends the memory viewed email template.
func (s *EmailService) SendMemoryViewed(to string, name string, memoryTitle string, memoryID string) error {
	data := map[string]interface{}{
		"name":        name,
		"memoryTitle": memoryTitle,
		"memoryID":    memoryID,
		"currentYear": time.Now().Year(),
	}

	return s.sendEmail(to, "Someone viewed your memory", "internal/email/templates/memory-viewed.html", data)
}

// SendReactionReceived sends the reaction received email template.
func (s *EmailService) SendReactionReceived(to string, name string, reactingUser string, reactionType string, targetUser string) error {
	data := map[string]interface{}{
		"name":         name,
		"reactingUser": reactingUser,
		"reactionType": reactionType,
		"targetUser":   targetUser,
		"currentYear":  time.Now().Year(),
	}

	return s.sendEmail(to, "You received a reaction!", "internal/email/templates/reaction-received.html", data)
}

// SendStorageWarning sends the storage warning email template.
func (s *EmailService) SendStorageWarning(to string, name string, used int64, quota int64, percent int, deleteOption bool) error {
	data := map[string]interface{}{
		"name":        name,
		"used":        used,
		"quota":       quota,
		"percent":     percent,
		"deleteOption": deleteOption,
		"currentYear": time.Now().Year(),
	}

	return s.sendEmail(to, "Storage warning - Pikyon", "internal/email/templates/storage-warning.html", data)
}