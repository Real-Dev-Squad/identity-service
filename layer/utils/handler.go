package utils

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type Deps struct {
	Client *firestore.Client
	Ctx    context.Context
}

type HandlerFunc func(*Deps, events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

func InitializeLambdaWithFirestore(functionName string, handlerFunc HandlerFunc) {
	initCtx := context.Background()
	logger := GetLogger()
	
	client, err := InitializeFirestoreClient(initCtx)
	if err != nil {
		logger.Error("Failed to initialize Firestore client", err, map[string]interface{}{
			"function": functionName,
		})
		lambda.Start(func(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
			return events.APIGatewayProxyResponse{
				StatusCode: 500,
				Body:       "Service initialization failed",
			}, nil
		})
		return
	}

	wrappedHandler := func(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		reqCtx, cancel := context.WithTimeout(context.Background(), 55*time.Second)
		defer cancel()
		
		deps := &Deps{
			Client: client,
			Ctx:    reqCtx,
		}
		
		return handlerFunc(deps, request)
	}

	lambda.Start(wrappedHandler)
}
