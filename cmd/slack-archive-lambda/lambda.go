package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/events"
)

// Lambda predefined runtime environment variables
// ref: https://docs.aws.amazon.com/ja_jp/lambda/latest/dg/configuration-envvars.html#configuration-envvars-runtime
var (
	lambdaRegion  = os.Getenv("AWS_REGION")
	lambdaName    = os.Getenv("AWS_LAMBDA_FUNCTION_NAME")
	lambdaVersion = os.Getenv("AWS_LAMBDA_FUNCTION_VERSION")
)

// request and response event formats follow the same schema as the Amazon API Gateway payload format version 2.0.
// ref: https://docs.aws.amazon.com/lambda/latest/dg/urls-invocation.html#urls-payloads
func lambdaHandler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	logger := slog.Default()
	req := &archiveRequest{}
	if err := json.Unmarshal([]byte(request.Body), req); err != nil {
		logger.Error("failed to unmarshal request.Body", "error", err.Error(), "body", request.Body)
		return errToAPIGatewayResponse(err), err
	}

	response, err := handler(ctx, req)
	if err != nil {
		logger.Error("an error occurred", "error", err.Error(), "function", "handler")
		return errToAPIGatewayResponse(err), err
	}

	return events.APIGatewayProxyResponse{
		Body:       response,
		StatusCode: 200,
	}, nil
}

func errToAPIGatewayResponse(err error) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		Body:       err.Error(),
		StatusCode: 500,
	}
}
