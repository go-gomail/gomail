// Package gomail provides a simple interface to send emails.
//
// More info on Github: https://github.com/go-gomail/gomail
package gomail

import (
	"bytes"
	"io"
	"io/ioutil"
	"mime"
	"path/filepath"
	"sync"
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
	hEncoder    mime.WordEncoder
}

type header map[string][]string

type part struct {
	contentType string
	copier      func(io.Writer) error
}

// NewMessage creates a new message. It uses UTF-8 and quoted-printable encoding
// by default.
func NewMessage(settings ...MessageSetting) *Message {
	msg := &Message{
		header:   make(header),
		charset:  "UTF-8",
		encoding: QuotedPrintable,
	}

	msg.applySettings(settings)

	if msg.encoding == Base64 {
		msg.hEncoder = mime.BEncoding
	} else {
		msg.hEncoder = mime.QEncoding
	}

	return msg
}

func (msg *Message) applySettings(settings []MessageSetting) {
	for _, s := range settings {
		s(msg)
	}
}

// A MessageSetting can be used as an argument in NewMessage to configure an
// email.
type MessageSetting func(msg *Message)

// SetCharset is a message setting to set the charset of the email.
//
// Example:
//
//	msg := gomail.NewMessage(SetCharset("ISO-8859-1"))
func SetCharset(charset string) MessageSetting {
	return func(msg *Message) {
		msg.charset = charset
	}
}

// SetEncoding is a message setting to set the encoding of the email.
//
// Example:
//
//	msg := gomail.NewMessage(SetEncoding(gomail.Base64))
func SetEncoding(enc Encoding) MessageSetting {
	return func(msg *Message) {
		msg.encoding = enc
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
func (msg *Message) SetHeader(field string, value ...string) {
	for i := range value {
		value[i] = msg.encodeHeader(value[i])
	}
	msg.header[field] = value
}

// SetHeaders sets the message headers.
//
// Example:
//
//	msg.SetHeaders(map[string][]string{
//		"From":    {"alex@example.com"},
//		"To":      {"bob@example.com", "cora@example.com"},
//		"Subject": {"Hello"},
//	})
func (msg *Message) SetHeaders(h map[string][]string) {
	for k, v := range h {
		msg.SetHeader(k, v...)
	}
}

// SetAddressHeader sets an address to the given header field.
func (msg *Message) SetAddressHeader(field, address, name string) {
	msg.header[field] = []string{msg.FormatAddress(address, name)}
}

// FormatAddress formats an address and a name as a valid RFC 5322 address.
func (msg *Message) FormatAddress(address, name string) string {
	buf := getBuffer()
	defer putBuffer(buf)

	enc := msg.encodeHeader(name)
	if enc == name {
		quote(buf, name)
	} else if hasSpecials(name) {
		buf.WriteString(mime.BEncoding.Encode(msg.charset, name))
	} else {
		buf.WriteString(enc)
	}
	buf.WriteString(" <")
	buf.WriteString(address)
	buf.WriteByte('>')

	return buf.String()
}

// SetDateHeader sets a date to the given header field.
func (msg *Message) SetDateHeader(field string, date time.Time) {
	msg.header[field] = []string{msg.FormatDate(date)}
}

// FormatDate formats a date as a valid RFC 5322 date.
func (msg *Message) FormatDate(date time.Time) string {
	return date.Format(time.RFC1123Z)
}

// GetHeader gets a header field.
func (msg *Message) GetHeader(field string) []string {
	return msg.header[field]
}

// DelHeader deletes a header field.
func (msg *Message) DelHeader(field string) {
	delete(msg.header, field)
}

// SetBody sets the body of the message.
func (msg *Message) SetBody(contentType, body string) {
	msg.parts = []part{
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
//	msg.SetBody("text/plain", "Hello!")
//	msg.AddAlternative("text/html", "<p>Hello!</p>")
//
// More info: http://en.wikipedia.org/wiki/MIME#Alternative
func (msg *Message) AddAlternative(contentType, body string) {
	msg.parts = append(msg.parts,
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
//	msg.AddAlternativeWriter("text/plain", func(w io.Writer) error {
//		return t.Execute(w, "Bob")
//	})
func (msg *Message) AddAlternativeWriter(contentType string, f func(io.Writer) error) {
	msg.parts = []part{
		part{
			contentType: contentType,
			copier:      f,
		},
	}
}

// A File represents a file that can be attached or embedded in an email.
type File struct {
	Name      string
	MimeType  string
	Content   []byte
	ContentID string
}

// OpenFile opens a file on disk to create a gomail.File.
func OpenFile(filename string) (*File, error) {
	content, err := readFile(filename)
	if err != nil {
		return nil, err
	}

	f := CreateFile(filepath.Base(filename), content)

	return f, nil
}

// CreateFile creates a gomail.File from the given name and content.
func CreateFile(name string, content []byte) *File {
	mimeType := mime.TypeByExtension(filepath.Ext(name))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	return &File{
		Name:     name,
		MimeType: mimeType,
		Content:  content,
	}
}

// Attach attaches the files to the email.
func (msg *Message) Attach(f ...*File) {
	if msg.attachments == nil {
		msg.attachments = f
	} else {
		msg.attachments = append(msg.attachments, f...)
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
//	msg.Embed(f)
//	msg.SetBody("text/html", `<img src="cid:image.jpg" alt="My image" />`)
func (msg *Message) Embed(image ...*File) {
	if msg.embedded == nil {
		msg.embedded = image
	} else {
		msg.embedded = append(msg.embedded, image...)
	}
}

// Stubbed out for testing.
var readFile = ioutil.ReadFile

func quote(buf *bytes.Buffer, text string) {
	buf.WriteByte('"')
	for i := 0; i < len(text); i++ {
		if text[i] == '\\' || text[i] == '"' {
			buf.WriteByte('\\')
		}
		buf.WriteByte(text[i])
	}
	buf.WriteByte('"')
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

func (msg *Message) encodeHeader(value string) string {
	return msg.hEncoder.Encode(msg.charset, value)
}

var bufPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func getBuffer() *bytes.Buffer {
	return bufPool.Get().(*bytes.Buffer)
}

func putBuffer(buf *bytes.Buffer) {
	if buf.Len() > 1024 {
		return
	}
	buf.Reset()
	bufPool.Put(buf)
}
