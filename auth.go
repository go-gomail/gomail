package gomail

import (
	"errors"
	"fmt"
	"net/smtp"
	"strings"
)

// loginAuth is an smtp.Auth that implements the LOGIN authentication mechanism.
type loginAuth struct {
	username string
	password string
	host     string
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	if !server.TLS {
		advertised := false
		for _, mechanism := range server.Auth {
			if mechanism == "LOGIN" {
				advertised = true
				break
			}
		}
		if !advertised {
			return "", nil, errors.New("gomail: unencrypted connection")
		}
	}
	if server.Name != a.host {
		return "", nil, errors.New("gomail: wrong host name")
	}
	return "LOGIN", nil, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}

	cmd := string(fromServer)
	cmd = strings.TrimSpace(cmd)
	cmd = strings.TrimSuffix(cmd, ":")
	cmd = strings.ToLower(cmd)

	switch {
	case strings.EqualFold(cmd,"username"):
		return []byte(a.username), nil
	case strings.EqualFold(cmd,"password"):
		return []byte(a.password), nil
	default:
		return nil, fmt.Errorf("gomail: unexpected server challenge: %s", cmd)
	}
}