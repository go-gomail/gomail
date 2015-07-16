// Package gomail provides a simple interface to send emails.
//
// More info on Github: https://github.com/go-gomail/gomail
package gomail

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Message represents an email.
type Message struct {
	header      header
	parts       []part
	attachments []*File
	embedded    []*File
	charset     string
	encoding    Encoding
	hEncoder    mimeEncoder
	buf         bytes.Buffer
}

type header map[string][]string

type part struct {
	contentType string
	copier      func(io.Writer) error
}

// NewMessage creates a new message. It uses UTF-8 and quoted-printable encoding
// by default.
func NewMessage(settings ...MessageSetting) *Message {
	m := &Message{
		header:   make(header),
		charset:  "UTF-8",
		encoding: QuotedPrintable,
	}

	m.applySettings(settings)

	if m.encoding == Base64 {
		m.hEncoder = bEncoding
	} else {
		m.hEncoder = qEncoding
	}

	return m
}

// Reset resets the message so it can be reused. The message keeps its previous
// settings so it is in the same state that after a call to NewMessage.
func (m *Message) Reset() {
	for k := range m.header {
		delete(m.header, k)
	}
	m.parts = nil
	m.attachments = nil
	m.embedded = nil
}

func (m *Message) applySettings(settings []MessageSetting) {
	for _, s := range settings {
		s(m)
	}
}

// A MessageSetting can be used as an argument in NewMessage to configure an
// email.
type MessageSetting func(m *Message)

// SetCharset is a message setting to set the charset of the email.
//
// Example:
//
//	m := gomail.NewMessage(SetCharset("ISO-8859-1"))
func SetCharset(charset string) MessageSetting {
	return func(m *Message) {
		m.charset = charset
	}
}

// SetEncoding is a message setting to set the encoding of the email.
//
// Example:
//
//	m := gomail.NewMessage(SetEncoding(gomail.Base64))
func SetEncoding(enc Encoding) MessageSetting {
	return func(m *Message) {
		m.encoding = enc
	}
}

// Encoding represents a MIME encoding scheme like quoted-printable or base64.
type Encoding string

const (
	// QuotedPrintable represents the quoted-printable encoding as defined in
	// RFC 2045.
	QuotedPrintable Encoding = "quoted-printable"
	// Base64 represents the base64 encoding as defined in RFC 2045.
	Base64 Encoding = "base64"
	// Unencoded can be used to avoid encoding the body of an email. The headers
	// will still be encoded using quoted-printable encoding.
	Unencoded Encoding = "8bit"
)

// SetHeader sets a value to the given header field.
func (m *Message) SetHeader(field string, value ...string) {
	for i := range value {
		value[i] = m.encodeHeader(value[i])
	}
	m.header[field] = value
}

// SetHeaders sets the message headers.
//
// Example:
//
//	m.SetHeaders(map[string][]string{
//		"From":    {"alex@example.com"},
//		"To":      {"bob@example.com", "cora@example.com"},
//		"Subject": {"Hello"},
//	})
func (m *Message) SetHeaders(h map[string][]string) {
	for k, v := range h {
		m.SetHeader(k, v...)
	}
}

// SetAddressHeader sets an address to the given header field.
func (m *Message) SetAddressHeader(field, address, name string) {
	m.header[field] = []string{m.FormatAddress(address, name)}
}

// FormatAddress formats an address and a name as a valid RFC 5322 address.
func (m *Message) FormatAddress(address, name string) string {
	enc := m.encodeHeader(name)
	if enc == name {
		m.buf.WriteByte('"')
		for i := 0; i < len(name); i++ {
			b := name[i]
			if b == '\\' || b == '"' {
				m.buf.WriteByte('\\')
			}
			m.buf.WriteByte(b)
		}
		m.buf.WriteByte('"')
	} else if hasSpecials(name) {
		m.buf.WriteString(bEncoding.Encode(m.charset, name))
	} else {
		m.buf.WriteString(enc)
	}
	m.buf.WriteString(" <")
	m.buf.WriteString(address)
	m.buf.WriteByte('>')

	addr := m.buf.String()
	m.buf.Reset()
	return addr
}

