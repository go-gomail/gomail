package gomail

import (
	"encoding/base64"
	"errors"
	"io"
	"mime/multipart"
	"mime/quotedprintable"
	"time"
)

// WriteTo implements io.WriterTo. It dumps the whole message into w.
func (msg *Message) WriteTo(w io.Writer) (int64, error) {
	mw := &messageWriter{w: w}
	mw.writeMessage(msg)
	return mw.n, mw.err
}

func (w *messageWriter) writeMessage(msg *Message) {
	if _, ok := msg.header["Mime-Version"]; !ok {
		w.writeString("Mime-Version: 1.0\r\n")
	}
	if _, ok := msg.header["Date"]; !ok {
		w.writeHeader("Date", msg.FormatDate(now()))
	}
	w.writeHeaders(msg.header)

	if msg.hasMixedPart() {
		w.openMultipart("mixed")
	}

	if msg.hasRelatedPart() {
		w.openMultipart("related")
	}

	if msg.hasAlternativePart() {
		w.openMultipart("alternative")
	}
	for _, part := range msg.parts {
		contentType := part.contentType + "; charset=" + msg.charset
		w.writeHeaders(map[string][]string{
			"Content-Type":              []string{contentType},
			"Content-Transfer-Encoding": []string{string(msg.encoding)},
		})
		w.writeBody(part.body.Bytes(), msg.encoding)
	}
	if msg.hasAlternativePart() {
		w.closeMultipart()
	}

	w.addFiles(msg.embedded, false)
	if msg.hasRelatedPart() {
		w.closeMultipart()
	}

	w.addFiles(msg.attachments, true)
	if msg.hasMixedPart() {
		w.closeMultipart()
	}
}

func (msg *Message) hasMixedPart() bool {
	return (len(msg.parts) > 0 && len(msg.attachments) > 0) || len(msg.attachments) > 1
}

func (msg *Message) hasRelatedPart() bool {
	return (len(msg.parts) > 0 && len(msg.embedded) > 0) || len(msg.embedded) > 1
}

func (msg *Message) hasAlternativePart() bool {
	return len(msg.parts) > 1
}

type messageWriter struct {
	w          io.Writer
	n          int64
	writers    [3]*multipart.Writer
	partWriter io.Writer
	depth      uint8
	err        error
}

func (w *messageWriter) openMultipart(mimeType string) {
	mw := multipart.NewWriter(w)
	contentType := "multipart/" + mimeType + "; boundary=" + mw.Boundary()
	w.writers[w.depth] = mw

	if w.depth == 0 {
		w.writeHeader("Content-Type", contentType)
		w.writeString("\r\n")
	} else {
		w.createPart(map[string][]string{
			"Content-Type": []string{contentType},
		})
	}
	w.depth++
}

func (w *messageWriter) createPart(h map[string][]string) {
	w.partWriter, w.err = w.writers[w.depth-1].CreatePart(h)
}

func (w *messageWriter) closeMultipart() {
	if w.depth > 0 {
		w.writers[w.depth-1].Close()
		w.depth--
	}
}

func (w *messageWriter) addFiles(files []*File, isAttachment bool) {
	for _, f := range files {
		h := make(map[string][]string)
		h["Content-Type"] = []string{f.MimeType + "; name=\"" + f.Name + "\""}
		h["Content-Transfer-Encoding"] = []string{string(Base64)}
		if isAttachment {
			h["Content-Disposition"] = []string{"attachment; filename=\"" + f.Name + "\""}
		} else {
			h["Content-Disposition"] = []string{"inline; filename=\"" + f.Name + "\""}
			if f.ContentID != "" {
				h["Content-ID"] = []string{"<" + f.ContentID + ">"}
			} else {
				h["Content-ID"] = []string{"<" + f.Name + ">"}
			}
		}
		w.writeHeaders(h)
		w.writeBody(f.Content, Base64)
	}
}

func (w *messageWriter) Write(p []byte) (int, error) {
	if w.err != nil {
		return 0, errors.New("gomail: cannot write as writer is in error")
	}

	var n int
	n, w.err = w.w.Write(p)
	w.n += int64(n)
	return n, w.err
}

func (w *messageWriter) writeString(s string) {
	n, _ := io.WriteString(w.w, s)
	w.n += int64(n)
}

func (w *messageWriter) writeStrings(a []string, sep string) {
	if len(a) > 0 {
		w.writeString(a[0])
		if len(a) == 1 {
			return
		}
	}
	for _, s := range a[1:] {
		w.writeString(sep)
		w.writeString(s)
	}
}

func (w *messageWriter) writeHeader(k string, v ...string) {
	w.writeString(k)
	w.writeString(": ")
	w.writeStrings(v, ", ")
	w.writeString("\r\n")
}

func (w *messageWriter) writeHeaders(h map[string][]string) {
	if w.depth == 0 {
		for k, v := range h {
			if k != "Bcc" {
				w.writeHeader(k, v...)
			}
		}
	} else {
		w.createPart(h)
	}
}

func (w *messageWriter) writeBody(body []byte, enc Encoding) {
	var subWriter io.Writer
	if w.depth == 0 {
		w.writeString("\r\n")
		subWriter = w.w
	} else {
		subWriter = w.partWriter
	}

	if enc == Base64 {
		wc := base64.NewEncoder(base64.StdEncoding, newBase64LineWriter(subWriter))
		wc.Write(body)
		wc.Close()
	} else if enc == Unencoded {
		subWriter.Write(body)
	} else {
		wc := quotedprintable.NewWriter(subWriter)
		wc.Write(body)
		wc.Close()
	}
}

// As required by RFC 2045, 6.7. (page 21) for quoted-printable, and
// RFC 2045, 6.8. (page 25) for base64.
const maxLineLen = 76

// base64LineWriter limits text encoded in base64 to 76 characters per line
type base64LineWriter struct {
	w       io.Writer
	lineLen int
}

func newBase64LineWriter(w io.Writer) *base64LineWriter {
	return &base64LineWriter{w: w}
}

func (w *base64LineWriter) Write(p []byte) (int, error) {
	n := 0
	for len(p)+w.lineLen > maxLineLen {
		w.w.Write(p[:maxLineLen-w.lineLen])
		w.w.Write([]byte("\r\n"))
		p = p[maxLineLen-w.lineLen:]
		n += maxLineLen - w.lineLen
		w.lineLen = 0
	}

	w.w.Write(p)
	w.lineLen += len(p)

	return n + len(p), nil
}

// Stubbed out for testing.
var now = time.Now
