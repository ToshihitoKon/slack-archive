package main

import (
	"fmt"
	"time"

	archive "github.com/ToshihitoKon/slack-archive"
)

type archiveRequest struct {
	SlackToken   string `json:"slack_token"`
	SlackChannel string `json:"slack_channel"`
	Since        string `json:"since"`
	Until        string `json:"until"`
	To           string `json:"to"`
	S3Bucket     string `json:"s_3_bucket,omitempty"`
	S3Key        string `json:"s_3_key,omitempty"`
}

func (r *archiveRequest) toConfig() (*archive.Config, error) {
	since, err := time.Parse(time.RFC3339, r.Since)
	if err != nil {
		return nil, fmt.Errorf("error time.Parse since: %w", err)
	}
	until, err := time.Parse(time.RFC3339, r.Until)
	if err != nil {
		return nil, fmt.Errorf("error time.Parse until: %w", err)
	}
	return &archive.Config{
		Since:     since,
		Until:     until,
		Exporter:  "ses",
		Formatter: "text",
	}, nil
}
