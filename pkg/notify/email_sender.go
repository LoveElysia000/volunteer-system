package notify

import (
	"errors"
	"fmt"
	"net/smtp"
	"strings"
	"volunteer-system/config"
)

// SendEmail sends an email using SMTP config.
func SendEmail(cfg *config.EmailConfig, to, subject, body string) error {
	if cfg == nil || !cfg.Enabled {
		return nil
	}
	if strings.TrimSpace(to) == "" {
		return errors.New("email receiver is empty")
	}
	if strings.TrimSpace(cfg.SMTP.Host) == "" || cfg.SMTP.Port <= 0 {
		return errors.New("smtp config is invalid")
	}
	if strings.TrimSpace(cfg.From.Address) == "" {
		return errors.New("email from address is empty")
	}

	hostPort := fmt.Sprintf("%s:%d", cfg.SMTP.Host, cfg.SMTP.Port)
	headers := []string{
		"From: " + cfg.From.Address,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
	}
	msg := strings.Join(headers, "\r\n") + "\r\n\r\n" + body

	var auth smtp.Auth
	if strings.TrimSpace(cfg.SMTP.User) != "" {
		auth = smtp.PlainAuth("", cfg.SMTP.User, cfg.SMTP.Pass, cfg.SMTP.Host)
	}

	return smtp.SendMail(hostPort, auth, cfg.From.Address, []string{to}, []byte(msg))
}