func hasSpecials(text string) bool {
	for i := 0; i < len(text); i++ {
		switch c := text[i]; c {
		case '(', ')', '<', '>', '[', ']', ':', ';', '@', '\\', ',', '.', '"':
			return true
		}
	}

	return false
}

func (m *Message) encodeHeader(value string) string {
	return m.hEncoder.Encode(m.charset, value)
}

// SetDateHeader sets a date to the given header field.
func (m *Message) SetDateHeader(field string, date time.Time) {
	m.header[field] = []string{m.FormatDate(date)}
}

// FormatDate formats a date as a valid RFC 5322 date.
func (m *Message) FormatDate(date time.Time) string {
	return date.Format(time.RFC1123Z)
}

// GetHeader gets a header field.
func (m *Message) GetHeader(field string) []string {
	return m.header[field]
}

// DelHeader deletes a header field.
func (m *Message) DelHeader(field string) {
	delete(m.header, field)
}

// SetBody sets the body of the message.
func (m *Message) SetBody(contentType, body string) {
	m.parts = []part{
		part{
			contentType: contentType,
			copier: func(w io.Writer) error {
				_, err := io.WriteString(w, body)
				return err
			},
		},
	}
}

// AddAlternative adds an alternative part to the message. Commonly used to
// send HTML emails that default to the plain text version for backward
// compatibility.
//
// Example:
//
//	m.SetBody("text/plain", "Hello!")
//	m.AddAlternative("text/html", "<p>Hello!</p>")
//
// More info: http://en.wikipedia.org/wiki/MIME#Alternative
func (m *Message) AddAlternative(contentType, body string) {
	m.parts = append(m.parts,
		part{
			contentType: contentType,
			copier: func(w io.Writer) error {
				_, err := io.WriteString(w, body)
				return err
			},
		},
	)
}

// AddAlternativeWriter adds an alternative part to the message. It can be
// useful with the text/template and html/template packages.
//
// Example:
//
//	t := template.Must(template.New("example").Parse("Hello {{.}}!"))
//	m.AddAlternativeWriter("text/plain", func(w io.Writer) error {
//		return t.Execute(w, "Bob")
//	})
func (m *Message) AddAlternativeWriter(contentType string, f func(io.Writer) error) {
	m.parts = []part{
		part{
			contentType: contentType,
			copier:      f,
		},
	}
}

// A File represents a file that can be attached or embedded in an email.
type File struct {
	// Name represents the base name of the file. If the file is attached to the
	// message it is the name of the attachment.
	Name string
	// Header represents the MIME header of the message part that contains the
	// file content.
	Header map[string][]string
	// Copier is a function run when the message is sent. It should copy the
	// content of the file to w.
	Copier func(w io.Writer) error
}

// NewFile creates a File from the given filename.
func NewFile(filename string) *File {
	return &File{
		Name:   filepath.Base(filename),
		Header: make(map[string][]string),
		Copier: func(w io.Writer) error {
			h, err := os.Open(filename)
			if err != nil {
				return err
			}
			if _, err := io.Copy(w, h); err != nil {
				h.Close()
				return err
			}
			return h.Close()
		},
	}
}

func (f *File) setHeader(field string, value ...string) {
	f.Header[field] = value
}

// Attach attaches the files to the email.
func (m *Message) Attach(f ...*File) {
	if m.attachments == nil {
		m.attachments = f
	} else {
		m.attachments = append(m.attachments, f...)
	}
}

// Embed embeds the images to the email.
//
// Example:
//
//	f, err := gomail.OpenFile("/tmp/image.jpg")
//	if err != nil {
//		panic(err)
//	}
//	m.Embed(f)
//	m.SetBody("text/html", `<img src="cid:image.jpg" alt="My image" />`)
func (m *Message) Embed(image ...*File) {
	if m.embedded == nil {
		m.embedded = image
	} else {
		m.embedded = append(m.embedded, image...)
	}
}
