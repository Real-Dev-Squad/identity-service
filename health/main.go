package main

import (
	// "errors"
	"fmt"
	"io/ioutil"

	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var (
	memberurl = "https://1ngy2alfy3.execute-api.us-east-2.amazonaws.com/Prod/health"
)

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	resp, err := http.Get(memberurl)

	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	r, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}
	fmt.Printf("%v", string(r))
	//save the response/err in firestore
	return events.APIGatewayProxyResponse{
		Body:       "Awesome, Your Server health is good!!!",
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(handler)
}
