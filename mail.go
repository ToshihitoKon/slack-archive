package archive

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"math/rand"
	"mime/multipart"
	"net/textproto"
)

type Mail struct {
	From     string
	To       string
	Subject  string
	Body     []byte
	Boundary string
}

func (m *Mail) headerString() string {
	s := fmt.Sprintf(`From: %s
To: %s
Subject: %s
MIME-Version: 1.0
Content-Type: multipart/mixed; boundary=%s

`,
		m.From,
		m.To,
		m.Subject,
		m.Boundary,
	)
	return s
}

func toMIMEBody(data []byte, boundary string) ([]byte, error) {
	body := new(bytes.Buffer)
	bodyWriter := multipart.NewWriter(body)
	if err := bodyWriter.SetBoundary(boundary); err != nil {
		return nil, err
	}

	// multipart html part
	part, err := bodyWriter.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {"text/plain; charset=utf-8"},
		"Content-Transfer-Encoding": {"base64"},
	})
	if err != nil {
		bodyWriter.Close()
		return nil, err
	}

	enc := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
	base64.StdEncoding.Encode(enc, data)
	if _, err := part.Write(enc); err != nil {
		bodyWriter.Close()
		return nil, err
	}

	bodyWriter.Close()
	return body.Bytes(), nil
}

func boundary() string {
	length := 32
	runes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, length)
	for i := range b {
		b[i] = runes[rand.Intn(len(runes))]
	}
	return string(b)
}
