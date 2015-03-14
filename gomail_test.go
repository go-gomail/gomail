package gomail

import (
	"encoding/base64"
	"net/smtp"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

type message struct {
	from    string
	to      []string
	content string
}

func TestMessage(t *testing.T) {
	msg := NewMessage()
	msg.SetAddressHeader("From", "from@example.com", "Señor From")
	msg.SetHeader("To", msg.FormatAddress("to@example.com", "Señor To"), "tobis@example.com")
	msg.SetAddressHeader("Cc", "cc@example.com", "A, B")
	msg.SetAddressHeader("X-To", "ccbis@example.com", "à, b")
	msg.SetDateHeader("X-Date", stubNow())
	msg.SetHeader("X-Date-2", msg.FormatDate(stubNow()))
	msg.SetHeader("Subject", "¡Hola, señor!")
	msg.SetHeaders(map[string][]string{
		"X-Headers": {"Test", "Café"},
	})
	msg.SetBody("text/plain", "¡Hola, señor!")

	want := message{
		from: "from@example.com",
		to: []string{
			"to@example.com",
			"tobis@example.com",
			"cc@example.com",
		},
		content: "From: =?UTF-8?Q?Se=C3=B1or_From?= <from@example.com>\r\n" +
			"To: =?UTF-8?Q?Se=C3=B1or_To?= <to@example.com>, tobis@example.com\r\n" +
			"Cc: \"A, B\" <cc@example.com>\r\n" +
			"X-To: =?UTF-8?B?w6AsIGI=?= <ccbis@example.com>\r\n" +
			"X-Date: Wed, 25 Jun 2014 17:46:00 +0000\r\n" +
			"X-Date-2: Wed, 25 Jun 2014 17:46:00 +0000\r\n" +
			"X-Headers: Test, =?UTF-8?Q?Caf=C3=A9?=\r\n" +
			"Subject: =?UTF-8?Q?=C2=A1Hola,_se=C3=B1or!?=\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"Content-Transfer-Encoding: quoted-printable\r\n" +
			"\r\n" +
			"=C2=A1Hola, se=C3=B1or!",
	}

	testMessage(t, msg, 0, want)
}

func TestBodyWriter(t *testing.T) {
	msg := NewMessage()
	msg.SetHeader("From", "from@example.com")
	msg.SetHeader("To", "to@example.com")
	w := msg.GetBodyWriter("text/plain")
	w.Write([]byte("Test message"))

	want := message{
		from: "from@example.com",
		to:   []string{"to@example.com"},
		content: "From: from@example.com\r\n" +
			"To: to@example.com\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"Content-Transfer-Encoding: quoted-printable\r\n" +
			"\r\n" +
			"Test message",
	}

	testMessage(t, msg, 0, want)
}

func TestCustomMessage(t *testing.T) {
	msg := NewMessage(SetCharset("ISO-8859-1"), SetEncoding(Base64))
	msg.SetHeaders(map[string][]string{
		"From":    {"from@example.com"},
		"To":      {"to@example.com"},
		"Subject": {"Café"},
	})
	msg.SetBody("text/html", "¡Hola, señor!")

	want := message{
		from: "from@example.com",
		to:   []string{"to@example.com"},
		content: "From: from@example.com\r\n" +
			"To: to@example.com\r\n" +
			"Subject: =?ISO-8859-1?B?Q2Fmw6k=?=\r\n" +
			"Content-Type: text/html; charset=ISO-8859-1\r\n" +
			"Content-Transfer-Encoding: base64\r\n" +
			"\r\n" +
			"wqFIb2xhLCBzZcOxb3Ih",
	}

	testMessage(t, msg, 0, want)
}

func TestUnencodedMessage(t *testing.T) {
	msg := NewMessage(SetEncoding(Unencoded))
	msg.SetHeaders(map[string][]string{
		"From":    {"from@example.com"},
		"To":      {"to@example.com"},
		"Subject": {"Café"},
	})
	msg.SetBody("text/html", "¡Hola, señor!")

	want := message{
		from: "from@example.com",
		to:   []string{"to@example.com"},
		content: "From: from@example.com\r\n" +
			"To: to@example.com\r\n" +
			"Subject: =?UTF-8?Q?Caf=C3=A9?=\r\n" +
			"Content-Type: text/html; charset=UTF-8\r\n" +
			"Content-Transfer-Encoding: 8bit\r\n" +
			"\r\n" +
			"¡Hola, señor!",
	}

	testMessage(t, msg, 0, want)
}

func TestRecipients(t *testing.T) {
	msg := NewMessage()
	msg.SetHeaders(map[string][]string{
		"From":    {"from@example.com"},
		"To":      {"to@example.com"},
		"Cc":      {"cc@example.com"},
		"Bcc":     {"bcc1@example.com", "bcc2@example.com"},
		"Subject": {"Hello!"},
	})
	msg.SetBody("text/plain", "Test message")

	want := message{
		from: "from@example.com",
		to:   []string{"to@example.com", "cc@example.com"},
		content: "From: from@example.com\r\n" +
			"To: to@example.com\r\n" +
			"Cc: cc@example.com\r\n" +
			"Subject: Hello!\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"Content-Transfer-Encoding: quoted-printable\r\n" +
			"\r\n" +
			"Test message",
	}
	wantBcc1 := message{
		from: "from@example.com",
		to:   []string{"bcc1@example.com"},
		content: "From: from@example.com\r\n" +
			"To: to@example.com\r\n" +
			"Cc: cc@example.com\r\n" +
			"Bcc: bcc1@example.com\r\n" +
			"Subject: Hello!\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"Content-Transfer-Encoding: quoted-printable\r\n" +
			"\r\n" +
			"Test message",
	}
	wantBcc2 := message{
		from: "from@example.com",
		to:   []string{"bcc2@example.com"},
		content: "From: from@example.com\r\n" +
			"To: to@example.com\r\n" +
			"Cc: cc@example.com\r\n" +
			"Bcc: bcc2@example.com\r\n" +
			"Subject: Hello!\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"Content-Transfer-Encoding: quoted-printable\r\n" +
			"\r\n" +
			"Test message",
	}

	testMessage(t, msg, 0, want, wantBcc1, wantBcc2)
}

func TestAlternative(t *testing.T) {
	msg := NewMessage()
	msg.SetHeader("From", "from@example.com")
	msg.SetHeader("To", "to@example.com")
	msg.SetBody("text/plain", "¡Hola, señor!")
	msg.AddAlternative("text/html", "¡<b>Hola</b>, <i>señor</i>!</h1>")

	want := message{
		from: "from@example.com",
		to:   []string{"to@example.com"},
		content: "From: from@example.com\r\n" +
			"To: to@example.com\r\n" +
			"Content-Type: multipart/alternative; boundary=_BOUNDARY_1_\r\n" +
			"\r\n" +
			"--_BOUNDARY_1_\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"Content-Transfer-Encoding: quoted-printable\r\n" +
			"\r\n" +
			"=C2=A1Hola, se=C3=B1or!\r\n" +
			"--_BOUNDARY_1_\r\n" +
			"Content-Type: text/html; charset=UTF-8\r\n" +
			"Content-Transfer-Encoding: quoted-printable\r\n" +
			"\r\n" +
			"=C2=A1<b>Hola</b>, <i>se=C3=B1or</i>!</h1>\r\n" +
			"--_BOUNDARY_1_--\r\n",
	}

	testMessage(t, msg, 1, want)
}

func TestAttachmentOnly(t *testing.T) {
	readFile = func(filename string) ([]byte, error) {
		return []byte("Content of " + filepath.Base(filename)), nil
	}

	msg := NewMessage()
	msg.SetHeader("From", "from@example.com")
	msg.SetHeader("To", "to@example.com")
	f, err := OpenFile("/tmp/test.pdf")
	if err != nil {
		panic(err)
	}
	msg.Attach(f)

	want := message{
		from: "from@example.com",
		to:   []string{"to@example.com"},
		content: "From: from@example.com\r\n" +
			"To: to@example.com\r\n" +
			"Content-Type: application/pdf; name=\"test.pdf\"\r\n" +
			"Content-Disposition: attachment; filename=\"test.pdf\"\r\n" +
			"Content-Transfer-Encoding: base64\r\n" +
			"\r\n" +
			base64.StdEncoding.EncodeToString([]byte("Content of test.pdf")),
	}

	testMessage(t, msg, 0, want)
}

func TestAttachment(t *testing.T) {
	msg := NewMessage()
	msg.SetHeader("From", "from@example.com")
	msg.SetHeader("To", "to@example.com")
	msg.SetBody("text/plain", "Test")
	msg.Attach(CreateFile("test.pdf", []byte("Content")))

	want := message{
		from: "from@example.com",
		to:   []string{"to@example.com"},
		content: "From: from@example.com\r\n" +
			"To: to@example.com\r\n" +
			"Content-Type: multipart/mixed; boundary=_BOUNDARY_1_\r\n" +
			"\r\n" +
			"--_BOUNDARY_1_\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"Content-Transfer-Encoding: quoted-printable\r\n" +
			"\r\n" +
			"Test\r\n" +
			"--_BOUNDARY_1_\r\n" +
			"Content-Type: application/pdf; name=\"test.pdf\"\r\n" +
			"Content-Disposition: attachment; filename=\"test.pdf\"\r\n" +
			"Content-Transfer-Encoding: base64\r\n" +
			"\r\n" +
			base64.StdEncoding.EncodeToString([]byte("Content")) + "\r\n" +
			"--_BOUNDARY_1_--\r\n",
	}

	testMessage(t, msg, 1, want)
}

func TestAttachmentsOnly(t *testing.T) {
	msg := NewMessage()
	msg.SetHeader("From", "from@example.com")
	msg.SetHeader("To", "to@example.com")
	msg.Attach(CreateFile("test.pdf", []byte("Content 1")))
	msg.Attach(CreateFile("test.zip", []byte("Content 2")))

	want := message{
		from: "from@example.com",
		to:   []string{"to@example.com"},
		content: "From: from@example.com\r\n" +
			"To: to@example.com\r\n" +
			"Content-Type: multipart/mixed; boundary=_BOUNDARY_1_\r\n" +
			"\r\n" +
			"--_BOUNDARY_1_\r\n" +
			"Content-Type: application/pdf; name=\"test.pdf\"\r\n" +
			"Content-Disposition: attachment; filename=\"test.pdf\"\r\n" +
			"Content-Transfer-Encoding: base64\r\n" +
			"\r\n" +
			base64.StdEncoding.EncodeToString([]byte("Content 1")) + "\r\n" +
			"--_BOUNDARY_1_\r\n" +
			"Content-Type: application/zip; name=\"test.zip\"\r\n" +
			"Content-Disposition: attachment; filename=\"test.zip\"\r\n" +
			"Content-Transfer-Encoding: base64\r\n" +
			"\r\n" +
			base64.StdEncoding.EncodeToString([]byte("Content 2")) + "\r\n" +
			"--_BOUNDARY_1_--\r\n",
	}

	testMessage(t, msg, 1, want)
}

func TestAttachments(t *testing.T) {
	msg := NewMessage()
	msg.SetHeader("From", "from@example.com")
	msg.SetHeader("To", "to@example.com")
	msg.SetBody("text/plain", "Test")
	msg.Attach(CreateFile("test.pdf", []byte("Content 1")))
	msg.Attach(CreateFile("test.zip", []byte("Content 2")))

	want := message{
		from: "from@example.com",
		to:   []string{"to@example.com"},
		content: "From: from@example.com\r\n" +
			"To: to@example.com\r\n" +
			"Content-Type: multipart/mixed; boundary=_BOUNDARY_1_\r\n" +
			"\r\n" +
			"--_BOUNDARY_1_\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"Content-Transfer-Encoding: quoted-printable\r\n" +
			"\r\n" +
			"Test\r\n" +
			"--_BOUNDARY_1_\r\n" +
			"Content-Type: application/pdf; name=\"test.pdf\"\r\n" +
			"Content-Disposition: attachment; filename=\"test.pdf\"\r\n" +
			"Content-Transfer-Encoding: base64\r\n" +
			"\r\n" +
			base64.StdEncoding.EncodeToString([]byte("Content 1")) + "\r\n" +
			"--_BOUNDARY_1_\r\n" +
			"Content-Type: application/zip; name=\"test.zip\"\r\n" +
			"Content-Disposition: attachment; filename=\"test.zip\"\r\n" +
			"Content-Transfer-Encoding: base64\r\n" +
			"\r\n" +
			base64.StdEncoding.EncodeToString([]byte("Content 2")) + "\r\n" +
			"--_BOUNDARY_1_--\r\n",
	}

	testMessage(t, msg, 1, want)
}

func TestEmbedded(t *testing.T) {
	msg := NewMessage()
	msg.SetHeader("From", "from@example.com")
	msg.SetHeader("To", "to@example.com")
	f := CreateFile("image1.jpg", []byte("Content 1"))
	f.ContentID = "test-content-id"
	msg.Embed(f)
	msg.Embed(CreateFile("image2.jpg", []byte("Content 2")))
	msg.SetBody("text/plain", "Test")

	want := message{
		from: "from@example.com",
		to:   []string{"to@example.com"},
		content: "From: from@example.com\r\n" +
			"To: to@example.com\r\n" +
			"Content-Type: multipart/related; boundary=_BOUNDARY_1_\r\n" +
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
			base64.StdEncoding.EncodeToString([]byte("Content 1")) + "\r\n" +
			"--_BOUNDARY_1_\r\n" +
			"Content-Type: image/jpeg; name=\"image2.jpg\"\r\n" +
			"Content-Disposition: inline; filename=\"image2.jpg\"\r\n" +
			"Content-ID: <image2.jpg>\r\n" +
			"Content-Transfer-Encoding: base64\r\n" +
			"\r\n" +
			base64.StdEncoding.EncodeToString([]byte("Content 2")) + "\r\n" +
			"--_BOUNDARY_1_--\r\n",
	}

	testMessage(t, msg, 1, want)
}

func TestFullMessage(t *testing.T) {
	msg := NewMessage()
	msg.SetHeader("From", "from@example.com")
	msg.SetHeader("To", "to@example.com")
	msg.SetBody("text/plain", "¡Hola, señor!")
	msg.AddAlternative("text/html", "¡<b>Hola</b>, <i>señor</i>!</h1>")
	msg.Attach(CreateFile("test.pdf", []byte("Content 1")))
	msg.Embed(CreateFile("image.jpg", []byte("Content 2")))

	want := message{
		from: "from@example.com",
		to:   []string{"to@example.com"},
		content: "From: from@example.com\r\n" +
			"To: to@example.com\r\n" +
			"Content-Type: multipart/mixed; boundary=_BOUNDARY_1_\r\n" +
			"\r\n" +
			"--_BOUNDARY_1_\r\n" +
			"Content-Type: multipart/related; boundary=_BOUNDARY_2_\r\n" +
			"\r\n" +
			"--_BOUNDARY_2_\r\n" +
			"Content-Type: multipart/alternative; boundary=_BOUNDARY_3_\r\n" +
			"\r\n" +
			"--_BOUNDARY_3_\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"Content-Transfer-Encoding: quoted-printable\r\n" +
			"\r\n" +
			"=C2=A1Hola, se=C3=B1or!\r\n" +
			"--_BOUNDARY_3_\r\n" +
			"Content-Type: text/html; charset=UTF-8\r\n" +
			"Content-Transfer-Encoding: quoted-printable\r\n" +
			"\r\n" +
			"=C2=A1<b>Hola</b>, <i>se=C3=B1or</i>!</h1>\r\n" +
			"--_BOUNDARY_3_--\r\n" +
			"\r\n" +
			"--_BOUNDARY_2_\r\n" +
			"Content-Type: image/jpeg; name=\"image.jpg\"\r\n" +
			"Content-Disposition: inline; filename=\"image.jpg\"\r\n" +
			"Content-ID: <image.jpg>\r\n" +
			"Content-Transfer-Encoding: base64\r\n" +
			"\r\n" +
			base64.StdEncoding.EncodeToString([]byte("Content 2")) + "\r\n" +
			"--_BOUNDARY_2_--\r\n" +
			"\r\n" +
			"--_BOUNDARY_1_\r\n" +
			"Content-Type: application/pdf; name=\"test.pdf\"\r\n" +
			"Content-Disposition: attachment; filename=\"test.pdf\"\r\n" +
			"Content-Transfer-Encoding: base64\r\n" +
			"\r\n" +
			base64.StdEncoding.EncodeToString([]byte("Content 1")) + "\r\n" +
			"--_BOUNDARY_1_--\r\n",
	}

	testMessage(t, msg, 3, want)
}

func TestQpLineLength(t *testing.T) {
	msg := NewMessage()
	msg.SetHeader("From", "from@example.com")
	msg.SetHeader("To", "to@example.com")
	msg.SetBody("text/plain",
		strings.Repeat("0", 77)+"\r\n"+
			strings.Repeat("0", 76)+"à\r\n"+
			strings.Repeat("0", 75)+"à\r\n"+
			strings.Repeat("0", 74)+"à\r\n"+
			strings.Repeat("0", 73)+"à\r\n"+
			strings.Repeat("0", 76)+"\r\n"+
			strings.Repeat("0", 77)+"\n")

	want := message{
		from: "from@example.com",
		to:   []string{"to@example.com"},
		content: "From: from@example.com\r\n" +
			"To: to@example.com\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"Content-Transfer-Encoding: quoted-printable\r\n" +
			"\r\n" +
			strings.Repeat("0", 76) + "=\r\n0\r\n" +
			strings.Repeat("0", 76) + "=\r\n=C3=A0\r\n" +
			strings.Repeat("0", 75) + "=\r\n=C3=A0\r\n" +
			strings.Repeat("0", 74) + "=\r\n=C3=A0\r\n" +
			strings.Repeat("0", 73) + "=C3=\r\n=A0\r\n" +
			strings.Repeat("0", 76) + "\r\n" +
			strings.Repeat("0", 76) + "=\r\n0\n",
	}

	testMessage(t, msg, 0, want)
}

func TestBase64LineLength(t *testing.T) {
	msg := NewMessage(SetCharset("UTF-8"), SetEncoding(Base64))
	msg.SetHeader("From", "from@example.com")
	msg.SetHeader("To", "to@example.com")
	msg.SetBody("text/plain", strings.Repeat("0", 58))

	want := message{
		from: "from@example.com",
		to:   []string{"to@example.com"},
		content: "From: from@example.com\r\n" +
			"To: to@example.com\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"Content-Transfer-Encoding: base64\r\n" +
			"\r\n" +
			strings.Repeat("MDAw", 19) + "\r\nMA==",
	}

	testMessage(t, msg, 0, want)
}

func testMessage(t *testing.T, msg *Message, bCount int, emails ...message) {
	now = stubNow
	mailer := NewMailer("host", "username", "password", 587, SetSendMail(stubSendMail(t, bCount, emails...)))

	err := mailer.Send(msg)
	if err != nil {
		t.Error(err)
	}
}

func stubNow() time.Time {
	return time.Date(2014, 06, 25, 17, 46, 0, 0, time.UTC)
}

func stubSendMail(t *testing.T, bCount int, emails ...message) SendMailFunc {
	i := 0
	return func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		if i > len(emails) {
			t.Fatalf("Only %d mails should be sent", len(emails))
		}
		want := emails[i]

		if addr != "host:587" {
			t.Fatalf("Invalid address, got %q, want host:587", addr)
		}

		if from != want.from {
			t.Fatalf("Invalid from, got %q, want %q", from, want.from)
		}

		if len(to) != len(want.to) {
			t.Fatalf("Invalid recipient count, \ngot %d: %q\nwant %d: %q",
				len(to), to,
				len(want.to), want.to,
			)
		}
		for i := range want.to {
			if to[i] != want.to[i] {
				t.Fatalf("Invalid recipient, got %q, want %q",
					to[i], want.to[i],
				)
			}
		}

		got := string(msg)
		wantMsg := string("Mime-Version: 1.0\r\n" +
			"Date: Wed, 25 Jun 2014 17:46:00 +0000\r\n" +
			want.content)
		if bCount > 0 {
			boundaries := getBoundaries(t, bCount, got)
			for i, b := range boundaries {
				wantMsg = strings.Replace(wantMsg, "_BOUNDARY_"+strconv.Itoa(i+1)+"_", b, -1)
			}
		}
		i++

		compareBodies(t, got, wantMsg)

		return nil
	}
}

