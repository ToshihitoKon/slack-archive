package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	archive "github.com/ToshihitoKon/slack-archive"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// Lambda predefined runtime environment variables
// ref: https://docs.aws.amazon.com/ja_jp/lambda/latest/dg/configuration-envvars.html#configuration-envvars-runtime
var (
	lambdaRegion  = os.Getenv("AWS_REGION")
	lambdaName    = os.Getenv("AWS_LAMBDA_FUNCTION_NAME")
	lambdaVersion = os.Getenv("AWS_LAMBDA_FUNCTION_VERSION")
)

func main() {
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
		slog.Info("Start on lambda runtime", "region", lambdaRegion, "name", lambdaName, "version", lambdaVersion)
		lambda.Start(lambdaHandler)
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))
		slog.Info("Start on local")

		req, err := readLocalRequestJson(os.Getenv("SA_LAMBDA_REQUEST_JSON_PATH"))
		if err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}

		res, err := lambdaHandler(context.Background(), req)
		if err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}

		slog.Info("ok", slog.Any("response", res))
	}
}

func readLocalRequestJson(filePath string) (events.APIGatewayProxyRequest, error) {
	req := events.APIGatewayProxyRequest{}

	f, err := os.Open(filePath)
	if err != nil {
		return req, err
	}
	defer f.Close()

	requestBytes, err := io.ReadAll(f)
	if err != nil {
		return req, err
	}

	if err := json.Unmarshal(requestBytes, &req); err != nil {
		return req, err
	}

	return req, nil
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

	replyIndent := "    | "
	formatter := archive.NewTextFormatter(replyIndent)

	var (
		configSetName = archive.Getenv("SES_EXPORTER_CONFIG_SET_NAME")
		sourceArn     = archive.Getenv("SES_EXPORTER_SOURCE_ARN")
		from          = "husky@platen.sre-ws.donuts.ne.jp"
		to            = req.To
		subject       = req.Subject
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
