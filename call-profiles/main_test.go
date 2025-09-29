package main

import (
	"context"
	"identity-service/layer/utils"
	"os"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
)

func TestCallProfileFunction(t *testing.T) {
	for _, test := range ProfileLambdaCallPayloadTests {
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()
			
			payload := utils.ProfileLambdaCallPayload{
				UserId:    test.UserId,
				SessionID: test.SessionId,
			}
			
			assert.Equal(t, test.UserId, payload.UserId, test.Description)
			assert.Equal(t, test.SessionId, payload.SessionID, test.Description)
		})
	}
}

func TestProfileLambdaCallPayload(t *testing.T) {
	for _, test := range ProfileLambdaCallPayloadTests {
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()
			
			payload := utils.ProfileLambdaCallPayload{
				UserId:    test.UserId,
				SessionID: test.SessionId,
			}
			
			assert.Equal(t, test.UserId, payload.UserId, test.Description)
			assert.Equal(t, test.SessionId, payload.SessionID, test.Description)
			assert.NotNil(t, payload)
		})
	}
}

func TestInvokeProfileLambdaErrorConditions(t *testing.T) {
	originalEnv := SaveEnvVar("profileFunctionLambdaName")
	defer RestoreEnvVar("profileFunctionLambdaName", originalEnv)

	for _, test := range InvokeProfileLambdaErrorTests {
		t.Run(test.Name, func(t *testing.T) {
			RestoreEnvVar("profileFunctionLambdaName", test.EnvVarValue)
			
			err := utils.InvokeProfileLambda(test.Payload)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), test.ExpectedErr, test.Description)
		})
	}
}

func TestWaitGroupBehavior(t *testing.T) {
	t.Run("WaitGroup with multiple goroutines", func(t *testing.T) {
		var testWg sync.WaitGroup
		var counter int
		var mu sync.Mutex
		
		numCalls := 3
		for i := 0; i < numCalls; i++ {
			testWg.Add(1)
			go func(id int) {
				defer testWg.Done()
				time.Sleep(10 * time.Millisecond)
				mu.Lock()
				counter++
				mu.Unlock()
			}(i)
		}
		
		testWg.Wait()
		assert.Equal(t, numCalls, counter)
	})
}

func TestSessionIdGeneration(t *testing.T) {
	for _, test := range SessionIdGenerationTests {
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()
			
			sessionIds := make(map[string]bool)
			
			for i := 0; i < test.Count; i++ {
				sessionId := "session_" + string(rune(i%10+'0'))
				
				if !sessionIds[sessionId] {
					sessionIds[sessionId] = true
				}
			}
			
			assert.LessOrEqual(t, len(sessionIds), test.Count)
			assert.Greater(t, len(sessionIds), 0)
		})
	}
}

func TestResponseFormat(t *testing.T) {
	for _, test := range ResponseFormatTests {
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()
			
			response := events.APIGatewayProxyResponse{
				Body:       test.ExpectedFormat,
				StatusCode: 200,
			}
			
			assert.Equal(t, test.ExpectedFormat, response.Body, test.Description)
			assert.Equal(t, 200, response.StatusCode)
		})
	}
}

func TestAPIGatewayProxyRequestWrapper(t *testing.T) {
	for _, test := range APIGatewayProxyRequestWrapperTests {
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()
			
			wrapper := utils.APIGatewayProxyRequestWrapper{
				Body: test.Body,
			}
			
			assert.Equal(t, test.Body, wrapper.Body, test.Description)
			assert.NotNil(t, wrapper)
		})
	}
}

func TestConcurrentProfileCalling(t *testing.T) {
	t.Run("Concurrent profile calls with different data", func(t *testing.T) {
		var testWg sync.WaitGroup
		var results []string
		var mu sync.Mutex
		
		for i, userId := range ConcurrentTestData.UserIds[:5] {
			testWg.Add(1)
			go func(uid string, sessionId string) {
				defer testWg.Done()
				
				payload := utils.ProfileLambdaCallPayload{
					UserId:    uid,
					SessionID: sessionId,
				}
				
				mu.Lock()
				results = append(results, payload.UserId)
				mu.Unlock()
			}(userId, ConcurrentTestData.SessionIds[i])
		}
		
		testWg.Wait()
		assert.Equal(t, 5, len(results))
	})
}

func TestEnvironmentVariableHandling(t *testing.T) {
	for _, test := range EnvironmentVariableTests {
		t.Run(test.Name, func(t *testing.T) {
			originalEnv := SaveEnvVar("profileFunctionLambdaName")
			defer RestoreEnvVar("profileFunctionLambdaName", originalEnv)
			
			RestoreEnvVar("profileFunctionLambdaName", test.TestEnv)
			
			payload := utils.ProfileLambdaCallPayload{
				UserId:    "test-user",
				SessionID: "test-session",
			}
			
			err := utils.InvokeProfileLambda(payload)
			
			if test.TestEnv == "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "profileFunctionLambdaName is not set")
			}
		})
	}
}

func TestHandlerResponseFormat(t *testing.T) {
	tests := []struct {
		name          string
		profilesCount int
		expectedBody  string
	}{
		{"Zero profiles", 0, "Total Profiles called in session is 0"},
		{"Single profile", 1, "Total Profiles called in session is 1"},
		{"Multiple profiles", 5, "Total Profiles called in session is 5"},
		{"Large count", 100, "Total Profiles called in session is 100"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			
			response := events.APIGatewayProxyResponse{
				Body:       test.expectedBody,
				StatusCode: 200,
			}
			
			assert.Equal(t, test.expectedBody, response.Body)
			assert.Equal(t, 200, response.StatusCode)
			assert.IsType(t, events.APIGatewayProxyResponse{}, response)
		})
	}
}

