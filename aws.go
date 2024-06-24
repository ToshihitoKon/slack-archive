package archive

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	sestypes "github.com/aws/aws-sdk-go-v2/service/ses/types"
)

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

type TextExporterSES struct {
	sesClient     *ses.Client
	configSetName string
	sourceArn     string
	maildata      *Mail
}

var _ TextExporterInterface = (*TextExporterSES)(nil)

func NewTextExporterSES(ctx context.Context) (*TextExporterSES, error) {
	cfg, err := awsConfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	cli := ses.NewFromConfig(cfg)

	configSetName := os.Getenv("SA_EXPORTER_SES_CONFIGURE_SET_NAME")
	sourceArn := os.Getenv("SA_EXPORTER_SES_SOURCE_ARN")
	from := os.Getenv("SA_EXPORTER_SES_FROM")
	to := os.Getenv("SA_EXPORTER_SES_TO")
	subject := os.Getenv("SA_EXPORTER_SES_SUBJECT")
	if configSetName == "" || sourceArn == "" || from == "" || to == "" || subject == "" {
		return nil, fmt.Errorf("SA_EXPORTER_SES_{CONFIGURE_SET_NAME, SOURCE_ARN, FROM, TO, SUBJECT} are required")
	}

	maildata := &Mail{
		From:     from,
		To:       to,
		Subject:  subject,
		Boundary: boundary(),
	}

	return &TextExporterSES{
		sesClient:     cli,
		configSetName: configSetName,
		sourceArn:     sourceArn,
		maildata:      maildata,
	}, nil
}

func (e *TextExporterSES) Write(ctx context.Context, data []byte) error {
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

func (e *TextExporterSES) sendMail(ctx context.Context, maildata *Mail) error {
	header := maildata.headerString()

	rawMessage := append([]byte(header), maildata.Body...)
	logger.Println(string(rawMessage))

	msg := &sestypes.RawMessage{
		Data: rawMessage,
	}

	input := &ses.SendRawEmailInput{
		ConfigurationSetName: aws.String(e.configSetName),
		SourceArn:            aws.String(e.sourceArn),

		Source:       aws.String(maildata.From),
		Destinations: []string{maildata.To},
		RawMessage:   msg,
	}

	if _, err := e.sesClient.SendRawEmail(ctx, input); err != nil {
		return err
	}

	return nil
}
