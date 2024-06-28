package main

import (
	"context"
	"log/slog"
	"os"

	archive "github.com/ToshihitoKon/slack-archive"

	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
		slog.Info("Start on lambda runtime")
		lambda.Start(lambdaHandler)
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))
		slog.Info("Start on local")
		// TODO: ファイルからよしなにする
		req := &archiveRequest{
			SlackToken:   os.Getenv("SA_SLACK_TOKEN"),
			SlackChannel: os.Getenv("SA_SLACK_CHANNEL"),
			Since:        "2024-06-24T00:00:00+09:00", // RFC3339 "2006-01-02T15:04:05Z07:00"
			Until:        "2024-06-24T23:59:59+09:00",
			To:           os.Getenv("SA_SES_EXPORTER_TO"),
			S3Bucket:     os.Getenv("SA_S3_EXPORTER_BUCKET"),
			S3Key:        os.Getenv("SA_S3_EXPORTER_FILES_KEY_PREFIX"),
		}
		response, err := handler(context.Background(), req)
		if err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}
		slog.Info("ok", "response", response)
	}
}

func handler(ctx context.Context, req *archiveRequest) (string, error) {
	_ = ctx
	archiveConf, err := req.toConfig()
	if err != nil {
		return "internal server error", err
	}

	if err := archive.Run(archiveConf); err != nil {
		return "internal server error", err
	}

	return "success", nil
}
