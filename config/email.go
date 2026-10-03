package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// EmailConfig contains the SMTP connection and sender settings.
type EmailConfig struct {
	Host       string
	Port       int
	Username   string
	APIKey     string
	SenderName string
	FromEmail  string
}

// NewEmailConfig loads optional local .env values and then reads the effective
// configuration from the process environment.
func NewEmailConfig() (*EmailConfig, error) {
	// A missing .env is expected in deployed environments, where configuration
	// is supplied directly through environment variables.
	_ = godotenv.Load(".env")

	port := 465
	if value := os.Getenv("SMTP_PORT"); value != "" {
		parsedPort, err := strconv.Atoi(value)
		if err != nil || parsedPort < 1 || parsedPort > 65535 {
			return nil, fmt.Errorf("invalid SMTP_PORT %q", value)
		}
		port = parsedPort
	}

	config := &EmailConfig{
		Host:       getEnvOrDefault("SMTP_HOST", "smtp.resend.com"),
		Port:       port,
		Username:   getEnvOrDefault("SMTP_USERNAME", "resend"),
		APIKey:     os.Getenv("RESEND_API_KEY"),
		SenderName: getEnvOrDefault("SMTP_SENDER_NAME", "COLABORA"),
		FromEmail:  os.Getenv("SMTP_FROM_EMAIL"),
	}
	if config.APIKey == "" {
		return nil, fmt.Errorf("RESEND_API_KEY is required")
	}
	if config.FromEmail == "" {
		return nil, fmt.Errorf("SMTP_FROM_EMAIL is required")
	}
	if config.Host == "" || config.Username == "" {
		return nil, fmt.Errorf("SMTP_HOST and SMTP_USERNAME must not be empty")
	}

	return config, nil
}
