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
	// slog.Info("lambdaHandler handled request", "body", request.Body)
	req := &archiveRequest{}
	if err := json.Unmarshal([]byte(request.Body), req); err != nil {
		return errToAPIGatewayResponse(err), err
	}
	slog.Info("success archiveRequest parse.", slog.Any("body", req))

	response, err := handler(ctx, req)
	if err != nil {
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
