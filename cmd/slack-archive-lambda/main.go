package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/events"
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
