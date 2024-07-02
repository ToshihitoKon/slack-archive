package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	archive "github.com/ToshihitoKon/slack-archive"
)

func main() {
	ctx := context.Background()
	conf := newConfig()
	conf.parseFlags()

	formatter, err := conf.formatter()
	if err != nil {
		conf.logger.Error("config.formatter() failed", "error", err)
		os.Exit(1)
	}
	textExporter, err := conf.textExporter(ctx)
	if err != nil {
		conf.logger.Error("config.textExporter() failed", "error", err)
		os.Exit(1)
	}
	fileExporter, err := conf.fileExporter(ctx)
	if err != nil {
		conf.logger.Error("config.fileExporter() failed", "error", err)
		os.Exit(1)
	}

	archiveConf := &archive.Config{
		Since:  conf.since,
		Until:  conf.until,
		Logger: conf.logger,

		SlackToken:   archive.Getenv("SLACK_TOKEN"),
		SlackChannel: archive.Getenv("SLACK_CHANNEL"),

		Formatter:    formatter,
		TextExporter: textExporter,
		FileExporter: fileExporter,
	}
	if err := archive.Run(ctx, archiveConf); err != nil {
		conf.logger.Error("an error occurred", "function", "archive.Run", "error", err.Error())
		os.Exit(1)
	}
}

type config struct {
	since            time.Time
	until            time.Time
	formatterName    string
	textExporterName string
	fileExporterName string
	logger           *slog.Logger
}

func newConfig() *config {
	return &config{
		logger: slog.Default(),
	}
}

func (c *config) parseFlags() {
	since := flag.Int64("since", 0, "Archive message since")
	until := flag.Int64("until", 0, "Archive message until")
	duration := flag.String("duration", "", "Archive message duration")
	formatter := flag.String("formatter", "text", "Log format default: text")
	textExporter := flag.String("text-exporter", "local", "Exporter default: local")
	fileExporter := flag.String("file-exporter", "local", "Exporter default: local")
	flag.Parse()

	c.formatterName = *formatter
	c.textExporterName = *textExporter
	c.fileExporterName = *fileExporter

	if *duration != "" {
		if *since != 0 && *until != 0 {
			panic("You can't specify both since and until when specify duration")
		}
		dur, err := time.ParseDuration(*duration)
		if err != nil {
			panic(err)
		}

		switch {
		case *since != 0:
			c.since = time.Unix(*since, 0)
			c.until = c.since.Add(dur * 1)
		case *until != 0:
			c.until = time.Unix(*until, 0)
			c.since = c.until.Add(dur * -1)
		default:
			c.until = time.Now()
			c.since = c.until.Add(dur * -1)
		}
	} else {
		if *since != 0 {
			c.since = time.Unix(*since, 0)
		}
		if *until != 0 {
			c.until = time.Unix(*until, 0)
		}
	}
}

func (c *config) formatter() (archive.FormatterInterface, error) {
	var formatter archive.FormatterInterface
	switch c.formatterName {
	case "text":
		indent := " | "
		formatter = archive.NewTextFormatter(indent)
	default:
		return nil, fmt.Errorf("Formatter is not available. FormatterName: %s", c.formatterName)
	}
	return formatter, nil
}

func (c *config) textExporter(ctx context.Context) (archive.TextExporterInterface, error) {
	var textExporter archive.TextExporterInterface
	switch c.textExporterName {
	case "none":
		exp := &archive.NoneExporter{}
		textExporter = exp
	case "local":
		logPath := archive.Getenv("LOCAL_EXPORTER_LOGFILE")
		fileDir := archive.Getenv("LOCAL_EXPORTER_FILEDIR")
		exp := archive.NewLocalExporter(c.logger, logPath, fileDir)
		textExporter = exp
	case "s3":
		exp, err := archive.NewS3Exporter(ctx, c.logger,
			archive.Getenv("S3_EXPORTER_BUCKET"),
			archive.Getenv("S3_EXPORTER_ARCHIVE_FILENAME"),
			archive.Getenv("S3_EXPORTER_FILES_KEY_PREFIX"),
		)
		if err != nil {
			return nil, err
		}
		textExporter = exp
	case "ses":
		var (
			configSetName = archive.Getenv("SES_EXPORTER_CONFIG_SET_NAME")
			sourceArn     = archive.Getenv("SES_EXPORTER_SOURCE_ARN")
			from          = archive.Getenv("SES_EXPORTER_FROM")
			to            = []string{archive.Getenv("SES_EXPORTER_TO")}
			subject       = archive.Getenv("SES_EXPORTER_SUBJECT")
		)
		log.Println(configSetName, sourceArn, from, to, subject)
		exp, err := archive.NewSESTextExporter(ctx, c.logger, configSetName, sourceArn, from, to, subject)
		if err != nil {
			return nil, err
		}
		textExporter = exp
	default:
		return nil, fmt.Errorf("TextExporter %s is not available", c.textExporterName)
	}

	return textExporter, nil
}

func (c *config) fileExporter(ctx context.Context) (archive.FileExporterInterface, error) {
	var fileExporter archive.FileExporterInterface
	switch c.fileExporterName {
	case "none":
		exp := &archive.NoneExporter{}
		fileExporter = exp
	case "local":
		logPath := archive.Getenv("LOCAL_EXPORTER_LOGFILE")
		fileDir := archive.Getenv("LOCAL_EXPORTER_FILEDIR")
		exp := archive.NewLocalExporter(c.logger, logPath, fileDir)
		fileExporter = exp
	case "s3":
		exp, err := archive.NewS3Exporter(ctx, c.logger,
			archive.Getenv("S3_EXPORTER_BUCKET"),
			archive.Getenv("S3_EXPORTER_ARCHIVE_FILENAME"),
			archive.Getenv("S3_EXPORTER_FILES_KEY_PREFIX"),
		)
		if err != nil {
			return nil, err
		}
		fileExporter = exp
	default:
		return nil, fmt.Errorf("File exporter %s is not available", c.fileExporterName)
	}

	return fileExporter, nil
}
