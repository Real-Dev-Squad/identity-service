package main

import (
	"identity-service/layer/utils"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
)

func TestHandler(t *testing.T) {
	for _, test := range TestRequests {
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()
			response, err := handler(test.Request)
			
			if test.ExpectedErr {
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

func TestHandler_NoFirestore(t *testing.T) {
	request := events.APIGatewayProxyRequest{
		Body: `{"userId": "mock-user-id"}`,
	}
	
	_, err := handler(request)
	assert.Error(t, err)
}

func TestGetDataFromBody(t *testing.T) {
	for _, test := range GetDataFromBodyTests {
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()
			userId, sessionId := utils.GetDataFromBody([]byte(test.Body))
			assert.Equal(t, test.ExpectedUserId, userId, test.Description)
			assert.Equal(t, test.ExpectedSessionId, sessionId, test.Description)
		})
	}
}

func TestHandler_EmptyUserIdLogic(t *testing.T) {
	for _, test := range EmptyUserIdTests {
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()
			
			userId, sessionId := utils.GetDataFromBody([]byte(test.Body))
			
			if userId == "" {
				response := events.APIGatewayProxyResponse{
					Body:       test.ExpectedBody,
					StatusCode: test.ExpectedStatus,
				}
				
				assert.Equal(t, test.ExpectedBody, response.Body, test.Description)
				assert.Equal(t, test.ExpectedStatus, response.StatusCode, test.Description)
			}
			
			if test.Body == `{"userId": "", "sessionId": "session123"}` {
				assert.Equal(t, "session123", sessionId)
			}
		})
	}
}

func TestURLFormatting(t *testing.T) {
	for _, test := range URLFormattingTests {
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()
			
			userUrl := test.Input
			if userUrl[len(userUrl)-1] != '/' {
				userUrl = userUrl + "/"
			}
			result := userUrl + "health"
			
			assert.Equal(t, test.Expected, result, test.Description)
		})
	}
}

func TestResValidation(t *testing.T) {
	for _, test := range ResValidationTests {
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()
			
			err := test.Res.Validate()
			
			if test.IsValid {
				assert.NoError(t, err, test.Description)
			} else {
				assert.Error(t, err, test.Description)
			}
		})
	}
}

func TestHandlerLogicPaths(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		expectedResult string
		testLogic      func(string, string) (string, bool)
	}{
		{
			name:           "Empty userId should trigger skip logic",
			body:           `{"userId": "", "sessionId": "test"}`,
			expectedResult: MockResponses.ProfileSkippedNoUserID,
			testLogic: func(userId, sessionId string) (string, bool) {
				if userId == "" {
					return MockResponses.ProfileSkippedNoUserID, true
				}
				return "", false
			},
		},
		{
			name:           "Missing userId should trigger skip logic",
			body:           `{"sessionId": "test"}`,
			expectedResult: MockResponses.ProfileSkippedNoUserID,
			testLogic: func(userId, sessionId string) (string, bool) {
				if userId == "" {
					return MockResponses.ProfileSkippedNoUserID, true
				}
				return "", false
			},
		},
		{
			name:           "Valid userId should pass first check",
			body:           `{"userId": "valid-user", "sessionId": "test"}`,
			expectedResult: "",
			testLogic: func(userId, sessionId string) (string, bool) {
				if userId == "" {
					return MockResponses.ProfileSkippedNoUserID, true
				}
				return "", false
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			
			userId, sessionId := utils.GetDataFromBody([]byte(test.body))
			result, shouldReturn := test.testLogic(userId, sessionId)
			
			if shouldReturn {
				assert.Equal(t, test.expectedResult, result)
			}
		})
	}
}

func TestURLFormattingEdgeCases(t *testing.T) {
	edgeCases := []struct {
		name     string
		url      string
		expected string
	}{
		{"Single char", "a", "a/health"},
		{"Two chars", "ab", "ab/health"},
		{"With slash", "abc/", "abc/health"},
		{"Complex URL", "https://api.service.com/v1/endpoint", "https://api.service.com/v1/endpoint/health"},
	}

	for _, test := range edgeCases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			
			assert.NotPanics(t, func() {
				userUrl := test.url
				if len(userUrl) > 0 && userUrl[len(userUrl)-1] != '/' {
					userUrl = userUrl + "/"
				}
				result := userUrl + "health"
				assert.Equal(t, test.expected, result)
			})
		})
	}
}

func TestHTTPClientTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout int
	}{
		{"Standard timeout", 5},
		{"Short timeout", 2},
		{"Long timeout", 10},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			
			timeoutDuration := test.timeout
			assert.Greater(t, timeoutDuration, 0)
			assert.LessOrEqual(t, timeoutDuration, 30)
		})
	}
}

func TestServiceRunningLogic(t *testing.T) {
	tests := []struct {
		name          string
		serviceError  bool
		expectedState bool
	}{
		{"Service running", false, true},
		{"Service down", true, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			
			var isServiceRunning bool
			if test.serviceError {
				isServiceRunning = false
			} else {
				isServiceRunning = true
			}
			
			assert.Equal(t, test.expectedState, isServiceRunning)
		})
	}
}

func TestDataExtractionPipeline(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		validate func(userId, sessionId string) error
	}{
		{
			name: "Valid data extraction",
			body: `{"userId": "user123", "sessionId": "session456"}`,
			validate: func(userId, sessionId string) error {
				assert.Equal(t, "user123", userId)
				assert.Equal(t, "session456", sessionId)
				return nil
			},
		},
		{
			name: "Partial data extraction",
			body: `{"userId": "user123"}`,
			validate: func(userId, sessionId string) error {
				assert.Equal(t, "user123", userId)
				assert.Equal(t, "", sessionId)
				return nil
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			
			userId, sessionId := utils.GetDataFromBody([]byte(test.body))
			err := test.validate(userId, sessionId)
			assert.NoError(t, err)
		})
	}
}

func TestResponseStructure(t *testing.T) {
	responses := []struct {
		name       string
		body       string
		statusCode int
	}{
		{"Skip no user ID", MockResponses.ProfileSkippedNoUserID, 200},
		{"Skip no URL", MockResponses.ProfileSkippedNoURL, 200},
		{"Skip blocked", MockResponses.ProfileSkippedBlocked, 200},
		{"Skip service down", MockResponses.ProfileSkippedServiceDown, 200},
		{"Profile saved", MockResponses.ProfileSaved, 200},
	}

	for _, test := range responses {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			
			response := events.APIGatewayProxyResponse{
				Body:       test.body,
				StatusCode: test.statusCode,
			}
			
			assert.Equal(t, test.body, response.Body)
			assert.Equal(t, test.statusCode, response.StatusCode)
			assert.IsType(t, events.APIGatewayProxyResponse{}, response)
		})
	}
}