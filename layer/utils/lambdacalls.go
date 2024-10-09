package utils

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
)

// ProfileLambdaCallPayload represents the payload to send to the CallProfileFunction
type ProfileLambdaCallPayload struct {
	UserId    string `json:"userId"`
	SessionID string `json:"sessionId"`
}

// APIGatewayProxyRequestWrapper wraps the ProfileLambdaCallPayload inside the Body field
type APIGatewayProxyRequestWrapper struct {
	Body string `json:"body"`
}

func InvokeProfileLambda(payload ProfileLambdaCallPayload) error {
	session := session.Must(session.NewSession())
	client := lambda.New(session)

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshalling payload: %w", err)
	}

	// wrap the payload inside the body field
	wrapper := APIGatewayProxyRequestWrapper{
		Body: string(payloadBytes),
	}

	// marshal the wrapper back to json
	wrapperBytes, err := json.Marshal(wrapper)
	if err != nil {
		return fmt.Errorf("error marshalling wrapper: %w", err)
	}

	functionName := os.Getenv("profileFunctionLambdaName")
	if functionName == "" {
		return fmt.Errorf("profileFunctionLambdaName is not set")
	}

	input := &lambda.InvokeInput{
		FunctionName: aws.String(functionName),
		Payload:      wrapperBytes,
	}

	_, err = client.Invoke(input)
	if err != nil {
		return fmt.Errorf("error invoking lambda: %w", err)
	}

	return nil
}
