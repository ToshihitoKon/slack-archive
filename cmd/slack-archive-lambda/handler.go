package main

import (
	"context"

	archive "github.com/ToshihitoKon/slack-archive"
)

func handler(ctx context.Context, req *archiveRequest) (string, error) {
	logger := slog.Default()
	archiveConf, err := makeConfig(ctx, req)
	if err != nil {
		logger.Error("Failed to make config", "error", err.Error())
		return "internal server error: config creation failed", err
	}

	if err := archive.Run(ctx, archiveConf); err != nil {
		logger.Error("Failed to archive run", "error", err.Error())
		return "internal server error: archive run failed", err
	}

	return "success", nil
}
