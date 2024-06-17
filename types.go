package archive

import (
	"context"
	"fmt"
	"io"
	"sort"
	"time"
)

/* Example
ctx := context.Background()
var outputs []*ArchiveOutput

collector := &ArchiveCollectorImplement{}
exporter := &ArchiveExporterText{}

outputs, _ := collector.Execute(ctx)
_ = exporter.Write(ctx, output, os.Stdout)
*/

type ArchiveOutput struct {
	Timestamp time.Time `json:"timestamp,omitempty"`
	Username  string    `json:"username,omitempty"`
	Text      string    `json:"text,omitempty"`
}

type ArchiveCollectorInterface interface {
	Execute(context.Context) ([]*ArchiveOutput, error)
}

type ArchiveExporterInterface interface {
	Write(context.Context, []*ArchiveOutput, io.Writer) error
}

// Text Exporter
type ArchiveExporterString struct{}

var _ ArchiveExporterInterface = (*ArchiveExporterString)(nil)

func (e *ArchiveExporterString) Write(ctx context.Context, outputs []*ArchiveOutput, w io.Writer) error {
	sort.Slice(outputs, func(i, j int) bool { return outputs[i].Timestamp.Before(outputs[j].Timestamp) })
	for _, output := range outputs {
		if _, err := fmt.Fprintf(w, "%s: %s\n%s\n---\n",
			output.Timestamp.Format("2006/01/02 15:04:05"),
			output.Username,
			output.Text,
		); err != nil {
			return err
		}
	}
	return nil
}

// Json Exporter
type ArchiveExporterJson struct{}

var _ ArchiveExporterInterface = (*ArchiveExporterJson)(nil)

func (e *ArchiveExporterJson) Write(ctx context.Context, outputs []*ArchiveOutput, w io.Writer) error {
	// TODO: impl
	fmt.Println("Json.Write not impremented")
	return nil
}
