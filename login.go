package gomail

import (
	"errors"
	"fmt"
	"net/smtp"
	"strings"
)

type loginAuth struct {
	username string
	password string
	host     string
}

// LoginAuth returns an Auth that implements the LOGIN authentication mechanism.
func LoginAuth(username, password, host string) smtp.Auth {
	return &loginAuth{username, password, host}
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

	command := strings.ToLower(strings.TrimSuffix(string(fromServer), ":"))
	switch command {
	case "username":
		return []byte(fmt.Sprintf("%s", a.username)), nil
	case "password":
		return []byte(fmt.Sprintf("%s", a.password)), nil
	default:
		return nil, fmt.Errorf("gomail: unexpected server challenge: %s", command)
	}
}
