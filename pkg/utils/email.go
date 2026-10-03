package utils

import (
	"fmt"
	"net/mail"

	"github.com/pln-colabora/colabora-be/config"

	"gopkg.in/gomail.v2"
)

func SendMail(toEmail string, subject string, body string) error {
	emailConfig, err := config.NewEmailConfig()
	if err != nil {
		return err
	}

	dialer := gomail.NewDialer(
		emailConfig.Host,
		emailConfig.Port,
		emailConfig.Username,
		emailConfig.APIKey,
	)
	return sendMailWith(dialer, emailConfig, toEmail, subject, body)
}

type mailDialer interface {
	DialAndSend(...*gomail.Message) error
}

func sendMailWith(dialer mailDialer, emailConfig *config.EmailConfig, toEmail string, subject string, body string) error {
	message := gomail.NewMessage()
	from := mail.Address{Name: emailConfig.SenderName, Address: emailConfig.FromEmail}
	message.SetHeader("From", from.String())
	message.SetHeader("To", toEmail)
	message.SetHeader("Subject", subject)
	message.SetBody("text/html", body)
	if err := dialer.DialAndSend(message); err != nil {
		return fmt.Errorf("send email via SMTP: %w", err)
	}
	return nil
}
