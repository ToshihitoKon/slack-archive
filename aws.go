package archive

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/url"
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

func NewS3Exporter(ctx context.Context, logger *slog.Logger, bucket, archiveFilename, filesKeyPrefix string) (*S3Exporter, error) {
	if bucket == "" || archiveFilename == "" || filesKeyPrefix == "" {
		return nil, fmt.Errorf("bucket, archiveFilename and filesKeyPrefix are required.")
	}

	cfg, err := awsConfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	s3cli := s3.NewFromConfig(cfg)

	return &S3Exporter{
		s3Client:        s3cli,
		bucket:          bucket,
		archiveFilename: archiveFilename,
		filesKeyPrefix:  filesKeyPrefix,
		logger:          logger,
	}, nil
}

func (e *S3Exporter) Write(ctx context.Context, data []byte) error {
	params := &s3.PutObjectInput{
		Bucket: aws.String(e.bucket),
		Key:    aws.String(e.archiveFilename),
		Body:   bytes.NewReader(data),
	}
	if _, err := e.s3Client.PutObject(ctx, params); err != nil {
		return err
	}
	e.logger.Info(fmt.Sprintf("S3Exporter: Write success. s3_object: s3://%s", path.Join(e.bucket, e.archiveFilename)))

	return nil
}

func (e *S3Exporter) WriteFiles(ctx context.Context, files []*LocalFile) error {
	for _, file := range files {
		ctype, err := file.detectContentType()
		if err != nil {
			e.logger.Error("an error occurred", "error", err.Error())
			continue
		}

		if err := e.putFileToS3(ctx, file.path, ctype, e.getS3Key(file)); err != nil {
			return err
		}
	}
	e.logger.Info(fmt.Sprintf("S3Exporter: WriteFiles success. files_num: %d", len(files)))
	return nil
}

func (e *S3Exporter) FormatFileName(f *LocalFile) string {
	return e.getS3Url(f).String()
}

func (e *S3Exporter) getS3Key(f *LocalFile) string {
	return path.Join(e.filesKeyPrefix, fmt.Sprintf("%s_%s", f.id, f.name))
}

func (e *S3Exporter) getS3Url(f *LocalFile) *url.URL {
	return &url.URL{
		Scheme: "https",
		Host:   "s3.ap-northeast-1.amazonaws.com",
		Path:   path.Join("/", e.bucket, e.getS3Key(f)),
	}
}

func (e *S3Exporter) putFileToS3(ctx context.Context, srcPath, contentType, dstKey string) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()

	params := &s3.PutObjectInput{
		Bucket: aws.String(e.bucket),
		Key:    aws.String(dstKey),
		Body:   f,
	}
	if contentType != "" {
		params.ContentType = aws.String(contentType)
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

	logger *slog.Logger
}

var _ TextExporterInterface = (*SESTextExporter)(nil)

func NewSESTextExporter(ctx context.Context, logger *slog.Logger,
	sesConfigSetName string, sesSourceArn string,
	from string, to []string, subject string,
) (*SESTextExporter, error) {
	if sesConfigSetName == "" || sesSourceArn == "" ||
		from == "" || len(to) == 0 || subject == "" {
		return nil, fmt.Errorf("sesConfigSetName, sesSourceArn, from, to and subject are required.")
	}
	cfg, err := awsConfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	cli := ses.NewFromConfig(cfg)

	maildata := &Mail{
		From:     from,
		To:       to,
		Subject:  subject,
		Boundary: boundary(),
	}

	return &SESTextExporter{
		sesClient:     cli,
		configSetName: sesConfigSetName,
		sourceArn:     sesSourceArn,
		maildata:      maildata,
		logger:        logger,
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
	e.logger.Info("SESTextExporter: Write success.")
	return nil
}

func (e *SESTextExporter) sendMail(ctx context.Context, maildata *Mail) error {
	header := maildata.headerString()

	rawMessage := append([]byte(header), maildata.Body...)
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
