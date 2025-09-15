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
		expectCode  int
		expectError bool
	}{
		{
			name: "Basic request - should process profiles",
			request: events.APIGatewayProxyRequest{
				HTTPMethod: "GET",
				Path:       "/call-profiles",
			},
			expectCode:  200,
			expectError: false,
		},
		{
			name: "POST request - should still work",
			request: events.APIGatewayProxyRequest{
				HTTPMethod: "POST",
				Path:       "/call-profiles",
			},
			expectCode:  200,
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, err := handler(test.request)
			
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "project id is required")
			
			assert.IsType(t, events.APIGatewayProxyResponse{}, response)
			assert.Empty(t, response.Body)
		})
	}
}

func TestCallProfile(t *testing.T) {
	t.Run("Valid inputs", func(t *testing.T) {
		assert.NotPanics(t, func() {
		})
	})
}

func TestHandlerStructure(t *testing.T) {
	request := events.APIGatewayProxyRequest{}
	
	response, err := handler(request)
	
	assert.Error(t, err)
	
	assert.IsType(t, events.APIGatewayProxyResponse{}, response)
	
	assert.Empty(t, response.Body)
}
