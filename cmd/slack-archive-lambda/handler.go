package main

import (
	"context"

	archive "github.com/ToshihitoKon/slack-archive"
)

func handler(ctx context.Context, req *archiveRequest) (string, error) {
	archiveConf, err := makeConfig(ctx, req)
	if err != nil {
		return "internal server error", err
	}

	if err := archive.Run(ctx, archiveConf); err != nil {
		return "internal server error", err
	}

	return "success", nil
}
