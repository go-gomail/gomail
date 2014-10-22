package gomail

import (
	"crypto/tls"
	"io"
	"net"
	"net/smtp"
)

func (m *Mailer) getSendMailFunc(ssl bool) SendMailFunc {
	return func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		var c smtpClient
		var err error
		if ssl {
			c, err = sslDial(addr, m.host, m.config)
		} else {
			c, err = starttlsDial(addr, m.config)
		}
		if err != nil {
			return err
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

func sslDial(addr, host string, config *tls.Config) (smtpClient, error) {
	conn, err := initTLS("tcp", addr, config)
	if err != nil {
		return nil, err
	}

	return newClient(conn, host)
}

func starttlsDial(addr string, config *tls.Config) (smtpClient, error) {
	c, err := initSMTP(addr)
	if err != nil {
		return c, err
	}

	if ok, _ := c.Extension("STARTTLS"); ok {
		return c, c.StartTLS(config)
	}

	return c, nil
}

var initSMTP = func(addr string) (smtpClient, error) {
	return smtp.Dial(addr)
}

var initTLS = func(network, addr string, config *tls.Config) (*tls.Conn, error) {
	return tls.Dial(network, addr, config)
}

var newClient = func(conn net.Conn, host string) (smtpClient, error) {
	return smtp.NewClient(conn, host)
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
