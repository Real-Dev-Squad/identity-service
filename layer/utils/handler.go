package utils

import (
	"context"

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
	ctx := context.Background()
	logger := GetLogger()
	
	client, err := InitializeFirestoreClient(ctx)
	if err != nil {
		logger.Error("Failed to initialize Firestore client", err, map[string]interface{}{
			"function": functionName,
		})
		return
	}

	deps := &Deps{
		Client: client,
		Ctx:    ctx,
	}

	wrappedHandler := func(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		return handlerFunc(deps, request)
	}

	lambda.Start(wrappedHandler)
}
