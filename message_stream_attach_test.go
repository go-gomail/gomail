package gomail

import (
	"bytes"
	"encoding/base64"
	"io"
	"path/filepath"
	"testing"
)

func mockCopyStream(name string) (io.Reader, string, FileSetting) {
	nm := filepath.Base(name)
	b := bytes.NewReader([]byte("Content of " + nm))
	return b, nm, func(f *file) {}
}
func mockCopyStreamWithHeader(m *Message, name string, h map[string][]string) (io.Reader, string, FileSetting) {
	rdr, nm, _ := mockCopyStream(name)
	return rdr, nm, SetHeader(h)
}

func TestStreamAttachmentsOnly(t *testing.T) {
	m := NewMessage()
	m.SetHeader("From", "from@example.com")
	m.SetHeader("To", "to@example.com")
	m.AttachStream(mockCopyStream("/tmp/test.pdf"))
	m.AttachStream(mockCopyStream("/tmp/test.zip"))

	want := &message{
		from: "from@example.com",
		to:   []string{"to@example.com"},
		content: "From: from@example.com\r\n" +
			"To: to@example.com\r\n" +
			"Content-Type: multipart/mixed;\r\n" +
			" boundary=_BOUNDARY_1_\r\n" +
			"\r\n" +
			"--_BOUNDARY_1_\r\n" +
			"Content-Type: application/pdf; name=\"test.pdf\"\r\n" +
			"Content-Disposition: attachment; filename=\"test.pdf\"\r\n" +
			"Content-Transfer-Encoding: base64\r\n" +
			"\r\n" +
			base64.StdEncoding.EncodeToString([]byte("Content of test.pdf")) + "\r\n" +
			"--_BOUNDARY_1_\r\n" +
			"Content-Type: application/zip; name=\"test.zip\"\r\n" +
			"Content-Disposition: attachment; filename=\"test.zip\"\r\n" +
			"Content-Transfer-Encoding: base64\r\n" +
			"\r\n" +
			base64.StdEncoding.EncodeToString([]byte("Content of test.zip")) + "\r\n" +
			"--_BOUNDARY_1_--\r\n",
	}

	testMessage(t, m, 1, want)
}

func TestStreamEmbedded(t *testing.T) {
	m := NewMessage()
	m.SetHeader("From", "from@example.com")
	m.SetHeader("To", "to@example.com")
	m.EmbedStream(mockCopyStreamWithHeader(m, "image1.jpg", map[string][]string{"Content-ID": {"<test-content-id>"}}))
	m.EmbedStream(mockCopyStream("image2.jpg"))
	m.SetBody("text/plain", "Test")

	want := &message{
		from: "from@example.com",
		to:   []string{"to@example.com"},
		content: "From: from@example.com\r\n" +
			"To: to@example.com\r\n" +
			"Content-Type: multipart/related;\r\n" +
			" boundary=_BOUNDARY_1_\r\n" +
			"\r\n" +
			"--_BOUNDARY_1_\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"Content-Transfer-Encoding: quoted-printable\r\n" +
			"\r\n" +
			"Test\r\n" +
			"--_BOUNDARY_1_\r\n" +
			"Content-Type: image/jpeg; name=\"image1.jpg\"\r\n" +
			"Content-Disposition: inline; filename=\"image1.jpg\"\r\n" +
			"Content-ID: <test-content-id>\r\n" +
			"Content-Transfer-Encoding: base64\r\n" +
			"\r\n" +
			base64.StdEncoding.EncodeToString([]byte("Content of image1.jpg")) + "\r\n" +
			"--_BOUNDARY_1_\r\n" +
			"Content-Type: image/jpeg; name=\"image2.jpg\"\r\n" +
			"Content-Disposition: inline; filename=\"image2.jpg\"\r\n" +
			"Content-ID: <image2.jpg>\r\n" +
			"Content-Transfer-Encoding: base64\r\n" +
			"\r\n" +
			base64.StdEncoding.EncodeToString([]byte("Content of image2.jpg")) + "\r\n" +
			"--_BOUNDARY_1_--\r\n",
	}

	testMessage(t, m, 1, want)
}
