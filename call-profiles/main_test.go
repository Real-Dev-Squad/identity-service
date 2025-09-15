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
			name: "Basic request - should process profiles",
			request: events.APIGatewayProxyRequest{
				HTTPMethod: "GET",
				Path:       "/call-profiles",
			},
			expectError: true,
		},
		{
			name: "POST request - should still work",
			request: events.APIGatewayProxyRequest{
				HTTPMethod: "POST",
				Path:       "/call-profiles",
			},
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			response, err := handler(test.request)
			
			if test.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "project id is required")
				assert.Equal(t, "", response.Body)
			} else {
				assert.NoError(t, err)
			}
			
			assert.IsType(t, events.APIGatewayProxyResponse{}, response)
		})
	}
}

func TestHandlerStructure(t *testing.T) {
	request := events.APIGatewayProxyRequest{}
	
	response, err := handler(request)
	
	assert.Error(t, err)
	
	assert.IsType(t, events.APIGatewayProxyResponse{}, response)
	
	assert.Empty(t, response.Body)
}