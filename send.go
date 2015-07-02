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
	to, err := getRecipients(message)
	if err != nil {
		return err
	}

	h := flattenHeader(message)
	body, err := ioutil.ReadAll(message.Body)
	if err != nil {
		return err
	}

	mail := bytes.NewReader(append(h, body...))
	if err := s.Send(from, to, mail); err != nil {
		return err
	}

	return nil
}

func flattenHeader(msg *mail.Message) []byte {
	buf := getBuffer()
	defer putBuffer(buf)

	for field, value := range msg.Header {
		if field != "Bcc" {
			buf.WriteString(field)
			buf.WriteString(": ")
			buf.WriteString(strings.Join(value, ", "))
			buf.WriteString("\r\n")
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
			return "", errors.New("gomail: invalid message, \"From\" field is absent")
		}
	}

	return parseAddress(from)
}

func getRecipients(msg *mail.Message) ([]string, error) {
	var list []string
	for _, field := range []string{"To", "Cc", "Bcc"} {
		if addresses, ok := msg.Header[field]; ok {
			for _, a := range addresses {
				addr, err := parseAddress(a)
				if err != nil {
					return nil, err
				}
				list = addAdress(list, addr)
			}
		}
	}

	return list, nil
}

func addAdress(list []string, addr string) []string {
	for _, a := range list {
		if addr == a {
			return list
		}
	}

	return append(list, addr)
}

func parseAddress(field string) (string, error) {
	a, err := mail.ParseAddress(field)
	if a == nil {
		return "", err
	}

	return a.Address, err
}