func compareBodies(t *testing.T, got, want string) {
	// We cannot do a simple comparison since the ordering of headers' fields
	// is random.
	gotLines := strings.Split(got, "\r\n")
	wantLines := strings.Split(want, "\r\n")

	// We only test for too many lines, missing lines are tested after
	if len(gotLines) > len(wantLines) {
		t.Fatalf("Message has too many lines, \ngot %d:\n%s\nwant %d:\n%s", len(gotLines), got, len(wantLines), want)
	}

	isInHeader := true
	headerStart := 0
	for i, line := range wantLines {
		if line == gotLines[i] {
			if line == "" {
				isInHeader = false
			} else if !isInHeader && len(line) > 2 && line[:2] == "--" {
				isInHeader = true
				headerStart = i + 1
			}
			continue
		}

		if !isInHeader {
			missingLine(t, line, got, want)
		}

		isMissing := true
		for j := headerStart; j < len(gotLines); j++ {
			if gotLines[j] == "" {
				break
			}
			if gotLines[j] == line {
				isMissing = false
				break
			}
		}
		if isMissing {
			missingLine(t, line, got, want)
		}
	}
}

func missingLine(t *testing.T, line, got, want string) {
	t.Fatalf("Missing line %q\ngot:\n%s\nwant:\n%s", line, got, want)
}

