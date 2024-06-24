package archive

import (
	"context"
	"time"
)

/* Example
ctx := context.Background()
var outputs []*ArchiveOutput

collector := &ArchiveCollectorImplement{}
outputs, _ := collector.Execute(ctx)

exporter := &ArchiveExporterFileString{ Writer: os.Stdout }
_ = exporter.Write(ctx, output, os.Stdout)
*/

type CollectorInterface interface {
	Execute(context.Context) (Outputs, error)
}

type FormatterInterface interface {
	Format(Outputs) []byte
	WriteFileName(*LocalFile) string
}

type TextExporterInterface interface {
	Write(context.Context, []byte) error
}
type FileExporterInterface interface {
	WriteFiles(context.Context, []*LocalFile, func(*LocalFile) string) error
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
