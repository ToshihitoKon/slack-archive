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
	WriteFileName(*TempFile) string
}

type ExporterInterface interface {
	Write(context.Context, []byte) error
	WriteFiles(context.Context, []*TempFile) error
}

type TempFile struct {
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

	Replies   Outputs `json:"replies,omitempty"`
	TempFiles []*TempFile
}

type Outputs []*Output

func (outputs Outputs) TempFiles() []*TempFile {
	res := []*TempFile{}
	for _, output := range outputs {
		res = append(res, output.TempFiles...)
		for _, reply := range output.Replies {
			res = append(res, reply.TempFiles...)
		}
	}
	return res
}
