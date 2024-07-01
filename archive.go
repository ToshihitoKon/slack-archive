package archive

import (
	"context"
	"fmt"
	"os"
)

func Run(ctx context.Context, config *Config) error {
	defer func() {
		config.Logger.Info(fmt.Sprintf("Remove %s", config.LocalFileDir))
		os.RemoveAll(config.LocalFileDir)
	}()

	slackCollectorConfig := NewSlackCollectorConfig(config)
	collector := NewSlackCollector(config.Logger, slackCollectorConfig, config)

	outputs, err := collector.Execute(ctx)
	if err != nil {
		return err
	}

	bytes := config.Formatter.Format(outputs)
	if err := config.TextExporter.Write(ctx, bytes); err != nil {
		return err
	}
	if err := config.FileExporter.WriteFiles(ctx, outputs.LocalFiles(), config.Formatter.WriteFileName); err != nil {
		return err
	}

	return nil
}
