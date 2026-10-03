package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEmailConfigUsesResendDefaultsAndEnvironment(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_PORT", "")
	t.Setenv("SMTP_USERNAME", "")
	t.Setenv("SMTP_SENDER_NAME", "")
	t.Setenv("SMTP_FROM_EMAIL", "noreply@example.com")
	t.Setenv("RESEND_API_KEY", "re_test_key")

	got, err := NewEmailConfig()
	require.NoError(t, err)
	assert.Equal(t, "smtp.resend.com", got.Host)
	assert.Equal(t, 465, got.Port)
	assert.Equal(t, "resend", got.Username)
	assert.Equal(t, "COLABORA", got.SenderName)
	assert.Equal(t, "noreply@example.com", got.FromEmail)
	assert.Equal(t, "re_test_key", got.APIKey)
}

func TestNewEmailConfigRejectsMissingCredentialsAndInvalidPort(t *testing.T) {
	t.Setenv("SMTP_HOST", "smtp.resend.com")
	t.Setenv("SMTP_PORT", "not-a-port")
	t.Setenv("SMTP_USERNAME", "resend")
	t.Setenv("SMTP_FROM_EMAIL", "noreply@example.com")
	t.Setenv("RESEND_API_KEY", "re_test_key")

	_, err := NewEmailConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SMTP_PORT")

	t.Setenv("SMTP_PORT", "465")
	t.Setenv("RESEND_API_KEY", "")
	_, err = NewEmailConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RESEND_API_KEY")
}
