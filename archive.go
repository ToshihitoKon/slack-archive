package archive

import (
	"context"
	"fmt"
	"os"
)

func Run(conf *Config) error {
	ctx := context.Background()
	return run(ctx, conf)
}

func run(ctx context.Context, config *Config) error {
	defer func() {
		config.Logger.Info(fmt.Sprintf("Remove %s", config.LocalFileDir))
		os.RemoveAll(config.LocalFileDir)
	}()

	slackCollectorConfig := NewSlackCollectorConfig(config)
	collector := NewSlackCollector(config.Logger, slackCollectorConfig, config)

	var formatter FormatterInterface
	switch config.Formatter {
	case "text":
		formatter = NewTextFormatter()
	default:
		return fmt.Errorf("Format %s is not available", config.Exporter)
	}

	var textExporter TextExporterInterface
	var fileExporter FileExporterInterface
	switch config.Exporter {
	case "none":
		exp := &NoneExporter{}
		textExporter = exp
		fileExporter = exp
	case "local":
		exp := NewLocalExporter(config.Logger)
		textExporter = exp
		fileExporter = exp
	case "s3":
		exp, err := NewS3Exporter(ctx, config)
		if err != nil {
			return err
		}
		textExporter = exp
		fileExporter = exp
	case "ses":
		tExp, err := NewSESTextExporter(ctx, config)
		if err != nil {
			return err
		}
		textExporter = tExp

		fExp, err := NewS3Exporter(ctx, config)
		if err != nil {
			return err
		}
		fileExporter = fExp
	default:
		return fmt.Errorf("Exporter %s is not available", config.Exporter)
	}

	outputs, err := collector.Execute(ctx)
	if err != nil {
		return err
	}

	bytes := formatter.Format(outputs)

	if err := textExporter.Write(ctx, bytes); err != nil {
		return err
	}
	if err := fileExporter.WriteFiles(ctx, outputs.LocalFiles(), formatter.WriteFileName); err != nil {
		return err
	}

	return nil
}
