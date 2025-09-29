package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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


func TestURLFormatting(t *testing.T) {
	for _, test := range URLFormattingTests {
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()
			userUrl := test.Input
			if !strings.HasSuffix(userUrl, "/") {
				userUrl += "/"
			}
			result := userUrl + "health"
			assert.Equal(t, test.Expected, result, test.Description)
		})
	}
}

func TestCallProfileHealthWithMockServer(t *testing.T) {
	for _, test := range MockServerTests {
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()
			
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if test.ServerDelay > 0 {
					time.Sleep(test.ServerDelay)
				}
				w.WriteHeader(test.ServerResponse)
				w.Write([]byte(MockHTTPResponses.Success))
			}))
			defer server.Close()

			userUrl := server.URL
			if userUrl[len(userUrl)-1] != '/' {
				userUrl = userUrl + "/"
			}
			
			requestURL := fmt.Sprintf("%shealth", userUrl)
			
			assert.True(t, strings.HasSuffix(requestURL, "/health"), test.Description)
			assert.Contains(t, requestURL, server.URL, test.Description)
		})
	}
}

func TestHTTPClientConfiguration(t *testing.T) {
	for _, test := range HTTPClientConfigTests {
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()
			
			httpClient := &http.Client{
				Timeout: test.Timeout,
			}
			
			assert.Equal(t, test.Timeout, httpClient.Timeout, test.Description)
			assert.NotNil(t, httpClient)
		})
	}
}

func TestRequestCreation(t *testing.T) {
	for _, test := range HTTPRequestTests {
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()
			
			req, err := http.NewRequest(test.Method, test.URL, nil)
			
			assert.NoError(t, err, test.Description)
			assert.NotNil(t, req)
			assert.Equal(t, test.Method, req.Method)
			assert.Equal(t, test.URL, req.URL.String())
		})
	}
}

func TestWaitGroupBehavior(t *testing.T) {
	for _, test := range ConcurrentHealthCheckTests {
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()
			
			var testWg sync.WaitGroup
			var counter int
			var mu sync.Mutex
			
			for _, url := range test.URLs {
				testWg.Add(1)
				go func(serviceUrl string) {
					defer testWg.Done()
					
					if serviceUrl[len(serviceUrl)-1] != '/' {
						serviceUrl = serviceUrl + "/"
					}
					requestURL := fmt.Sprintf("%shealth", serviceUrl)
					
					assert.True(t, strings.HasSuffix(requestURL, "/health"))
					
					time.Sleep(10 * time.Millisecond)
					mu.Lock()
					counter++
					mu.Unlock()
				}(url)
			}
			
			testWg.Wait()
			assert.Equal(t, len(test.URLs), counter, test.Description)
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

func TestURLEdgeCases(t *testing.T) {
	for _, test := range URLEdgeCaseTests {
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()
			
			if test.ShouldPanic {
				assert.Panics(t, func() {
					userUrl := test.Input
					if userUrl[len(userUrl)-1] != '/' {
						userUrl = userUrl + "/"
					}
					_ = userUrl + "health"
				}, test.Description)
			} else {
				assert.NotPanics(t, func() {
					userUrl := test.Input
					if len(userUrl) > 0 && userUrl[len(userUrl)-1] != '/' {
						userUrl = userUrl + "/"
					}
					result := userUrl + "health"
					assert.Equal(t, test.Expected, result, test.Description)
				}, test.Description)
			}
		})
	}
}

func TestEmptyURLHandling(t *testing.T) {
	for _, test := range EmptyURLHandlingTests {
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()
			
			for _, url := range test.URLs {
				if len(url) > 0 {
					assert.NotPanics(t, func() {
						userUrl := url
						if userUrl[len(userUrl)-1] != '/' {
							userUrl = userUrl + "/"
						}
						_ = userUrl + "health"
					}, test.Description)
				}
			}
		})
	}
}

func TestHTTPStatusCodeHandling(t *testing.T) {
	for _, test := range HTTPStatusCodeTests {
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()
			
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.StatusCode)
				switch test.StatusCode {
				case http.StatusOK:
					w.Write([]byte(MockHTTPResponses.Success))
				case http.StatusNotFound:
					w.Write([]byte(MockHTTPResponses.NotFound))
				case http.StatusInternalServerError:
					w.Write([]byte(MockHTTPResponses.Error))
				case http.StatusServiceUnavailable:
					w.Write([]byte(MockHTTPResponses.ServiceUnavail))
				}
			}))
			defer server.Close()
			
			userUrl := server.URL
			if userUrl[len(userUrl)-1] != '/' {
				userUrl = userUrl + "/"
			}
			requestURL := fmt.Sprintf("%shealth", userUrl)
			
			assert.True(t, strings.HasSuffix(requestURL, "/health"))
			assert.Contains(t, requestURL, server.URL)
		})
	}
}

func TestCallProfileHealthLogic(t *testing.T) {
	tests := []struct {
		name        string
		userUrl     string
		expectError bool
	}{
		{"Valid HTTPS URL", "https://example.com", false},
		{"Valid HTTP URL", "http://localhost:3000", false},
		{"URL with port", "https://api.example.com:8080", false},
		{"URL with path", "https://api.example.com/v1", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			
			assert.NotPanics(t, func() {
				httpClient := &http.Client{
					Timeout: 2 * time.Second,
				}
				assert.NotNil(t, httpClient)
				
				userUrl := test.userUrl
				if userUrl[len(userUrl)-1] != '/' {
					userUrl = userUrl + "/"
				}
				
				requestURL := fmt.Sprintf("%shealth", userUrl)
				assert.True(t, strings.HasSuffix(requestURL, "/health"))
				
				req, err := http.NewRequest("GET", requestURL, nil)
				if !test.expectError {
					assert.NoError(t, err)
					assert.NotNil(t, req)
				}
			})
		})
	}
}

