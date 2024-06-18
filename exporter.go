package archive

import (
	"context"
	"io"
	"os"
	"path"
)

type ExporterFile struct {
	writer      io.Writer
	fileSaveDir string
}

func NewExporterFile() *ExporterFile {
	saveDir := os.Getenv("SA_EXPORTER_FILE_DIR")
	if saveDir == "" {
		panic("ExporterFile is require SA_EXPORTER_FILE_DIR")
	}

	return &ExporterFile{
		writer:      os.Stdout,
		fileSaveDir: saveDir,
	}
}

func (e *ExporterFile) Write(ctx context.Context, bytes []byte) error {
	if _, err := e.writer.Write(bytes); err != nil {
		return err
	}
	return nil
}

func (e *ExporterFile) WriteFiles(ctx context.Context, files []*TempFile, fileNameFunc func(*TempFile) string) error {
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