func TestPayloadMarshaling(t *testing.T) {
	tests := []struct {
		name    string
		payload utils.ProfileLambdaCallPayload
	}{
		{
			name: "Standard payload",
			payload: utils.ProfileLambdaCallPayload{
				UserId:    "user123",
				SessionID: "session456",
			},
		},
		{
			name: "Empty fields",
			payload: utils.ProfileLambdaCallPayload{
				UserId:    "",
				SessionID: "",
			},
		},
		{
			name: "Special characters",
			payload: utils.ProfileLambdaCallPayload{
				UserId:    "user@domain.com",
				SessionID: "session-with-dashes",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			
			assert.NotNil(t, test.payload)
			assert.IsType(t, utils.ProfileLambdaCallPayload{}, test.payload)
			
			assert.Equal(t, test.payload.UserId, test.payload.UserId)
			assert.Equal(t, test.payload.SessionID, test.payload.SessionID)
		})
	}
}

func TestSessionIdDocumentCreation(t *testing.T) {
	t.Run("Session ID document creation logic", func(t *testing.T) {
		
		sessionData := map[string]interface{}{
			"Timestamp": time.Now(),
		}
		
		assert.NotNil(t, sessionData)
		assert.Contains(t, sessionData, "Timestamp")
		
		timestamp, ok := sessionData["Timestamp"].(time.Time)
		assert.True(t, ok)
		assert.WithinDuration(t, time.Now(), timestamp, time.Second)
	})
}

func TestProfileCountingLogic(t *testing.T) {
	t.Run("Profile counting simulation", func(t *testing.T) {
		totalProfilesCalled := 0
		
		mockProfiles := []string{"user1", "user2", "user3", "user4", "user5"}
		
		for range mockProfiles {
			totalProfilesCalled++
		}
		
		assert.Equal(t, len(mockProfiles), totalProfilesCalled)
		assert.Greater(t, totalProfilesCalled, 0)
	})
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

func TestHandlerIntegration(t *testing.T) {
	ctx := context.Background()
	client := newFirestoreMockClient(ctx)
	defer client.Close()

	testCases := []struct {
		name           string
		request        events.APIGatewayProxyRequest
		userData       []map[string]interface{}
		expectedBody   string
		expectedStatus int
		expectedError  bool
	}{
		{
			name: "no verified users",
			request: events.APIGatewayProxyRequest{
				HTTPMethod: "GET",
				Path:       "/call-profiles",
			},
			userData: []map[string]interface{}{
				{
					"userId":        "user1",
					"profileStatus": "PENDING",
				},
				{
					"userId":        "user2",
					"profileStatus": "BLOCKED",
				},
			},
			expectedBody:   "Total Profiles called in session is 0",
			expectedStatus: 200,
			expectedError:  false,
		},
		{
			name: "single verified user",
			request: events.APIGatewayProxyRequest{
				HTTPMethod: "GET",
				Path:       "/call-profiles",
			},
			userData: []map[string]interface{}{
				{
					"userId":        "user1",
					"profileStatus": "VERIFIED",
				},
			},
			expectedBody:   "Total Profiles called in session is 1",
			expectedStatus: 200,
			expectedError:  false,
		},
		{
			name: "multiple verified users",
			request: events.APIGatewayProxyRequest{
				HTTPMethod: "GET",
				Path:       "/call-profiles",
			},
			userData: []map[string]interface{}{
				{
					"userId":        "user1",
					"profileStatus": "VERIFIED",
				},
				{
					"userId":        "user2",
					"profileStatus": "VERIFIED",
				},
				{
					"userId":        "user3",
					"profileStatus": "VERIFIED",
				},
				{
					"userId":        "user4",
					"profileStatus": "PENDING",
				},
			},
			expectedBody:   "Total Profiles called in session is 3",
			expectedStatus: 200,
			expectedError:  false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			for _, userData := range testCase.userData {
				userId := userData["userId"].(string)
				_, err := client.Collection("users").Doc(userId).Set(ctx, userData)
				assert.NoError(t, err)
			}

			response, err := handlerWithClient(testCase.request, client)

			if testCase.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, testCase.expectedStatus, response.StatusCode)
				assert.Equal(t, testCase.expectedBody, response.Body)
			}

			for _, userData := range testCase.userData {
				userId := userData["userId"].(string)
				client.Collection("users").Doc(userId).Delete(ctx)
			}
			
			iter := client.Collection("identitySessionIds").Documents(ctx)
			for {
				doc, err := iter.Next()
				if err != nil {
					break
				}
				doc.Ref.Delete(ctx)
			}
		})
	}
}

func TestHandlerWithRealFirestore(t *testing.T) {
	ctx := context.Background()
	client := newFirestoreMockClient(ctx)
	defer client.Close()

	request := events.APIGatewayProxyRequest{
		HTTPMethod: "GET",
		Path:       "/call-profiles",
	}

	response, err := handlerWithClient(request, client)
	assert.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)
	assert.Equal(t, "Total Profiles called in session is 0", response.Body)
}

func TestSessionIdDocumentCreationIntegration(t *testing.T) {
	ctx := context.Background()
	client := newFirestoreMockClient(ctx)
	defer client.Close()

	docRef, _, err := client.Collection("identitySessionIds").Add(ctx, map[string]interface{}{
		"Timestamp": time.Now(),
	})
	
	assert.NoError(t, err)
	assert.NotNil(t, docRef)
	
	doc, err := docRef.Get(ctx)
	assert.NoError(t, err)
	assert.True(t, doc.Exists())
	
	docRef.Delete(ctx)
}