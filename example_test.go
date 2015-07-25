package gomail_test

import (
	"fmt"
	"io"
	"time"

	"gopkg.in/gomail.v2-unstable"
)

func Example() {
	m := gomail.NewMessage()
	m.SetHeader("From", "alex@example.com")
	m.SetHeader("To", "bob@example.com", "cora@example.com")
	m.SetAddressHeader("Cc", "dan@example.com", "Dan")
	m.SetHeader("Subject", "Hello!")
	m.SetBody("text/html", "Hello <b>Bob</b> and <i>Cora</i>!")
	m.Attach(gomail.NewFile("/home/Alex/lolcat.jpg"))

	d := gomail.NewPlainDialer("smtp.example.com", "user", "123456", 587)

	// Send the email to Bob, Cora and Dan
	if err := d.DialAndSend(m); err != nil {
		panic(err)
	}
}

func Example_daemon() {
	ch := make(chan *gomail.Message)

	go func() {
		d := gomail.NewPlainDialer("smtp.example.com", "user", "123456", 587)

		var s gomail.SendCloser
		var err error
		open := false
		for {
			select {
			case m, ok := <-ch:
				if !ok {
					return
				}
				if !open {
					if s, err = d.Dial(); err != nil {
						panic(err)
					}
					open = true
				}
				if err := gomail.Send(s, m); err != nil {
					panic(err)
				}
			// Close the connection to the SMTP server if no email was sent in
			// the last 30 seconds.
			case <-time.After(30 * time.Second):
				if open {
					s.Close()
					open = false
				}
			}
		}
	}()

	// Use the channel in your program to send emails.

	// Close the channel to stop the mail daemon.
	close(ch)
}

func Example_noAuth() {
	m := gomail.NewMessage()
	m.SetHeader("From", "from@example.com")
	m.SetHeader("To", "to@example.com")
	m.SetHeader("Subject", "Hello!")
	m.SetBody("text/plain", "Hello!")

	d := gomail.Dialer{Host: "smtp.example.com", Port: 587}
	if err := d.DialAndSend(m); err != nil {
		panic(err)
	}
}

func Example_send() {
	m := gomail.NewMessage()
	m.SetHeader("From", "from@example.com")
	m.SetHeader("To", "to@example.com")
	m.SetHeader("Subject", "Hello!")
	m.SetBody("text/plain", "Hello!")

	s := gomail.SendFunc(func(from string, to []string, msg io.WriterTo) error {
		// Implements you email-sending function, for example by calling
		// an API, or running postfix, etc.
		fmt.Println("From:", from)
		fmt.Println("To:", to)
		return nil
	})

	if err := gomail.Send(s, m); err != nil {
		panic(err)
	}
	// Output:
	// From: from@example.com
	// To: [to@example.com]
}
