package archive

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

type Config struct {
	Since  time.Time
	Until  time.Time
	Logger *slog.Logger

	SlackToken   string
	SlackChannel string

	TextExporter TextExporterInterface
	FileExporter FileExporterInterface
	Formatter    FormatterInterface
}

type LocalFile struct {
	id        string
	path      string
	name      string
	timestamp time.Time
}

func (lf *LocalFile) detectContentType() (string, error) {
	f, err := os.Open(lf.path)
	if err != nil {
		return "", fmt.Errorf("error os.Open %w", err)
	}
	// DetectContentType read first 512 bytes
	// ref: https://pkg.go.dev/net/http#DetectContentType
	b := make([]byte, 512, 512)
	if _, err := f.Read(b); err != nil {
		return "", fmt.Errorf("error os.File.Read %w", err)
	}
	return http.DetectContentType(b), nil
}

type Output struct {
	ID        string    `json:"id,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
	Username  string    `json:"username,omitempty"`
	Text      string    `json:"text,omitempty"`

	Replies    Outputs `json:"replies,omitempty"`
	LocalFiles []*LocalFile
}

type Outputs []*Output

func (outputs Outputs) LocalFiles() []*LocalFile {
	res := []*LocalFile{}
	for _, output := range outputs {
		res = append(res, output.LocalFiles...)
		for _, reply := range output.Replies {
			res = append(res, reply.LocalFiles...)
		}
	}
	return res
}
