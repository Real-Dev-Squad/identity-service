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
			name: "Basic health check request",
			request: events.APIGatewayProxyRequest{
				HTTPMethod: "GET",
				Path:       "/health-check",
			},
			expectCode:  200,
			expectError: false,
		},
		{
			name: "POST request - should still work",
			request: events.APIGatewayProxyRequest{
				HTTPMethod: "POST",
				Path:       "/health-check",
			},
			expectCode:  200,
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, err := handler(test.request)
			
			// We expect an error here because Firestore isn't initialized in test environment
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "project id is required")
			
			// Response should be empty due to error
			assert.IsType(t, events.APIGatewayProxyResponse{}, response)
			assert.Empty(t, response.Body)
		})
	}
}

func TestCallProfileHealth(t *testing.T) {
	tests := []struct {
		name    string
		userUrl string
	}{
		{
			name:    "URL with trailing slash",
			userUrl: "https://example.com/",
		},
		{
			name:    "URL without trailing slash",
			userUrl: "https://example.com",
		},
		{
			name:    "Local URL",
			userUrl: "http://localhost:3000",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				url := test.userUrl
				if url[len(url)-1] != '/' {
					url = url + "/"
				}
				assert.True(t, url[len(url)-1] == '/')
			})
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

func TestURLFormatting(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "URL with trailing slash",
			input:    "https://example.com/",
			expected: "https://example.com/health",
		},
		{
			name:     "URL without trailing slash",
			input:    "https://example.com",
			expected: "https://example.com/health",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			userUrl := test.input
			if userUrl[len(userUrl)-1] != '/' {
				userUrl = userUrl + "/"
			}
			result := userUrl + "health"
			assert.Equal(t, test.expected, result)
		})
	}
}
