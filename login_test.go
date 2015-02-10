package gomail

import (
	"net/smtp"
	"testing"
)

type output struct {
	proto string
	data  []string
	err   error
}

const (
	testUser = "user"
	testPwd  = "pwd"
)

func TestPlainAuth(t *testing.T) {
	tests := []struct {
		serverProtos     []string
		serverChallenges []string
		proto            string
		data             []string
	}{
		{
			serverProtos:     []string{"LOGIN"},
			serverChallenges: []string{"Username:", "Password:"},
			proto:            "LOGIN",
			data:             []string{"", testUser, testPwd},
		},
	}

	for _, test := range tests {
		auth := LoginAuth(testUser, testPwd, testHost)
		server := &smtp.ServerInfo{
			Name: testHost,
			TLS:  true,
			Auth: test.serverProtos,
		}
		proto, toServer, err := auth.Start(server)
		if err != nil {
			t.Fatalf("Start error: %v", err)
		}
		if proto != test.proto {
			t.Errorf("Invalid protocol, got %q, want %q", proto, test.proto)
		}

		i := 0
		got := string(toServer)
		if got != test.data[i] {
			t.Errorf("Invalid response, got %q, want %q", got, test.data[i])
		}
		for _, challenge := range test.serverChallenges {
			toServer, err = auth.Next([]byte(challenge), true)
			if err != nil {
				t.Fatalf("Auth error: %v", err)
			}
			i++
			got = string(toServer)
			if got != test.data[i] {
				t.Errorf("Invalid response, got %q, want %q", got, test.data[i])
			}
		}
	}
}
