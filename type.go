package archive

import (
	"log/slog"
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
