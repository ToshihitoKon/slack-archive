package archive

import (
	"context"
)

/* Example
formatter := NewFormatter()
textExporter := NewTextExporter()
fileExporter := NewFileExporter()

config := &Config{
	formatter:    formatter,
	textExporter: textExporter,
	fileExporter: fileExporter,
	...
}

Run(context.Background(), config)
*/

type CollectorInterface interface {
	Execute(context.Context) (Outputs, error)
}

type FormatterInterface interface {
	Format(Outputs, func(*LocalFile) string) []byte
}

type TextExporterInterface interface {
	Write(context.Context, []byte) error
}
type FileExporterInterface interface {
	WriteFiles(context.Context, []*LocalFile) error
	FormatFileName(*LocalFile) string
}
