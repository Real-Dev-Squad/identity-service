package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/rishirishhh/identity-service/layer/utils"
)

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Use the utility function to generate the response body
	message := utils.GenerateHealthMessage()

	return events.APIGatewayProxyResponse{
		Body:       message,
		StatusCode: 200,
	}, nil
}
func main() {
	lambda.Start(handler)
}
