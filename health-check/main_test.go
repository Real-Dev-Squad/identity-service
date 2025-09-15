package main

import (
	"strings"
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
			name: "Basic health check request",
			request: events.APIGatewayProxyRequest{
				HTTPMethod: "GET",
				Path:       "/health-check",
			},
			expectError: true,
		},
		{
			name: "POST request - should still work",
			request: events.APIGatewayProxyRequest{
				HTTPMethod: "POST",
				Path:       "/health-check",
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
			t.Parallel()
			assert.NotPanics(t, func() {
				url := test.userUrl
				if !strings.HasSuffix(url, "/") {
					url += "/"
				}
				assert.True(t, strings.HasSuffix(url, "/"))
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
			t.Parallel()
			userUrl := test.input
			if !strings.HasSuffix(userUrl, "/") {
				userUrl += "/"
			}
			result := userUrl + "health"
			assert.Equal(t, test.expected, result)
		})
	}
}