# Gomail
[![Build Status](https://travis-ci.org/go-mail/gomail.svg?branch=v2)](https://travis-ci.org/go-mail/gomail) [![Code Coverage](http://gocover.io/_badge/gopkg.in/gomail.v2)](http://gocover.io/gopkg.in/gomail.v2) [![Documentation](https://godoc.org/github.com/go-mail/gomail?status.svg)](https://godoc.org/github.com/go-mail/gomail)

**IMPORTANT** - This is a fork of [go-gomail/gomail](https://github.com/go-gomail/gomail). See this [issue#104](https://github.com/go-gomail/gomail/issues/104)


## Introduction

Gomail is a simple and efficient package to send emails. It is well tested and
documented.

Gomail can only send emails using an SMTP server. But the API is flexible and it
is easy to implement other methods for sending emails using a local Postfix, an
API, etc.


It requires Go 1.2 or newer. With Go 1.5, no external dependencies are used.


## Features

Gomail supports:
- Attachments
- Embedded images
- HTML and text templates
- Automatic encoding of special characters
- SSL and TLS
- Sending multiple emails with the same SMTP connection


## Documentation

https://godoc.org/github.com/go-mail/gomail


## Download

    go get github.com/go-mail/gomail


## Examples

See the [examples in the documentation](https://godoc.org/github.com/go-mail/gomail#example-package).


## FAQ

### x509: certificate signed by unknown authority

If you get this error it means the certificate used by the SMTP server is not
considered valid by the client running Gomail. As a quick workaround you can
bypass the verification of the server's certificate chain and host name by using
`SetTLSConfig`:

    package main

    import (
    	"crypto/tls"

    	"github.com/go-mail/gomail"
    )

    func main() {
    	d := gomail.NewDialer("smtp.example.com", 587, "user", "123456")
    	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}

        // Send emails using d.
    }

Note, however, that this is insecure and should not be used in production.


## Contribute

Contributions are more than welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for
more info.


## Change log

See [CHANGELOG.md](CHANGELOG.md).


## License

[MIT](LICENSE)


## Contact

You can ask questions on the [Gomail
thread](https://groups.google.com/d/topic/golang-nuts/jMxZHzvvEVg/discussion)
in the Go mailing-list.
