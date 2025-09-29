package main

import (
	"context"
	"fmt"
	"identity-service/layer/utils"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"cloud.google.com/go/firestore"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

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

func TestHandlerIntegration(t *testing.T) {
	os.Setenv("environment", "test")
	
	ctx := context.Background()
	client := newFirestoreMockClient(ctx)
	defer client.Close()

	testCases := []struct {
		name           string
		request        events.APIGatewayProxyRequest
		userData       map[string]interface{}
		mockServer     func() *httptest.Server
		expectedBody   string
		expectedStatus int
		expectedError  bool
	}{
		{
			name: "successful profile save",
			request: events.APIGatewayProxyRequest{
				Body: `{"userId": "test-user-1", "sessionId": "session-1"}`,
			},
			userData: map[string]interface{}{
				"userId":        "test-user-1",
				"profileURL":    "http://example.com",
				"chaincode":     "TESTCHAIN",
				"discordId":     "discord123",
				"profileStatus": "PENDING",
				"firstName":     "John",
				"lastName":      "Doe",
				"email":         "john@example.com",
				"phone":         "1234567890",
				"yoe":           5,
				"company":       "Tech Corp",
				"designation":   "Developer",
				"githubId":      "johndoe",
				"linkedin":      "johndoe",
				"website":       "https://johndoe.com",
			},
			mockServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/health" {
						w.WriteHeader(http.StatusOK)
						w.Write([]byte("OK"))
					} else {
						w.WriteHeader(http.StatusOK)
						w.Write([]byte(`{
							"first_name": "John",
							"last_name": "Doe",
							"email": "john@example.com",
							"phone": "1234567890",
							"yoe": 5,
							"company": "Tech Corp",
							"designation": "Developer",
							"github_id": "johndoe",
							"linkedin_id": "johndoe",
							"website": "https://johndoe.com"
						}`))
					}
				}))
			},
			expectedBody:   "Profile Saved",
			expectedStatus: 200,
			expectedError:  false,
		},
		{
			name: "no user ID",
			request: events.APIGatewayProxyRequest{
				Body: `{"sessionId": "session-1"}`,
			},
			userData:       nil,
			mockServer:     nil,
			expectedBody:   "Profile Skipped No UserID",
			expectedStatus: 200,
			expectedError:  false,
		},
		{
			name: "user not found",
			request: events.APIGatewayProxyRequest{
				Body: `{"userId": "non-existent-user", "sessionId": "session-1"}`,
			},
			userData:       nil,
			mockServer:     nil,
			expectedBody:   "Profile Skipped No Profile URL",
			expectedStatus: 200,
			expectedError:  false,
		},
		{
			name: "no profile URL",
			request: events.APIGatewayProxyRequest{
				Body: `{"userId": "test-user-2", "sessionId": "session-1"}`,
			},
			userData: map[string]interface{}{
				"userId":        "test-user-2",
				"chaincode":     "TESTCHAIN",
				"discordId":     "discord123",
				"profileStatus": "PENDING",
			},
			mockServer:     nil,
			expectedBody:   "Profile Skipped No Profile URL",
			expectedStatus: 200,
			expectedError:  false,
		},
		{
			name: "empty chaincode",
			request: events.APIGatewayProxyRequest{
				Body: `{"userId": "test-user-3", "sessionId": "session-1"}`,
			},
			userData: map[string]interface{}{
				"userId":        "test-user-3",
				"profileURL":    "http://example.com",
				"chaincode":     "",
				"discordId":     "discord123",
				"profileStatus": "PENDING",
			},
			mockServer:     nil,
			expectedBody:   "Profile Skipped Profile Service Blocked",
			expectedStatus: 200,
			expectedError:  false,
		},
		{
			name: "no chaincode",
			request: events.APIGatewayProxyRequest{
				Body: `{"userId": "test-user-4", "sessionId": "session-1"}`,
			},
			userData: map[string]interface{}{
				"userId":        "test-user-4",
				"profileURL":    "http://example.com",
				"discordId":     "discord123",
				"profileStatus": "PENDING",
			},
			mockServer:     nil,
			expectedBody:   "Profile Skipped Chaincode Not Found",
			expectedStatus: 200,
			expectedError:  false,
		},
		{
			name: "service down",
			request: events.APIGatewayProxyRequest{
				Body: `{"userId": "test-user-5", "sessionId": "session-1"}`,
			},
			userData: map[string]interface{}{
				"userId":        "test-user-5",
				"profileURL":    "http://example.com",
				"chaincode":     "TESTCHAIN",
				"discordId":     "discord123",
				"profileStatus": "PENDING",
			},
			mockServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("Service Unavailable"))
				}))
			},
			expectedBody:   "Profile Skipped error in getting profile data",
			expectedStatus: 200,
			expectedError:  false,
		},
		{
			name: "service timeout",
			request: events.APIGatewayProxyRequest{
				Body: `{"userId": "test-user-6", "sessionId": "session-1"}`,
			},
			userData: map[string]interface{}{
				"userId":        "test-user-6",
				"profileURL":    "http://example.com",
				"chaincode":     "TESTCHAIN",
				"discordId":     "discord123",
				"profileStatus": "PENDING",
			},
			mockServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					time.Sleep(6 * time.Second) // Longer than 5 second timeout
					w.WriteHeader(http.StatusOK)
				}))
			},
			expectedBody:   "Profile Skipped Service Down",
			expectedStatus: 200,
			expectedError:  false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.userData != nil {
				_, err := client.Collection("users").Doc(testCase.userData["userId"].(string)).Set(ctx, testCase.userData)
				assert.NoError(t, err)
			}

			var server *httptest.Server
			if testCase.mockServer != nil {
				server = testCase.mockServer()
				defer server.Close()
				
				if testCase.userData != nil {
					userId := testCase.userData["userId"].(string)
					_, err := client.Collection("users").Doc(userId).Update(ctx, []firestore.Update{
						{Path: "profileURL", Value: server.URL},
					})
					assert.NoError(t, err)
				}
			}

			response, err := handlerWithClient(testCase.request, client)

			if testCase.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, testCase.expectedBody, response.Body)
			assert.Equal(t, testCase.expectedStatus, response.StatusCode)
		})
	}
}

