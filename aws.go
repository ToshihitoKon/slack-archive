package archive

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	sestypes "github.com/aws/aws-sdk-go-v2/service/ses/types"
)

type S3Exporter struct {
	s3Client        *s3.Client
	bucket          string
	archiveFilename string
	filesKeyPrefix  string

	logger *slog.Logger
}

var _ TextExporterInterface = (*S3Exporter)(nil)
var _ FileExporterInterface = (*S3Exporter)(nil)

func NewS3Exporter(ctx context.Context, config *Config) (*S3Exporter, error) {
	cfg, err := awsConfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	s3cli := s3.NewFromConfig(cfg)

	bucket := firstString([]string{
		config.S3Bucket,
		os.Getenv("SA_S3_EXPORTER_BUCKET"),
	})
	filesKeyPrefix := firstString([]string{
		config.S3FileKey,
		os.Getenv("SA_S3_EXPORTER_FILES_KEY_PREFIX"),
	})
	archiveFilename := firstString([]string{
		config.S3ArchiveKey,
		os.Getenv("SA_S3_EXPORTER_ARCHIVE_FILENAME"),
		path.Join(filesKeyPrefix, "log"),
	})
	if bucket == "" || archiveFilename == "" || filesKeyPrefix == "" {
		return nil, fmt.Errorf("SA_S3_EXPORTER_BUCKET, SA_S3_EXPORTER_ARCHIVE_FILENAME and SA_S3_EXPORTER_FILES_KEY_PREFIX is required")
	}

	return &S3Exporter{
		s3Client:        s3cli,
		bucket:          bucket,
		archiveFilename: archiveFilename,
		filesKeyPrefix:  filesKeyPrefix,
		logger:          config.Logger,
	}, nil
}

func (e *S3Exporter) Write(ctx context.Context, data []byte) error {
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

func (e *S3Exporter) WriteFiles(ctx context.Context, files []*LocalFile, fileNameFunc func(*LocalFile) string) error {
	for _, file := range files {
		var key string = path.Join(e.filesKeyPrefix, fileNameFunc(file))
		if err := e.putFileToS3(ctx, file.path, key); err != nil {
			return err
		}
	}

	return nil
}

func (e *S3Exporter) putFileToS3(ctx context.Context, srcPath, dstKey string) error {
	e.logger.Info("s3.PutObject", "source", srcPath, "destination", path.Join(e.bucket, dstKey))
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

type SESTextExporter struct {
	sesClient     *ses.Client
	configSetName string
	sourceArn     string
	maildata      *Mail

	to     []string
	logger *slog.Logger
}

var _ TextExporterInterface = (*SESTextExporter)(nil)

func NewSESTextExporter(ctx context.Context, config *Config) (*SESTextExporter, error) {
	cfg, err := awsConfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	cli := ses.NewFromConfig(cfg)

	to := config.SESTo
	configSetName := getEnv("SA_SES_EXPORTER_CONFIGURE_SET_NAME")
	sourceArn := getEnv("SA_SES_EXPORTER_SOURCE_ARN")
	from := getEnv("SA_SES_EXPORTER_FROM")
	if t := getEnv("SA_SES_EXPORTER_TO"); t != "" {
		to = append(to, t)
	}
	subject := getEnv("SA_SES_EXPORTER_SUBJECT")
	if configSetName == "" || sourceArn == "" || from == "" || len(to) == 0 || subject == "" {
		return nil, fmt.Errorf("SA_SES_EXPORTER_{CONFIGURE_SET_NAME, SOURCE_ARN, FROM, TO, SUBJECT} are required")
	}

	maildata := &Mail{
		From:     from,
		To:       to,
		Subject:  subject,
		Boundary: boundary(),
	}

	return &SESTextExporter{
		sesClient:     cli,
		configSetName: configSetName,
		sourceArn:     sourceArn,
		maildata:      maildata,
		logger:        config.Logger,
	}, nil
}

func (e *SESTextExporter) Write(ctx context.Context, data []byte) error {
	mailbody, err := toMIMEBody(data, e.maildata.Boundary)
	if err != nil {
		return err
	}
	e.maildata.Body = mailbody

	if err := e.sendMail(ctx, e.maildata); err != nil {
		return err
	}
	return nil
}

func (e *SESTextExporter) sendMail(ctx context.Context, maildata *Mail) error {
	header := maildata.headerString()

	rawMessage := append([]byte(header), maildata.Body...)
	e.logger.Info(string(rawMessage))

	msg := &sestypes.RawMessage{
		Data: rawMessage,
	}

	input := &ses.SendRawEmailInput{
		ConfigurationSetName: aws.String(e.configSetName),
		SourceArn:            aws.String(e.sourceArn),

		Source:       aws.String(maildata.From),
		Destinations: maildata.To,
		RawMessage:   msg,
	}

	if _, err := e.sesClient.SendRawEmail(ctx, input); err != nil {
		return err
	}

	return nil
}
