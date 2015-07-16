package gomail_test

import (
	"fmt"
	"io"
	"strings"

	"github.com/go-gomail/gomail"
)

func ExampleSend() {
	msg := gomail.NewMessage()
	msg.SetHeader("From", "alex@example.com")
	msg.SetHeader("To", "bob@example.com", "cora@example.com")
	msg.SetAddressHeader("Cc", "dan@example.com", "Dan")
	msg.SetHeader("Subject", "Hello!")
	msg.SetBody("text/html", "Hello <b>Bob</b> and <i>Cora</i>!")

	s := gomail.SendFunc(func(from string, to []string, msg io.WriterTo) error {
		// Implements you email-sending function, for example by calling
		// an API, or running postfix, etc.
		fmt.Println("From:", from)
		fmt.Println("To:", strings.Join(to, ", "))
		return nil
	})

	if err := gomail.Send(s, msg); err != nil {
		panic(err)
	}
	// Output:
	// From: alex@example.com
	// To: bob@example.com, cora@example.com, dan@example.com
}