func getBoundaries(t *testing.T, count int, msg string) []string {
	if matches := boundaryRegExp.FindAllStringSubmatch(msg, count); matches != nil {
		boundaries := make([]string, count)
		for i, match := range matches {
			boundaries[i] = match[1]
		}
		return boundaries
	}

	t.Fatal("Boundary not found in body")
	return []string{""}
}

var boundaryRegExp = regexp.MustCompile("boundary=(\\w+)")

func BenchmarkFull(b *testing.B) {
	emptyFunc := func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		return nil
	}

	for n := 0; n < b.N; n++ {
		msg := NewMessage()
		msg.SetAddressHeader("From", "from@example.com", "Señor From")
		msg.SetHeaders(map[string][]string{
			"To":      {"to@example.com"},
			"Cc":      {"cc@example.com"},
			"Bcc":     {"bcc1@example.com", "bcc2@example.com"},
			"Subject": {"¡Hola, señor!"},
		})
		msg.SetBody("text/plain", "¡Hola, señor!")
		msg.AddAlternative("text/html", "<p>¡Hola, señor!</p>")
		msg.Attach(CreateFile("benchmark.txt", []byte("Benchmark")))
		msg.Embed(CreateFile("benchmark.jpg", []byte("Benchmark")))

		mailer := NewMailer("host", "username", "password", 587, SetSendMail(emptyFunc))
		if err := mailer.Send(msg); err != nil {
			panic(err)
		}
	}
}
