package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	archive "github.com/ToshihitoKon/slack-archive"
)

type archiveRequest struct {
	SlackToken   string   `json:"slack_token"`
	SlackChannel string   `json:"slack_channel"`
	Since        string   `json:"since"`
	Until        string   `json:"until"`
	To           []string `json:"to"`
	S3Bucket     string   `json:"s3_bucket,omitempty"`
	S3Key        string   `json:"s3_key,omitempty"`
}

func makeConfig(ctx context.Context, req *archiveRequest) (*archive.Config, error) {
	logger := slog.Default()

	since, err := time.Parse(time.RFC3339, req.Since)
	if err != nil {
		return nil, fmt.Errorf("error time.Parse since: %w", err)
	}
	until, err := time.Parse(time.RFC3339, req.Until)
	if err != nil {
		return nil, fmt.Errorf("error time.Parse until: %w", err)
	}

	indent := " | "
	formatter := archive.NewTextFormatter(indent)

	var (
		configSetName = archive.Getenv("SES_EXPORTER_CONFIG_SET_NAME")
		sourceArn     = archive.Getenv("SES_EXPORTER_SOURCE_ARN")
		from          = "noreply@platen.sre-ws.donuts.ne.jp"
		to            = req.To
		subject       = "Siberian husky" // TODO: 設定できるようにする
	)
	textExporter, err := archive.NewSESTextExporter(ctx, logger, configSetName, sourceArn, from, to, subject)
	if err != nil {
		return nil, err
	}

	fileExporter, err := archive.NewS3Exporter(ctx, logger, req.S3Bucket, "dummy", req.S3Key)

	if err != nil {
		return nil, err
	}

	conf := &archive.Config{
		Since:  since,
		Until:  until,
		Logger: slog.Default(),

		SlackToken:   req.SlackToken,
		SlackChannel: req.SlackChannel,

		Formatter:    formatter,
		TextExporter: textExporter,
		FileExporter: fileExporter,
	}
	return conf, nil
}
