package gomail

import (
	"crypto/tls"
	"io"
	"net"
	"net/smtp"
	"testing"
)

var (
	testAddr    = "smtp.example.com:587"
	testSSLAddr = "smtp.example.com:465"
	testTLSConn = &tls.Conn{}
	testConfig  = &tls.Config{InsecureSkipVerify: true}
	testHost    = "smtp.example.com"
	testAuth    = smtp.PlainAuth("", "user", "pwd", "smtp.example.com")
	testFrom    = "from@example.com"
	testTo      = []string{"to1@example.com", "to2@example.com"}
	testBody    = "Test message"
)

const wantMsg = "To: to1@example.com, to2@example.com\r\n" +
	"From: from@example.com\r\n" +
	"Mime-Version: 1.0\r\n" +
	"Date: Wed, 25 Jun 2014 17:46:00 +0000\r\n" +
	"Content-Type: text/plain; charset=UTF-8\r\n" +
	"Content-Transfer-Encoding: quoted-printable\r\n" +
	"\r\n" +
	"Test message"

func TestDefaultSendMail(t *testing.T) {
	testSendMail(t, testAddr, nil, []string{
		"Extension STARTTLS",
		"StartTLS",
		"Extension AUTH",
		"Auth",
		"Mail " + testFrom,
		"Rcpt " + testTo[0],
		"Rcpt " + testTo[1],
		"Data",
		"Write message",
		"Close writer",
		"Quit",
		"Close",
	})
}

func TestSSLSendMail(t *testing.T) {
	testSendMail(t, testSSLAddr, nil, []string{
		"Extension AUTH",
		"Auth",
		"Mail " + testFrom,
		"Rcpt " + testTo[0],
		"Rcpt " + testTo[1],
		"Data",
		"Write message",
		"Close writer",
		"Quit",
		"Close",
	})
}

func TestTLSConfigSendMail(t *testing.T) {
	testSendMail(t, testAddr, testConfig, []string{
		"Extension STARTTLS",
		"StartTLS",
		"Extension AUTH",
		"Auth",
		"Mail " + testFrom,
		"Rcpt " + testTo[0],
		"Rcpt " + testTo[1],
		"Data",
		"Write message",
		"Close writer",
		"Quit",
		"Close",
	})
}

func TestTLSConfigSSLSendMail(t *testing.T) {
	testSendMail(t, testSSLAddr, testConfig, []string{
		"Extension AUTH",
		"Auth",
		"Mail " + testFrom,
		"Rcpt " + testTo[0],
		"Rcpt " + testTo[1],
		"Data",
		"Write message",
		"Close writer",
		"Quit",
		"Close",
	})
}

type mockClient struct {
	t      *testing.T
	i      int
	want   []string
	addr   string
	auth   smtp.Auth
	config *tls.Config
}

func (c *mockClient) Extension(ext string) (bool, string) {
	c.do("Extension " + ext)
	return true, ""
}

func (c *mockClient) StartTLS(config *tls.Config) error {
	assertConfig(c.t, config, c.config)
	c.do("StartTLS")
	return nil
}

func (c *mockClient) Auth(a smtp.Auth) error {
	assertAuth(c.t, a, c.auth)
	c.do("Auth")
	return nil
}

func (c *mockClient) Mail(from string) error {
	c.do("Mail " + from)
	return nil
}

func (c *mockClient) Rcpt(to string) error {
	c.do("Rcpt " + to)
	return nil
}

func (c *mockClient) Data() (io.WriteCloser, error) {
	c.do("Data")
	return &mockWriter{c: c, want: wantMsg}, nil
}

func (c *mockClient) Quit() error {
	c.do("Quit")
	return nil
}

func (c *mockClient) Close() error {
	c.do("Close")
	return nil
}

func (c *mockClient) do(cmd string) {
	if c.i >= len(c.want) {
		c.t.Fatalf("Invalid command %q", cmd)
	}

	if cmd != c.want[c.i] {
		c.t.Fatalf("Invalid command, got %q, want %q", cmd, c.want[c.i])
	}
	c.i++
}

type mockWriter struct {
	want string
	c    *mockClient
}

func (w *mockWriter) Write(p []byte) (int, error) {
	w.c.do("Write message")
	compareBodies(w.c.t, string(p), w.want)
	return len(p), nil
}

func (w *mockWriter) Close() error {
	w.c.do("Close writer")
	return nil
}

func testSendMail(t *testing.T, addr string, config *tls.Config, want []string) {
	testClient := &mockClient{
		t:      t,
		want:   want,
		addr:   addr,
		auth:   testAuth,
		config: config,
	}

	initSMTP = func(addr string) (smtpClient, error) {
		assertAddr(t, addr, testClient.addr)
		return testClient, nil
	}

	initTLS = func(network, addr string, config *tls.Config) (*tls.Conn, error) {
		if network != "tcp" {
			t.Errorf("Invalid network, got %q, want tcp", network)
		}
		assertAddr(t, addr, testClient.addr)
		assertConfig(t, config, testClient.config)
		return testTLSConn, nil
	}

	newClient = func(conn net.Conn, host string) (smtpClient, error) {
		if conn != testTLSConn {
			t.Error("Invalid TLS connection used")
		}
		if host != testHost {
			t.Errorf("Invalid host, got %q, want %q", host, testHost)
		}
		return testClient, nil
	}

	msg := NewMessage()
	msg.SetHeader("From", testFrom)
	msg.SetHeader("To", testTo...)
	msg.SetBody("text/plain", testBody)

	var settings []MailerSetting
	if config != nil {
		settings = []MailerSetting{SetTLSConfig(config)}
	}

	mailer := NewCustomMailer(addr, testAuth, settings...)
	if err := mailer.Send(msg); err != nil {
		t.Error(err)
	}
}

func assertAuth(t *testing.T, got, want smtp.Auth) {
	if got != want {
		t.Errorf("Invalid auth, got %#v, want %#v", got, want)
	}
}

func assertAddr(t *testing.T, got, want string) {
	if got != want {
		t.Errorf("Invalid addr, got %q, want %q", got, want)
	}
}

func assertConfig(t *testing.T, got, want *tls.Config) {
	if want == nil {
		want = &tls.Config{ServerName: testHost}
	}
	if got.ServerName != want.ServerName {
		t.Errorf("Invalid field ServerName in config, got %q, want %q", got.ServerName, want.ServerName)
	}
	if got.InsecureSkipVerify != want.InsecureSkipVerify {
		t.Errorf("Invalid field InsecureSkipVerify in config, got %v, want %v", got.InsecureSkipVerify, want.InsecureSkipVerify)
	}
}
