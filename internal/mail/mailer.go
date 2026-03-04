package mail

import (
	"fmt"
	"net/smtp"
	"os"
)

type Mailer struct {
	Host     string
	Port     string
	Email    string
	Password string
}

func NewMailer() *Mailer {
	return &Mailer{
		Host:     os.Getenv("SMTP_HOST"),
		Port:     os.Getenv("SMTP_PORT"),
		Email:    os.Getenv("SMTP_EMAIL"),
		Password: os.Getenv("SMTP_PASSWORD"),
	}
}

func (m *Mailer) Send(to, subject, body string) error {

	auth := smtp.PlainAuth("", m.Email, m.Password, m.Host)

	msg := fmt.Sprintf("From: CampusCare <%s>\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-version: 1.0;\r\n"+
		"Content-Type: text/html; charset=\"UTF-8\";\r\n\r\n"+
		"%s",
		m.Email, to, subject, body)

	return smtp.SendMail(
		m.Host+":"+m.Port,
		auth,
		m.Email,
		[]string{to},
		[]byte(msg),
	)
}
