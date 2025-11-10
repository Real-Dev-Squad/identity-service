package utils

import (
	"errors"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
)

var (
	ErrInvalidUserID      = errors.New("invalid user ID")
	ErrProfileURLNotFound = errors.New("profile URL not found")
	ErrChaincodeNotFound  = errors.New("chaincode not found")
	ErrChaincodeEmpty     = errors.New("chaincode is empty")
	ErrUserDataNotFound   = errors.New("user data not found")
	ErrServiceDown        = errors.New("profile service is down")
)

type ProfileError struct {
	Code    string
	Message string
	Err     error
}

func (e *ProfileError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *ProfileError) Unwrap() error {
	return e.Err
}

func NewProfileError(code, message string, err error) *ProfileError {
	return &ProfileError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func HandleLambdaError(err error) (events.APIGatewayProxyResponse, error) {
	if err == nil {
		return events.APIGatewayProxyResponse{
			Body:       "Internal server error",
			StatusCode: 500,
		}, nil
	}

	if profileErr, ok := err.(*ProfileError); ok {
		return events.APIGatewayProxyResponse{
			Body:       fmt.Sprintf("Profile Error: %s", profileErr.Message),
			StatusCode: 400,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		Body:       fmt.Sprintf("Error: %v", err),
		StatusCode: 500,
	}, nil
}

func HandleProfileSkippedError(reason string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		Body:       fmt.Sprintf("Profile Skipped: %s", reason),
		StatusCode: 200,
	}
}
