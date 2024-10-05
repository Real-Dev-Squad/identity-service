package utils

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
)

type ProfileLambdaCallPayload struct {
	UserId    string `json:"userId"`
	SessionID string `json:"sessionId"`
}

func InvokeProfileLambda(payload ProfileLambdaCallPayload) error {
	session := session.Must(session.NewSession())
	client := lambda.New(session)

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshalling payload: %w", err)
	}

	functionName := "CallProfileFunction"
	if functionName == "" {
		return fmt.Errorf("PROFILE_LAMBDA_FUNCTION_ARN is not set")
	}

	input := &lambda.InvokeInput{
		FunctionName: aws.String(functionName),
		Payload:      payloadBytes,
	}

	_, err = client.Invoke(input)
	if err != nil {
		return fmt.Errorf("error invoking lambda: %w", err)
	}

	return nil
}
