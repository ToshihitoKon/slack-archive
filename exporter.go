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
	logFilePath string
	fileDirPath string
}

var _ TextExporterInterface = (*LocalExporter)(nil)
var _ FileExporterInterface = (*LocalExporter)(nil)

func NewLocalExporter() *LocalExporter {
	logPath := getEnv("SA_LOCAL_EXPORTER_LOGFILE")
	fileDirPath := getEnv("SA_LOCAL_EXPORTER_FILEDIR")
	if fileDirPath == "" || logPath == "" {
		panic("SA_LOCAL_EXPORTER_{LOGFILE, FILEDIR} are required")
	}

	return &LocalExporter{
		logFilePath: logPath,
		fileDirPath: fileDirPath,
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

func (e *LocalExporter) WriteFiles(ctx context.Context, files []*LocalFile, fileNameFunc func(*LocalFile) string) error {
	if _, err := os.ReadDir(e.fileDirPath); err != nil {
		if err := os.MkdirAll(e.fileDirPath, 0755); err != nil {
			return err
		}
	}
	logger.Printf("WriteFile files num %d\n", len(files))
	for _, file := range files {
		srcPath := file.path
		dstPath := path.Join(e.fileDirPath, fileNameFunc(file))
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
