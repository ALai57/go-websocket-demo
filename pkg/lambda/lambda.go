package main

import (
	"context"
	"go_websocket_demo/pkg/websocket_api"
	"log/slog"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func DecodeLambdaRequest(request events.APIGatewayWebsocketProxyRequest) (websocket_api.Command, error) {
	// Switch statement check for action
	return nil, nil
}

func main() {
	svc := websocket_api.NewService()

	slog.Info("Starting up Lambda")

	lambda.Start(func(ctx context.Context, request events.APIGatewayWebsocketProxyRequest) (*events.APIGatewayProxyResponse, error) {
		slog.Info("Receiving request")

		r, err := DecodeLambdaRequest(request)
		if err != nil {
			panic("Error!")
		}

		if err := r.Exec(svc); err != nil {
			panic("Error!")
		}

		return &events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
		}, nil
	})

}
