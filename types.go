package archive

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
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

type Output struct {
	ID        string    `json:"id,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
	Username  string    `json:"username,omitempty"`
	Text      string    `json:"text,omitempty"`

	Replies Outputs `json:"replies,omitempty"`
}

type Outputs []*Output

type CollectorInterface interface {
	Execute(context.Context) (Outputs, error)
}

type FormatterInterface interface {
	Format(Outputs) []byte
}

type ExporterInterface interface {
	Write(context.Context, []byte) error
}

// Implementations
type FormatterText struct{}

var _ FormatterInterface = (*FormatterText)(nil)

func (f *FormatterText) Format(outputs Outputs) []byte {
	sort.Slice(outputs, func(i, j int) bool { return outputs[i].Timestamp.Before(outputs[j].Timestamp) })

	texts := []string{}
	for _, output := range outputs {
		texts = append(texts, fmt.Sprintf(
			"%s %s> %s",
			output.Timestamp.Format("2006/01/02 15:04:05"),
			output.Username,
			output.Text,
		))
		sort.Slice(output.Replies, func(i, j int) bool { return output.Replies[i].Timestamp.Before(output.Replies[j].Timestamp) })
		for _, reply := range output.Replies {
			texts = append(texts, fmt.Sprintf(
				" | %s %s> %s",
				reply.Timestamp.Format("2006/01/02 15:04:05"),
				reply.Username,
				strings.ReplaceAll(reply.Text, "\n", "\n | "),
			))
		}
	}
	return []byte(strings.Join(texts, "\n"))
}

type ExporterFile struct {
	Writer io.Writer
}

var _ ExporterInterface = (*ExporterFile)(nil)

func (e *ExporterFile) Write(ctx context.Context, bytes []byte) error {
	if _, err := e.Writer.Write(bytes); err != nil {
		return err
	}
	return nil
}
