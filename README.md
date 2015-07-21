# Gomail

## Introduction

Gomail is a very simple and powerful package to send emails.

It requires Go 1.2 or newer.


## Features

 * Dead-simple API
 * Highly flexible
 * Backward compatibility promise
 * Supports HTML and text templates
 * Attachments
 * Embedded images
 * SSL/TLS support
 * Automatic encoding of special characters
 * Well-documented
 * High test coverage


## Documentation

https://godoc.org/gopkg.in/gomail.v1


## Download

    go get gopkg.in/gomail.v1


## Example

```go
package main

import (
	"gopkg.in/gomail.v1"
)

func main() {
	msg := gomail.NewMessage()
	msg.SetHeader("From", "alex@example.com")
	msg.SetHeader("To", "bob@example.com", "cora@example.com")
	msg.SetAddressHeader("Cc", "dan@example.com", "Dan")
	msg.SetHeader("Subject", "Hello!")
	msg.SetBody("text/html", "Hello <b>Bob</b> and <i>Cora</i>!")
	msg.Attach(gomail.NewFile("/home/Alex/lolcat.jpg"))

	d := gomail.NewPlainDialer("smtp.example.com", "user", "123456", 587)

	// Send the email to Bob, Cora and Dan
	if err := d.DialAndSend(msg); err != nil {
		panic(err)
	}
}
```


## FAQ

### x509: certificate signed by unknown authority

If you get this error it means the certificate used by the SMTP server is not
considered valid by the client running Gomail. As a quick workaround you can
bypass the verification of the server's certificate chain and host name by using
`SetTLSConfig`:

    d := gomail.NewPlainDialer("smtp.example.com", "user", "123456", 587)
    d.TLSConfig = &tls.Config{InsecureSkipVerify: true}

Note, however, that this is insecure and should not be used in production.


## Contribute

See [CONTRIBUTING.md](CONTRIBUTING.md).


## Change log

See [CHANGELOG.md](CHANGELOG.md).


## Contact

You can ask questions on the [Gomail
thread](https://groups.google.com/d/topic/golang-nuts/ywPpNlmSt6U/discussion)
in the Go mailing-list.


## Support

If you want to support the development of Gomail, I gladly accept donations.

I will give 100% of the money I receive to
[Enfants, Espoir Du Monde](http://www.eedm.fr/).
EEDM is a French NGO which helps children in Bangladesh, Cameroun, Haiti, India
and Madagascar. All its members are volunteers so its operating costs are only
1.9%. So your money will directly helps children of these countries.

As an added bonus, your donations will also tip me by lowering my taxes :smile:
I will send an email with the receipt of the donation to EEDM annually to all
donors.

[![foo](https://www.paypalobjects.com/en_US/i/btn/btn_donate_LG.gif)](https://www.paypal.com/cgi-bin/webscr?cmd=_s-xclick&hosted_button_id=PYQKC7VFVXCFG)
