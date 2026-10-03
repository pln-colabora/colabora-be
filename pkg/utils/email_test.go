package utils

import (
	"bytes"
	"errors"
	"testing"

	"github.com/pln-colabora/colabora-be/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/gomail.v2"
)

type captureMailDialer struct {
	message string
	err     error
}

func (d *captureMailDialer) DialAndSend(messages ...*gomail.Message) error {
	if d.err != nil {
		return d.err
	}
	var buffer bytes.Buffer
	if _, err := messages[0].WriteTo(&buffer); err != nil {
		return err
	}
	d.message = buffer.String()
	return nil
}

func TestSendMailWithBuildsResendCompatibleMessage(t *testing.T) {
	dialer := &captureMailDialer{}
	config := &config.EmailConfig{SenderName: "COLABORA", FromEmail: "noreply@example.com"}

	err := sendMailWith(dialer, config, "user@example.com", "Verify account", "<p>Verify</p>")
	require.NoError(t, err)
	assert.Contains(t, dialer.message, `From: "COLABORA" <noreply@example.com>`)
	assert.Contains(t, dialer.message, "To: user@example.com")
	assert.Contains(t, dialer.message, "Subject: Verify account")
	assert.Contains(t, dialer.message, "Content-Type: text/html")
	assert.Contains(t, dialer.message, "<p>Verify</p>")
}

func TestSendMailWithReturnsSMTPError(t *testing.T) {
	dialer := &captureMailDialer{err: errors.New("connection refused")}
	err := sendMailWith(dialer, &config.EmailConfig{FromEmail: "noreply@example.com"}, "user@example.com", "Subject", "Body")
	require.Error(t, err)
	assert.True(t, errors.Is(err, dialer.err))
}
