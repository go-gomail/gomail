package gomail

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
)

// A Mailer represents an SMTP server.
type Mailer struct {
	addr   string
	host   string
	config *tls.Config
	auth   smtp.Auth
	send   SendMailFunc
}

// A MailerSetting can be used in a mailer constructor to configure it.
type MailerSetting func(m *Mailer)

// SetSendMail allows to set the email-sending function of a mailer.
//
// Example:
//
//	myFunc := func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
//		// Implement your email-sending function similar to smtp.SendMail
//	}
//	mailer := gomail.NewMailer("host", "user", "pwd", 465, SetSendMail(myFunc))
func SetSendMail(s SendMailFunc) MailerSetting {
	return func(m *Mailer) {
		m.send = s
	}
}

// SetTLSConfig allows to set the TLS configuration used to connect the SMTP
// server.
func SetTLSConfig(c *tls.Config) MailerSetting {
	return func(m *Mailer) {
		m.config = c
	}
}

// A SendMailFunc is a function to send emails with the same signature than
// smtp.SendMail.
type SendMailFunc func(addr string, a smtp.Auth, from string, to []string, msg []byte) error

// NewMailer returns a mailer. The given parameters are used to connect to the
// SMTP server via a PLAIN authentication mechanism.
func NewMailer(host string, username string, password string, port int, settings ...MailerSetting) *Mailer {
	return NewCustomMailer(
		fmt.Sprintf("%s:%d", host, port),
		smtp.PlainAuth("", username, password, host),
		settings...,
	)
}

// NewCustomMailer creates a mailer with the given authentication mechanism.
//
// Example:
//
//	gomail.NewCustomMailer("host:587", smtp.CRAMMD5Auth("username", "secret"))
func NewCustomMailer(addr string, auth smtp.Auth, settings ...MailerSetting) *Mailer {
	// Error is not handled here to preserve backward compatibility
	host, port, _ := net.SplitHostPort(addr)

	m := &Mailer{
		addr: addr,
		host: host,
		auth: auth,
	}

	for _, s := range settings {
		s(m)
	}

	if m.config == nil {
		m.config = &tls.Config{ServerName: host}
	}
	if m.send == nil {
		m.send = m.getSendMailFunc(port == "465")
	}

	return m
}

// Send sends the emails to all the recipients of the message.
func (m *Mailer) Send(msg *Message) error {
	message := msg.Export()

	from, err := getFrom(message)
	if err != nil {
		return err
	}
	recipients, bcc, err := getRecipients(message)
	if err != nil {
		return err
	}

	h := flattenHeader(message, "")
	body, err := ioutil.ReadAll(message.Body)
	if err != nil {
		return err
	}

	mail := append(h, body...)
	if err := m.send(m.addr, m.auth, from, recipients, mail); err != nil {
		return err
	}

	for _, to := range bcc {
		h = flattenHeader(message, to)
		mail = append(h, body...)
		if err := m.send(m.addr, m.auth, from, []string{to}, mail); err != nil {
			return err
		}
	}

	return nil
}

func flattenHeader(msg *mail.Message, bcc string) []byte {
	var buf bytes.Buffer
	for field, value := range msg.Header {
		if field != "Bcc" {
			buf.WriteString(field + ": " + strings.Join(value, ", ") + "\r\n")
		} else if bcc != "" {
			for _, to := range value {
				if strings.Contains(to, bcc) {
					buf.WriteString(field + ": " + to + "\r\n")
				}
			}
		}
	}
	buf.WriteString("\r\n")

	return buf.Bytes()
}

func getFrom(msg *mail.Message) (string, error) {
	from := msg.Header.Get("Sender")
	if from == "" {
		from = msg.Header.Get("From")
		if from == "" {
			return "", errors.New("mailer: invalid message, \"From\" field is absent")
		}
	}

	return parseAddress(from)
}

func getRecipients(msg *mail.Message) (recipients, bcc []string, err error) {
	for _, field := range []string{"Bcc", "To", "Cc"} {
		if addresses, ok := msg.Header[field]; ok {
			for _, addr := range addresses {
				switch field {
				case "Bcc":
					bcc, err = addAdress(bcc, addr)
				default:
					recipients, err = addAdress(recipients, addr)
				}
				if err != nil {
					return recipients, bcc, err
				}
			}
		}
	}

	return recipients, bcc, nil
}

func addAdress(list []string, addr string) ([]string, error) {
	addr, err := parseAddress(addr)
	if err != nil {
		return list, err
	}
	for _, a := range list {
		if addr == a {
			return list, nil
		}
	}

	return append(list, addr), nil
}

func parseAddress(field string) (string, error) {
	a, err := mail.ParseAddress(field)
	if a == nil {
		return "", err
	}

	return a.Address, err
}
