package gomail

import (
	"crypto/tls"
	"io"
	"net/smtp"
)

func (m *Mailer) getSendMailFunc() SendMailFunc {
	return func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		c, err := initSMTP(addr)
		if err != nil {
			return err
		}
		if ok, _ := c.Extension("STARTTLS"); ok {
			return c.StartTLS(m.config)
		}
		defer c.Close()

		if a != nil {
			if ok, _ := c.Extension("AUTH"); ok {
				if err = c.Auth(a); err != nil {
					return err
				}
			}
		}

		if err = c.Mail(from); err != nil {
			return err
		}

		for _, addr := range to {
			if err = c.Rcpt(addr); err != nil {
				return err
			}
		}

		w, err := c.Data()
		if err != nil {
			return err
		}
		_, err = w.Write(msg)
		if err != nil {
			return err
		}
		err = w.Close()
		if err != nil {
			return err
		}

		return c.Quit()
	}
}

var initSMTP = func(addr string) (smtpClient, error) {
	return smtp.Dial(addr)
}

type smtpClient interface {
	Extension(string) (bool, string)
	StartTLS(*tls.Config) error
	Auth(smtp.Auth) error
	Mail(string) error
	Rcpt(string) error
	Data() (io.WriteCloser, error)
	Quit() error
	Close() error
}
