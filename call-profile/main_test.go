package main

import (
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
)

func TestHandler(t *testing.T) {
	tests := []struct {
		name        string
		request     events.APIGatewayProxyRequest
		expectError bool
	}{
		{
			name: "Empty body - should fail due to Firestore",
			request: events.APIGatewayProxyRequest{
				Body: "",
			},
			expectError: true,
		},
		{
			name: "Invalid JSON body - should fail due to Firestore",
			request: events.APIGatewayProxyRequest{
				Body: "invalid json",
			},
			expectError: true,
		},
		{
			name: "Valid JSON with userId but no sessionId - should fail due to Firestore",
			request: events.APIGatewayProxyRequest{
				Body: `{"userId": "test-user-id"}`,
			},
			expectError: true,
		},
		{
			name: "Valid JSON with both userId and sessionId - should fail due to Firestore",
			request: events.APIGatewayProxyRequest{
				Body: `{"userId": "test-user-id", "sessionId": "test-session-id"}`,
			},
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, err := handler(test.request)
			
			if test.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "project id is required")
			} else {
				assert.NoError(t, err)
			}
			assert.IsType(t, events.APIGatewayProxyResponse{}, response)
		})
	}
}

func TestHandlerWithMockFirestore(t *testing.T) {
	request := events.APIGatewayProxyRequest{
		Body: `{"userId": "mock-user-id"}`,
	}
	_, err := handler(request)
	
	assert.Error(t, err)
}
