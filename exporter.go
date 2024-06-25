package archive

import (
	"context"
	"io"
	"os"
	"path"
)

type NoneExporter struct{}

var _ TextExporterInterface = (*NoneExporter)(nil)
var _ FileExporterInterface = (*NoneExporter)(nil)

func (_ *NoneExporter) Write(_ context.Context, _ []byte) error { return nil }
func (_ *NoneExporter) WriteFiles(_ context.Context, _ []*LocalFile, _ func(*LocalFile) string) error {
	return nil
}

type LocalExporter struct {
	writer      io.Writer
	fileSaveDir string
}

var _ TextExporterInterface = (*LocalExporter)(nil)
var _ FileExporterInterface = (*LocalExporter)(nil)

func NewLocalExporter() *LocalExporter {
	saveDir := os.Getenv("SA_EXPORTER_LOCAL_DIR")
	if saveDir == "" {
		panic("LocalExporter is require SA_EXPORTER_LOCAL_DIR")
	}

	return &LocalExporter{
		writer:      os.Stdout,
		fileSaveDir: saveDir,
	}
}

func (e *LocalExporter) Write(ctx context.Context, data []byte) error {
	if _, err := e.writer.Write(data); err != nil {
		return err
	}
	return nil
}

func (e *LocalExporter) WriteFiles(ctx context.Context, files []*LocalFile, fileNameFunc func(*LocalFile) string) error {
	if _, err := os.ReadDir(e.fileSaveDir); err != nil {
		if err := os.MkdirAll(e.fileSaveDir, 0755); err != nil {
			return err
		}
	}
	logger.Printf("WriteFile files num %d\n", len(files))
	for _, file := range files {
		srcPath := file.path
		dstPath := path.Join(e.fileSaveDir, fileNameFunc(file))
		if err := copy(srcPath, dstPath); err != nil {
			return err
		}
	}
	return nil
}

func copy(srcPath, dstPath string) error {
	logger.Printf("WriteFile %s -> %s\n", srcPath, dstPath)
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	return nil
}