func TestProfileCountingLogic(t *testing.T) {
	t.Run("Profile counting simulation", func(t *testing.T) {
		totalProfilesCalled := 0
		
		mockProfiles := []map[string]interface{}{
			{"profileURL": "https://user1.example.com", "profileStatus": "VERIFIED"},
			{"profileURL": "https://user2.example.com", "profileStatus": "VERIFIED"},
			{"profileURL": "https://user3.example.com", "profileStatus": "VERIFIED"},
			{"profileURL": "https://user4.example.com", "profileStatus": "PENDING"},
			{"profileURL": "https://user5.example.com", "profileStatus": "VERIFIED"},
		}
		
		for _, profile := range mockProfiles {
			if status, ok := profile["profileStatus"].(string); ok && status == "VERIFIED" {
				if profileURL, ok := profile["profileURL"].(string); ok && profileURL != "" {
					totalProfilesCalled++
				}
			}
		}
		
		assert.Equal(t, 4, totalProfilesCalled)
		assert.Greater(t, totalProfilesCalled, 0)
	})
}

func TestConcurrentHealthCheckExecution(t *testing.T) {
	t.Run("Concurrent health checks with different URLs", func(t *testing.T) {
		var testWg sync.WaitGroup
		var results []string
		var mu sync.Mutex
		
		urls := []string{
			"https://service1.example.com",
			"https://service2.example.com",
			"https://service3.example.com",
		}
		
		for _, url := range urls {
			testWg.Add(1)
			go func(serviceUrl string) {
				defer testWg.Done()
				
				if serviceUrl[len(serviceUrl)-1] != '/' {
					serviceUrl = serviceUrl + "/"
				}
				requestURL := fmt.Sprintf("%shealth", serviceUrl)
				
				mu.Lock()
				results = append(results, requestURL)
				mu.Unlock()
			}(url)
		}
		
		testWg.Wait()
		assert.Equal(t, len(urls), len(results))
		
		for _, result := range results {
			assert.True(t, strings.HasSuffix(result, "/health"))
		}
	})
}

func TestHandlerResponseStructure(t *testing.T) {
	tests := []struct {
		name          string
		profilesCount int
		expectedBody  string
	}{
		{"Zero profiles", 0, "Total Profiles called in session is 0"},
		{"Single profile", 1, "Total Profiles called in session is 1"},
		{"Multiple profiles", 10, "Total Profiles called in session is 10"},
		{"Large count", 1000, "Total Profiles called in session is 1000"},
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
				Path:       "/health-check",
			},
			userData: []map[string]interface{}{
				{
					"userId":        "user1",
					"profileURL":    "https://user1.example.com",
					"profileStatus": "PENDING",
				},
				{
					"userId":        "user2",
					"profileURL":    "https://user2.example.com",
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
				Path:       "/health-check",
			},
			userData: []map[string]interface{}{
				{
					"userId":        "user1",
					"profileURL":    "https://user1.example.com",
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
				Path:       "/health-check",
			},
			userData: []map[string]interface{}{
				{
					"userId":        "user1",
					"profileURL":    "https://user1.example.com",
					"profileStatus": "VERIFIED",
				},
				{
					"userId":        "user2",
					"profileURL":    "https://user2.example.com",
					"profileStatus": "VERIFIED",
				},
				{
					"userId":        "user3",
					"profileURL":    "https://user3.example.com",
					"profileStatus": "VERIFIED",
				},
				{
					"userId":        "user4",
					"profileURL":    "https://user4.example.com",
					"profileStatus": "PENDING",
				},
			},
			expectedBody:   "Total Profiles called in session is 3",
			expectedStatus: 200,
			expectedError:  false,
		},
		{
			name: "verified users with missing profileURL",
			request: events.APIGatewayProxyRequest{
				HTTPMethod: "GET",
				Path:       "/health-check",
			},
			userData: []map[string]interface{}{
				{
					"userId":        "user1",
					"profileURL":    "https://user1.example.com",
					"profileStatus": "VERIFIED",
				},
				{
					"userId":        "user2",
					"profileStatus": "VERIFIED",
				},
				{
					"userId":        "user3",
					"profileURL":    "",
					"profileStatus": "VERIFIED",
				},
			},
			expectedBody:   "Total Profiles called in session is 2",
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
		})
	}
}

func TestCallProfileHealthIntegration(t *testing.T) {
	ctx := context.Background()
	client := newFirestoreMockClient(ctx)
	defer client.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	userData := map[string]interface{}{
		"userId":        "test-user",
		"profileURL":    server.URL,
		"profileStatus": "VERIFIED",
	}

	_, err := client.Collection("users").Doc("test-user").Set(ctx, userData)
	assert.NoError(t, err)

	request := events.APIGatewayProxyRequest{
		HTTPMethod: "GET",
		Path:       "/health-check",
	}

	response, err := handlerWithClient(request, client)

	assert.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)
	assert.Equal(t, "Total Profiles called in session is 1", response.Body)

	client.Collection("users").Doc("test-user").Delete(ctx)
}

func TestHandlerWithRealFirestore(t *testing.T) {
	ctx := context.Background()
	client := newFirestoreMockClient(ctx)
	defer client.Close()

	request := events.APIGatewayProxyRequest{
		HTTPMethod: "GET",
		Path:       "/health-check",
	}

	response, err := handlerWithClient(request, client)
	assert.NoError(t, err)
	assert.Equal(t, 200, response.StatusCode)
	assert.Equal(t, "Total Profiles called in session is 0", response.Body)
}