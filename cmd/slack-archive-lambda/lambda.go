package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log/slog"

	"github.com/aws/aws-lambda-go/events"
)

// request and response event formats follow the same schema as the Amazon API Gateway payload format version 2.0.
// ref: https://docs.aws.amazon.com/lambda/latest/dg/urls-invocation.html#urls-payloads
func lambdaHandler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	logger := slog.Default()

	body := []byte(request.Body)
	if request.IsBase64Encoded {
		b, err := base64.StdEncoding.DecodeString(request.Body)
		if err != nil {
			logger.Error("failed to base64 decode request.Body", "error", err.Error(), "body", request.Body)
			return errToAPIGatewayResponse(err, 400), err
		}
		body = b
	}

	req := &archiveRequest{}
	if err := json.Unmarshal(body, req); err != nil {
		logger.Error("failed to unmarshal request.Body", "error", err.Error(), "body", string(body))
		return errToAPIGatewayResponse(err, 400), err
	}

	response, err := handler(ctx, req)
	if err != nil {
		logger.Error("an error occurred", "error", err.Error(), "function", "handler")
		return errToAPIGatewayResponse(err, 500), err
	}

	return events.APIGatewayProxyResponse{
		Body:       response,
		StatusCode: 200,
	}, nil
}

func errToAPIGatewayResponse(err error, code int) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		Body:       err.Error(),
		StatusCode: code,
	}
}
