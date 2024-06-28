package archive

import (
	"context"
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
