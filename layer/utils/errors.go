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
	Status  int
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

func NewProfileError(code, message string, status int, err error) *ProfileError {
	return &ProfileError{
		Code:    code,
		Message: message,
		Status:  status,
		Err:     err,
	}
}

func HandleLambdaError(err error) (events.APIGatewayProxyResponse, error) {
	if err == nil {
		panic("HandleLambdaError called with nil error - this indicates a programming error")
	}

	if profileErr, ok := err.(*ProfileError); ok {
		statusCode := profileErr.Status
		if statusCode == 0 {
			statusCode = 400
		}
		return events.APIGatewayProxyResponse{
			Body:       fmt.Sprintf("Profile Error: %s", profileErr.Message),
			StatusCode: statusCode,
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
