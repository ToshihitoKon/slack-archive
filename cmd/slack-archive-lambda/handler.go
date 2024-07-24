package main

import (
	"context"
	"fmt"
	"log/slog"

	archive "github.com/ToshihitoKon/slack-archive"
)

func handler(ctx context.Context, req *archiveRequest) (string, error) {
	logger := slog.Default()
	archiveConf, err := makeConfig(ctx, req)
	if err != nil {
		logger.Error("Failed to make config", "error", err.Error())
		return "", fmt.Errorf("internal server error: config creation failed. %w", err)
	}

	if err := archive.Run(ctx, archiveConf); err != nil {
		logger.Error("Failed to archive run", "error", err.Error())
		return "", fmt.Errorf("internal server error: archive run failed. %w", err)
	}

	return "success", nil
}
