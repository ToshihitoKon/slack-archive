package archive

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
)

type NoneExporter struct{}

var _ TextExporterInterface = (*NoneExporter)(nil)
var _ FileExporterInterface = (*NoneExporter)(nil)

func (_ *NoneExporter) Write(_ context.Context, _ []byte) error { return nil }
func (_ *NoneExporter) WriteFiles(_ context.Context, _ []*LocalFile) error {
	return nil
}
func (_ *NoneExporter) FormatFileName(f *LocalFile) string {
	return ""
}

type LocalExporter struct {
	logFilePath string
	fileDirPath string

	logger *slog.Logger
}

var _ TextExporterInterface = (*LocalExporter)(nil)
var _ FileExporterInterface = (*LocalExporter)(nil)

func NewLocalExporter(logger *slog.Logger, logPath, fileDirPath string) *LocalExporter {
	if logPath == "" || fileDirPath == "" {
		panic("NewLocalExporter: logPath and fileDirPath are required")
	}

	return &LocalExporter{
		logFilePath: logPath,
		fileDirPath: fileDirPath,
		logger:      logger,
	}
}

func (e *LocalExporter) Write(ctx context.Context, data []byte) error {
	f, err := os.OpenFile(e.logFilePath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return err
	}
	return nil
}

func (e *LocalExporter) WriteFiles(ctx context.Context, files []*LocalFile) error {
	if _, err := os.ReadDir(e.fileDirPath); err != nil {
		if err := os.MkdirAll(e.fileDirPath, 0755); err != nil {
			return err
		}
	}
	e.logger.Info("WriteFile count", "num", len(files))
	for _, file := range files {
		filename := fmt.Sprintf("%s_%s", file.id, file.name)
		srcPath := file.path
		dstPath := path.Join(e.fileDirPath, filename)
		e.logger.Info("WriteFile copy", "source", srcPath, "destination", dstPath)
		if err := copy(srcPath, dstPath); err != nil {
			return err
		}
	}
	return nil
}

func (e *LocalExporter) FormatFileName(f *LocalFile) string {
	return fmt.Sprintf("%s_%s", f.id, f.name)
}

func copy(srcPath, dstPath string) error {
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