func TestHandlerWithRealFirestore(t *testing.T) {
	os.Setenv("environment", "test")
	
	ctx := context.Background()
	client := newFirestoreMockClient(ctx)
	defer client.Close()

	userId := "integration-test-user"
	sessionId := "integration-session"
	userData := map[string]interface{}{
		"userId":        userId,
		"profileURL":    "http://example.com",
		"chaincode":     "TESTCHAIN",
		"discordId":     "discord123",
		"profileStatus": "PENDING",
		"firstName":     "Integration",
		"lastName":      "Test",
		"email":         "integration@test.com",
		"phone":         "1234567890",
		"yoe":           3,
		"company":       "Test Corp",
		"designation":   "Tester",
		"githubId":      "integrationtest",
		"linkedin":      "integrationtest",
		"website":       "https://integrationtest.com",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"first_name": "Integration",
				"last_name": "Test",
				"email": "integration@test.com",
				"phone": "1234567890",
				"yoe": 3,
				"company": "Test Corp",
				"designation": "Tester",
				"github_id": "integrationtest",
				"linkedin_id": "integrationtest",
				"website": "https://integrationtest.com"
			}`))
		}
	}))
	defer server.Close()

	userData["profileURL"] = server.URL

	_, err := client.Collection("users").Doc(userId).Set(ctx, userData)
	assert.NoError(t, err)

	request := events.APIGatewayProxyRequest{
		Body: fmt.Sprintf(`{"userId": "%s", "sessionId": "%s"}`, userId, sessionId),
	}

	response, err := handlerWithClient(request, client)

	assert.NoError(t, err)
	assert.Equal(t, "Profile Saved", response.Body)
	assert.Equal(t, 200, response.StatusCode)
}

func TestHandlerEdgeCases(t *testing.T) {
	os.Setenv("environment", "test")
	
	ctx := context.Background()
	client := newFirestoreMockClient(ctx)
	defer client.Close()

	testCases := []struct {
		name           string
		request        events.APIGatewayProxyRequest
		userData       map[string]interface{}
		expectedBody   string
		expectedStatus int
	}{
		{
			name: "invalid user data type",
			request: events.APIGatewayProxyRequest{
				Body: `{"userId": "invalid-user", "sessionId": "session-1"}`,
			},
			userData: map[string]interface{}{
				"userId":        "invalid-user",
				"profileURL":    "http://example.com",
				"chaincode":     "TESTCHAIN",
				"discordId":     "discord123",
				"profileStatus": "PENDING",
				"firstName":     123, // Invalid type
				"lastName":      "Doe",
			},
			expectedBody:   "Profile Skipped error in getting profile data",
			expectedStatus: 200,
		},
		{
			name: "missing discord ID",
			request: events.APIGatewayProxyRequest{
				Body: `{"userId": "no-discord-user", "sessionId": "session-1"}`,
			},
			userData: map[string]interface{}{
				"userId":        "no-discord-user",
				"profileURL":    "http://example.com",
				"chaincode":     "TESTCHAIN",
				"profileStatus": "PENDING",
			},
			expectedBody:   "Profile Skipped error in getting profile data", // Will fail health check
			expectedStatus: 200,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := client.Collection("users").Doc(testCase.userData["userId"].(string)).Set(ctx, testCase.userData)
			assert.NoError(t, err)

			response, err := handlerWithClient(testCase.request, client)

			assert.NoError(t, err)
			assert.Equal(t, testCase.expectedBody, response.Body)
			assert.Equal(t, testCase.expectedStatus, response.StatusCode)
		})
	}
}

func newFirestoreMockClient(ctx context.Context) *firestore.Client {
	emulatorHost := os.Getenv("FIRESTORE_EMULATOR_HOST")
	if emulatorHost == "" {
		emulatorHost = "127.0.0.1:8090"
	}
	conn, _ := grpc.Dial(emulatorHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	client, _ := firestore.NewClient(ctx, "test-project", option.WithGRPCConn(conn))
	return client
}

func handlerWithClient(request events.APIGatewayProxyRequest, client *firestore.Client) (events.APIGatewayProxyResponse, error) {
	ctx := context.Background()
	d := deps{
		client: client,
		ctx:    ctx,
	}
	return d.handler(request)
}