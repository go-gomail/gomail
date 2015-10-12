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
	attachments []*file
	embedded    []*file
	charset     string
	encoding    Encoding
	hEncoder    mimeEncoder
	buf         bytes.Buffer
}

type header map[string][]string

type part struct {
	header header
	copier func(io.Writer) error
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
func SetCharset(charset string) MessageSetting {
	return func(m *Message) {
		m.charset = charset
	}
}

// SetEncoding is a message setting to set the encoding of the email.
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
	m.encodeHeader(value)
	m.header[field] = value
}

func (m *Message) encodeHeader(values []string) {
	for i := range values {
		values[i] = m.encodeString(values[i])
	}
}

func (m *Message) encodeString(value string) string {
	return m.hEncoder.Encode(m.charset, value)
}

// SetHeaders sets the message headers.
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
	enc := m.encodeString(name)
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

// SetBody sets the body of the message.
func (m *Message) SetBody(contentType, body string) {
	m.parts = []part{
		{
			header: m.getPartHeader(contentType),
			copier: func(w io.Writer) error {
				_, err := io.WriteString(w, body)
				return err
			},
		},
	}
}

// AddAlternative adds an alternative part to the message.
//
// It is commonly used to send HTML emails that default to the plain text
// version for backward compatibility.
//
// More info: http://en.wikipedia.org/wiki/MIME#Alternative
func (m *Message) AddAlternative(contentType, body string) {
	m.parts = append(m.parts,
		part{
			header: m.getPartHeader(contentType),
			copier: func(w io.Writer) error {
				_, err := io.WriteString(w, body)
				return err
			},
		},
	)
}

// AddAlternativeWriter adds an alternative part to the message. It can be
// useful with the text/template or html/template packages.
func (m *Message) AddAlternativeWriter(contentType string, f func(io.Writer) error) {
	m.parts = []part{
		{
			header: m.getPartHeader(contentType),
			copier: f,
		},
	}
}

func (m *Message) getPartHeader(contentType string) header {
	return map[string][]string{
		"Content-Type":              {contentType + "; charset=" + m.charset},
		"Content-Transfer-Encoding": {string(m.encoding)},
	}
}

type file struct {
	Name     string
	Header   map[string][]string
	CopyFunc func(w io.Writer) error
}

func (f *file) setHeader(field, value string) {
	f.Header[field] = []string{value}
}

// A FileSetting can be used as an argument in Message.Attach or Message.Embed.
type FileSetting func(*file)

// SetHeader is a file setting to set the MIME header of the message part that
// contains the file content.
//
// Mandatory headers are automatically added if they are not set when sending
// the email.
func SetHeader(h map[string][]string) FileSetting {
	return func(f *file) {
		for k, v := range h {
			f.Header[k] = v
		}
	}
}

// SetCopyFunc is a file setting to replace the function that runs when the
// message is sent. It should copy the content of the file to the io.Writer.
//
// The default copy function opens the file with the given filename, and copy
// its content to the io.Writer.
func SetCopyFunc(f func(io.Writer) error) FileSetting {
	return func(fi *file) {
		fi.CopyFunc = f
	}
}

func (m *Message) appendOSFile(list []*file, name string, settings []FileSetting) []*file {
	return m.appendFile(list, &file{
		Name:   filepath.Base(name),
		Header: make(map[string][]string),
		CopyFunc: func(w io.Writer) error {
			h, err := os.Open(name)
			if err != nil {
				return err
			}
			if _, err := io.Copy(w, h); err != nil {
				h.Close()
				return err
			}
			return h.Close()
		},
	}, settings)
}

func (m *Message) appendReaderFile(list []*file, name string, r io.Reader, settings []FileSetting) []*file {
	return m.appendFile(list, &file{
		Name:   name,
		Header: make(map[string][]string),
		CopyFunc: func(w io.Writer) error {
			if _, err := io.Copy(w, r); err != nil {
				return err
			}

			return nil
		},
	}, settings)
}

func (m *Message) appendFile(list []*file, f *file, settings []FileSetting) []*file {
	for _, s := range settings {
		s(f)
	}

	if list == nil {
		return []*file{f}
	}

	return append(list, f)
}

// Attach attaches the files to the email.
func (m *Message) Attach(filename string, settings ...FileSetting) {
	m.attachments = m.appendOSFile(m.attachments, filename, settings)
}

// Embed embeds the images to the email.
func (m *Message) Embed(filename string, settings ...FileSetting) {
	m.embedded = m.appendOSFile(m.embedded, filename, settings)
}

// AttachWithReader equal to Attach using a io.Reader as content of the file.
func (m *Message) AttachWithReader(filename string, r io.Reader, settings ...FileSetting) {
	m.attachments = m.appendReaderFile(m.attachments, filename, r, settings)
}

// EmbedWithReader equal to Attach using a io.Reader as content of the file.
func (m *Message) EmbedWithReader(filename string, r io.Reader, settings ...FileSetting) {
	m.embedded = m.appendReaderFile(m.attachments, filename, r, settings)
}
