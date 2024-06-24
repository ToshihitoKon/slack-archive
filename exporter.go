package archive

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type ExporterLocal struct {
	writer      io.Writer
	fileSaveDir string
}

var _ TextExporterInterface = (*ExporterLocal)(nil)
var _ FileExporterInterface = (*ExporterLocal)(nil)

func NewExporterLocal() *ExporterLocal {
	saveDir := os.Getenv("SA_EXPORTER_LOCAL_DIR")
	if saveDir == "" {
		panic("ExporterLocal is require SA_EXPORTER_LOCAL_DIR")
	}

	return &ExporterLocal{
		writer:      os.Stdout,
		fileSaveDir: saveDir,
	}
}

func (e *ExporterLocal) Write(ctx context.Context, data []byte) error {
	if _, err := e.writer.Write(data); err != nil {
		return err
	}
	return nil
}

func (e *ExporterLocal) WriteFiles(ctx context.Context, files []*LocalFile, fileNameFunc func(*LocalFile) string) error {
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

type ExporterS3 struct {
	s3Client        *s3.Client
	bucket          string
	archiveFilename string
	filesKeyPrefix  string
}

var _ TextExporterInterface = (*ExporterS3)(nil)
var _ FileExporterInterface = (*ExporterS3)(nil)

func NewExporterS3(ctx context.Context) (*ExporterS3, error) {
	cfg, err := awsConfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	s3cli := s3.NewFromConfig(cfg)

	bucket := os.Getenv("SA_EXPORTER_S3_BUCKET")
	archiveFilename := os.Getenv("SA_EXPORTER_S3_ARCHIVE_FILENAME")
	filesKeyPrefix := os.Getenv("SA_EXPORTER_S3_FILES_KEY_PREFIX")
	if bucket == "" || archiveFilename == "" || filesKeyPrefix == "" {
		return nil, fmt.Errorf("SA_EXPORTER_S3_BUCKET, SA_EXPORTER_S3_ARCHIVE_FILENAME and SA_EXPORTER_S3_FILES_KEY_PREFIX is required")
	}

	return &ExporterS3{
		s3Client:        s3cli,
		bucket:          bucket,
		archiveFilename: archiveFilename,
		filesKeyPrefix:  filesKeyPrefix,
	}, nil
}

func (e *ExporterS3) Write(ctx context.Context, data []byte) error {
	params := &s3.PutObjectInput{
		Bucket: &e.bucket,
		Key:    &e.archiveFilename,
		Body:   bytes.NewReader(data),
	}
	if _, err := e.s3Client.PutObject(ctx, params); err != nil {
		return err
	}

	return nil
}

func (e *ExporterS3) WriteFiles(ctx context.Context, files []*LocalFile, fileNameFunc func(*LocalFile) string) error {
	for _, file := range files {
		var key string = path.Join(e.filesKeyPrefix, fileNameFunc(file))
		if err := e.putFileToS3(ctx, file.path, key); err != nil {
			return err
		}
	}

	return nil
}

func (e *ExporterS3) putFileToS3(ctx context.Context, srcPath, dstKey string) error {
	logger.Printf("s3.PutObject %s -> %s/%s\n", srcPath, e.bucket, dstKey)
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()

	params := &s3.PutObjectInput{
		Bucket: &e.bucket,
		Key:    &dstKey,
		Body:   f,
	}
	if _, err := e.s3Client.PutObject(ctx, params); err != nil {
		return err
	}

	return nil
}
