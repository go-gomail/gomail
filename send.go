package gomail

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/mail"
	"strings"
)

// Sender is the interface that wraps the Send method.
//
// Send sends an email to the given addresses.
type Sender interface {
	Send(from string, to []string, msg io.Reader) error
}

// SendCloser is the interface that groups the Send and Close methods.
type SendCloser interface {
	Sender
	Close() error
}

// A SendFunc is a function that sends emails to the given adresses.
// The SendFunc type is an adapter to allow the use of ordinary functions as
// email senders. If f is a function with the appropriate signature, SendFunc(f)
// is a Sender object that calls f.
type SendFunc func(from string, to []string, msg io.Reader) error

// Send calls f(from, to, msg).
func (f SendFunc) Send(from string, to []string, msg io.Reader) error {
	return f(from, to, msg)
}

// Send sends emails using the given Sender.
func Send(s Sender, msg ...*Message) error {
	for i, m := range msg {
		if err := send(s, m); err != nil {
			return fmt.Errorf("gomail: could not send email %d: %v", i+1, err)
		}
	}

	return nil
}

func send(s Sender, msg *Message) error {
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

	mail := bytes.NewReader(append(h, body...))
	if err := s.Send(from, recipients, mail); err != nil {
		return err
	}

	for _, to := range bcc {
		h = flattenHeader(message, to)
		mail = bytes.NewReader(append(h, body...))
		if err := s.Send(from, []string{to}, mail); err != nil {
			return err
		}
	}

	return nil
}

func flattenHeader(msg *mail.Message, bcc string) []byte {
	buf := getBuffer()
	defer putBuffer(buf)

	for field, value := range msg.Header {
		if field != "Bcc" {
			buf.WriteString(field)
			buf.WriteString(": ")
			buf.WriteString(strings.Join(value, ", "))
			buf.WriteString("\r\n")
		} else if bcc != "" {
			for _, to := range value {
				if strings.Contains(to, bcc) {
					buf.WriteString(field)
					buf.WriteString(": ")
					buf.WriteString(to)
					buf.WriteString("\r\n")
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
