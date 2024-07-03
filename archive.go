package archive

import (
	"context"
)

func Run(ctx context.Context, config *Config) error {
	slackCollectorConfig := NewSlackCollectorConfig(config)
	collector := NewSlackCollector(config, slackCollectorConfig)
	defer collector.Clean()

	outputs, err := collector.Execute(ctx)
	if err != nil {
		return err
	}

	if err := config.FileExporter.WriteFiles(ctx, outputs.LocalFiles()); err != nil {
		return err
	}

	bytes := config.Formatter.Format(outputs, config.FileExporter.FormatFileName)

	if err := config.TextExporter.Write(ctx, bytes); err != nil {
		return err
	}

	return nil
}
