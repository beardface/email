package email

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/smtp"
	"path/filepath"
	"strings"
	"time"
)

type Attachment struct {
	Filename string
	Data     []byte
	Inline   bool
	Cid      string
}

type Message struct {
	From            string
	To              []string
	Cc              []string
	Bcc             []string
	Subject         string
	Body            string
	BodyContentType string
	Attachments     map[string]*Attachment
}

func (m *Message) attach(file string, inline bool) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	_, filename := filepath.Split(file)

	m.Attachments[filename] = &Attachment{
		Filename: filename,
		Data:     data,
		Inline:   inline,
	}

	return nil
}

func (m *Message) AttachFromBytes(filename string, data []byte, inline bool, cid string) error {

	m.Attachments[filename] = &Attachment{
		Filename: filename,
		Data:     data,
		Inline:   inline,
		Cid:      cid,
	}

	return nil
}

func (m *Message) Attach(file string) error {
	return m.attach(file, false)
}

func (m *Message) Inline(file string) error {
	return m.attach(file, true)
}

func newMessage(subject string, body string, bodyContentType string) *Message {
	m := &Message{Subject: subject, Body: body, BodyContentType: bodyContentType}

	m.Attachments = make(map[string]*Attachment)

	return m
}

// NewMessage returns a new Message that can compose an email with attachments
func NewMessage(subject string, body string) *Message {
	return newMessage(subject, body, "text/plain")
}

// NewMessage returns a new Message that can compose an HTML email with attachments
func NewHTMLMessage(subject string, body string) *Message {
	return newMessage(subject, body, "text/html")
}

// ToList returns all the recipients of the email
func (m *Message) Tolist() []string {
	tolist := m.To

	for _, cc := range m.Cc {
		tolist = append(tolist, cc)
	}

	for _, bcc := range m.Bcc {
		tolist = append(tolist, bcc)
	}

	return tolist
}

// Bytes returns the mail data
func (m *Message) Bytes() []byte {
	buf := bytes.NewBuffer(nil)

	buf.WriteString("From: " + m.From + "\r\n")

	t := time.Now()
	buf.WriteString("Date: " + t.Format(time.RFC822) + "\r\n")

	buf.WriteString("To: " + strings.Join(m.To, ",") + "\r\n")
	if len(m.Cc) > 0 {
		buf.WriteString("Cc: " + strings.Join(m.Cc, ",") + "\r\n")
	}

	buf.WriteString("Subject: " + m.Subject + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")

	boundary := "f46d043c813270fc6b04c2d223da"

	if len(m.Attachments) > 0 {
		buf.WriteString("Content-Type: multipart/related; boundary=" + boundary + "\r\n\r\n")

		buf.WriteString("--" + boundary + "\r\n")
		buf.WriteString(fmt.Sprintf("Content-Type: %s; charset=utf-8\r\n", m.BodyContentType))
		buf.WriteString(m.Body)

		for _, attachment := range m.Attachments {
			buf.WriteString("\r\n\r\n--" + boundary + "\r\n")

			if attachment.Inline {
				buf.WriteString("Content-Type: message/rfc822\r\n")
				buf.WriteString("Content-Disposition: inline; filename=\"" + attachment.Filename + "\"\r\n\r\n")

				buf.Write(attachment.Data)
			} else {
				buf.WriteString("Content-Type: application/octet-stream\r\n")
				if attachment.Cid != "" {
					buf.WriteString(fmt.Sprintf("Content-ID: <%s>\r\n", attachment.Cid))
				}
				buf.WriteString("Content-Disposition: attachment; filename=\"" + attachment.Filename + "\"\r\n")
				buf.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")

				b := make([]byte, base64.StdEncoding.EncodedLen(len(attachment.Data)))
				base64.StdEncoding.Encode(b, attachment.Data)
				buf.Write(b)
			}

			buf.WriteString("\r\n--" + boundary)
		}

		buf.WriteString("--")
	} else {
		buf.WriteString(fmt.Sprintf("Content-Type: %s; charset=utf-8\r\n\r\n", m.BodyContentType))
		buf.WriteString(m.Body)
	}

	return buf.Bytes()
}

func Send(addr string, auth smtp.Auth, m *Message) error {
	message := m.Bytes()
	log.Printf("Message size (%d)\r\n", len(message))
	return smtp.SendMail(addr, auth, m.From, m.Tolist(), message)
}
