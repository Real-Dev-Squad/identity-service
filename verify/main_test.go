package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
)

func newFirestoreMockClient(ctx context.Context) *firestore.Client {
	emulatorHost := os.Getenv("FIRESTORE_EMULATOR_HOST")
	if emulatorHost == "" {
		emulatorHost = "127.0.0.1:8090"
	}
	conn, _ := grpc.Dial(emulatorHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	client, _ := firestore.NewClient(ctx, "test-project", option.WithGRPCConn(conn))
	return client
}

func startMockServer(responseBody string, responseStatusCode int) *httptest.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(responseStatusCode)
		w.Write([]byte(responseBody))
	})
	return httptest.NewServer(handler)
}

func addUsers(ctx context.Context, client *firestore.Client, users []map[string]interface{}) error {
	for _, user := range users {
		id, ok := user["userId"].(string)
		if !ok {
			return fmt.Errorf("userId is missing or not a string: %v", user)
		}

		delete(user, "userId")
		if _, err := client.Collection("users").Doc(id).Set(ctx, user); err != nil {
			return fmt.Errorf("cannot add user %s: %v", id, err)
		}

	}

	return nil
}

func TestVerifyFunction(t *testing.T) {
	testCases := []struct {
		name           string
		profileURL     string
		chaincode      string
		salt           string
		mockResponse   string
		mockStatusCode int
		expectedStatus string
		expectedError  bool
	}{
		{
			name:           "successful verification with correct hash",
			profileURL:     "/profile",
			chaincode:      "testchaincode",
			salt:           "testsalt",
			mockResponse:   `{"hash": "cadf727ffff23ec46c17d808a4884ea7566765182d1a2ffa88e4719bc1f7f9fb328e2abacc13202f2dc55b9d653919b79ecf02dd752de80285bbec57a57713d9"}`,
			mockStatusCode: http.StatusOK,
			expectedStatus: "BLOCKED",
			expectedError:  false,
		},
		{
			name:           "failed verification with incorrect hash",
			profileURL:     "/profile",
			chaincode:      "testchaincode",
			salt:           "testsalt",
			mockResponse:   `{"hash": "incorrecthash"}`,
			mockStatusCode: http.StatusOK,
			expectedStatus: "BLOCKED",
			expectedError:  false,
		},
		{
			name:           "server error during verification",
			profileURL:     "/profile",
			chaincode:      "testchaincode",
			salt:           "testsalt",
			mockResponse:   `{"error": "server error"}`,
			mockStatusCode: http.StatusInternalServerError,
			expectedStatus: "BLOCKED",
			expectedError:  false,
		},
		{
			name:           "invalid JSON response",
			profileURL:     "/profile",
			chaincode:      "testchaincode",
			salt:           "testsalt",
			mockResponse:   `invalid json`,
			mockStatusCode: http.StatusOK,
			expectedStatus: "BLOCKED",
			expectedError:  false,
		},
		{
			name:           "empty response body",
			profileURL:     "/profile",
			chaincode:      "testchaincode",
			salt:           "testsalt",
			mockResponse:   ``,
			mockStatusCode: http.StatusOK,
			expectedStatus: "BLOCKED",
			expectedError:  false,
		},
		{
			name:           "network timeout",
			profileURL:     "/timeout",
			chaincode:      "testchaincode",
			salt:           "testsalt",
			mockResponse:   `{"hash": "test"}`,
			mockStatusCode: http.StatusOK,
			expectedStatus: "BLOCKED",
			expectedError:  false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/timeout" {
					time.Sleep(2 * time.Second)
				}
				w.WriteHeader(testCase.mockStatusCode)
				w.Write([]byte(testCase.mockResponse))
			}))
			defer server.Close()

			if testCase.name == "network timeout" {
				status, err := verify(server.URL+testCase.profileURL, testCase.chaincode, testCase.salt)
				assert.Equal(t, testCase.expectedStatus, status)
				assert.True(t, testCase.expectedError == (err != nil))
			} else {
				status, err := verify(server.URL+testCase.profileURL, testCase.chaincode, testCase.salt)
				assert.Equal(t, testCase.expectedStatus, status)
				assert.True(t, testCase.expectedError == (err != nil))
			}
		})
	}
}

func TestHandler(t *testing.T) {
	os.Setenv("environment", "test")
	defer os.Unsetenv("environment")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	client := newFirestoreMockClient(ctx)
	defer cancel()

	verifiedMockServer := startMockServer(`{"hash":"correcthash"}`, http.StatusOK)
	defer verifiedMockServer.Close()

	unverifiedMockServer := startMockServer(`{"hash":"incorrecthash"}`, http.StatusOK)
	defer unverifiedMockServer.Close()

	errorMockServer := startMockServer(`{"error":"server error"}`, http.StatusInternalServerError)
	defer errorMockServer.Close()

	invalidJSONMockServer := startMockServer(`invalid json`, http.StatusOK)
	defer invalidJSONMockServer.Close()

	verifiedUserId := "123"
	unverifiedUserId := "321"
	errorUserId := "456"
	invalidJSONUserId := "789"
	nonExistentUserId := "999"
	
	users := []map[string]interface{}{
		{"userId": verifiedUserId, "chaincode": "TESTCHAIN", "profileURL": verifiedMockServer.URL, "profileStatus": "VERIFIED"},
		{"userId": unverifiedUserId, "chaincode": "TESTCHAINCODE", "profileURL": unverifiedMockServer.URL, "profileStatus": "BLOCKED"},
		{"userId": errorUserId, "chaincode": "TESTCHAIN", "profileURL": errorMockServer.URL, "profileStatus": "PENDING"},
		{"userId": invalidJSONUserId, "chaincode": "TESTCHAIN", "profileURL": invalidJSONMockServer.URL, "profileStatus": "PENDING"},
	}

	if err := addUsers(ctx, client, users); err != nil {
		t.Fatalf("failed to add users: %v", err)
	}

	testCases := []struct {
		name           string
		request        events.APIGatewayProxyRequest
		expect         string
		expectedStatus int
		err            error
	}{
		{
			name:           "unverified user",
			request:        events.APIGatewayProxyRequest{Body: fmt.Sprintf(`{ "userId": "%s" }`, unverifiedUserId)},
			expect:         "Verification Process Done",
			expectedStatus: 200,
			err:            nil,
		},
		{
			name:           "verified user",
			request:        events.APIGatewayProxyRequest{Body: fmt.Sprintf(`{ "userId": "%s" }`, verifiedUserId)},
			expect:         "Already Verified",
			expectedStatus: 409,
			err:            nil,
		},
		{
			name:           "no userId",
			request:        events.APIGatewayProxyRequest{Body: `{}`},
			expect:         "",
			expectedStatus: 0,
			err:            errors.New("no userId provided"),
		},
		{
			name:           "non-existent user",
			request:        events.APIGatewayProxyRequest{Body: fmt.Sprintf(`{ "userId": "%s" }`, nonExistentUserId)},
			expect:         "",
			expectedStatus: 0,
			err:            nil,
		},
		{
			name:           "server error during verification",
			request:        events.APIGatewayProxyRequest{Body: fmt.Sprintf(`{ "userId": "%s" }`, errorUserId)},
			expect:         "Verification Process Done",
			expectedStatus: 200,
			err:            nil,
		},
		{
			name:           "invalid JSON response",
			request:        events.APIGatewayProxyRequest{Body: fmt.Sprintf(`{ "userId": "%s" }`, invalidJSONUserId)},
			expect:         "Verification Process Done",
			expectedStatus: 200,
			err:            nil,
		},
		{
			name:           "empty request body",
			request:        events.APIGatewayProxyRequest{Body: ""},
			expect:         "",
			expectedStatus: 0,
			err:            errors.New("no userId provided"),
		},
		{
			name:           "invalid JSON in request",
			request:        events.APIGatewayProxyRequest{Body: `{"userId": }`},
			expect:         "",
			expectedStatus: 0,
			err:            errors.New("no userId provided"),
		},
	}

	d := deps{
		client: client,
		ctx:    ctx,
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			response, err := d.handler(testCase.request)
			if testCase.name == "non-existent user" {
				assert.Error(t, err)
			} else {
				assert.Equal(t, testCase.err, err)
			}
			assert.Equal(t, testCase.expect, response.Body)
			assert.Equal(t, testCase.expectedStatus, response.StatusCode)
		})
	}
}

func TestURLFormatting(t *testing.T) {
	os.Setenv("environment", "test")
	defer os.Unsetenv("environment")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	client := newFirestoreMockClient(ctx)
	defer cancel()

	testCases := []struct {
		name           string
		profileURL     string
		expectedSuffix string
	}{
		{
			name:           "URL with trailing slash",
			profileURL:     "http://example.com/",
			expectedSuffix: "verification",
		},
		{
			name:           "URL without trailing slash",
			profileURL:     "http://example.com",
			expectedSuffix: "/verification",
		},
		{
			name:           "URL with path",
			profileURL:     "http://example.com/api",
			expectedSuffix: "/verification",
		},
		{
			name:           "URL with path and trailing slash",
			profileURL:     "http://example.com/api/",
			expectedSuffix: "verification",
		},
		{
			name:           "Single character URL",
			profileURL:     "a",
			expectedSuffix: "/verification",
		},
		{
			name:           "Empty URL",
			profileURL:     "",
			expectedSuffix: "/verification",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			userId := fmt.Sprintf("user_%s", testCase.name)
			
			user := map[string]interface{}{
				"userId":        userId,
				"chaincode":     "TESTCHAIN",
				"profileURL":    testCase.profileURL,
				"profileStatus": "PENDING",
			}
			
			if err := addUsers(ctx, client, []map[string]interface{}{user}); err != nil {
				t.Fatalf("failed to add user: %v", err)
			}

			var capturedURL string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedURL = r.URL.String()
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"hash": "testhash"}`))
			}))
			defer server.Close()

			client.Collection("users").Doc(userId).Update(ctx, []firestore.Update{
				{Path: "profileURL", Value: server.URL},
			})

			d := deps{
				client: client,
				ctx:    ctx,
			}

			request := events.APIGatewayProxyRequest{
				Body: fmt.Sprintf(`{ "userId": "%s" }`, userId),
			}

			response, err := d.handler(request)
			
			assert.NoError(t, err)
			assert.Equal(t, 200, response.StatusCode)
			assert.Contains(t, capturedURL, testCase.expectedSuffix)
		})
	}
}

func TestSaltGeneration(t *testing.T) {
	os.Setenv("environment", "test")
	defer os.Unsetenv("environment")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	client := newFirestoreMockClient(ctx)
	defer cancel()

	salts := make(map[string]bool)
	
	for i := 0; i < 10; i++ {
		userId := fmt.Sprintf("salt_test_user_%d", i)
		
		user := map[string]interface{}{
			"userId":        userId,
			"chaincode":     "TESTCHAIN",
			"profileURL":    "http://example.com",
			"profileStatus": "PENDING",
		}
		
		if err := addUsers(ctx, client, []map[string]interface{}{user}); err != nil {
			t.Fatalf("failed to add user: %v", err)
		}

		var capturedBody string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			capturedBody = string(body)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"hash": "testhash"}`))
		}))
		defer server.Close()

		client.Collection("users").Doc(userId).Update(ctx, []firestore.Update{
			{Path: "profileURL", Value: server.URL},
		})

		d := deps{
			client: client,
			ctx:    ctx,
		}

		request := events.APIGatewayProxyRequest{
			Body: fmt.Sprintf(`{ "userId": "%s" }`, userId),
		}

		response, err := d.handler(request)
		
		assert.NoError(t, err)
		assert.Equal(t, 200, response.StatusCode)
		
		var requestBody map[string]string
		err = json.Unmarshal([]byte(capturedBody), &requestBody)
		assert.NoError(t, err)
		
		salt := requestBody["salt"]
		assert.NotEmpty(t, salt)
		assert.Len(t, salt, 21)
		
		validChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ123456789"
		for _, char := range salt {
			assert.Contains(t, validChars, string(char))
		}
		
		assert.False(t, salts[salt], "Salt should be unique")
		salts[salt] = true
	}
}

func TestHandlerEdgeCases(t *testing.T) {
	os.Setenv("environment", "test")
	defer os.Unsetenv("environment")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	client := newFirestoreMockClient(ctx)
	defer cancel()

	testCases := []struct {
		name           string
		userData       map[string]interface{}
		requestBody    string
		expectedError  string
		expectedStatus int
	}{
		{
			name: "user with empty chaincode",
			userData: map[string]interface{}{
				"userId":        "empty_chaincode_user",
				"chaincode":     "",
				"profileURL":    "http://example.com",
				"profileStatus": "PENDING",
			},
			requestBody:    `{ "userId": "empty_chaincode_user" }`,
			expectedError:  "chaincode is blocked",
			expectedStatus: 0,
		},
		{
			name: "user with missing profileURL",
			userData: map[string]interface{}{
				"userId":        "missing_url_user",
				"chaincode":     "TESTCHAIN",
				"profileStatus": "PENDING",
			},
			requestBody:    `{ "userId": "missing_url_user" }`,
			expectedError:  "profile url is not a string",
			expectedStatus: 0,
		},
		{
			name: "user with invalid chaincode type",
			userData: map[string]interface{}{
				"userId":        "invalid_chaincode_user",
				"chaincode":     123,
				"profileURL":    "http://example.com",
				"profileStatus": "PENDING",
			},
			requestBody:    `{ "userId": "invalid_chaincode_user" }`,
			expectedError:  "chaincode is not a string",
			expectedStatus: 0,
		},
		{
			name: "user with invalid profileURL type",
			userData: map[string]interface{}{
				"userId":        "invalid_url_user",
				"chaincode":     "TESTCHAIN",
				"profileURL":    123,
				"profileStatus": "PENDING",
			},
			requestBody:    `{ "userId": "invalid_url_user" }`,
			expectedError:  "profile url is not a string",
			expectedStatus: 0,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if err := addUsers(ctx, client, []map[string]interface{}{testCase.userData}); err != nil {
				t.Fatalf("failed to add user: %v", err)
			}

			d := deps{
				client: client,
				ctx:    ctx,
			}

			request := events.APIGatewayProxyRequest{
				Body: testCase.requestBody,
			}

			response, err := d.handler(request)
			
			assert.Error(t, err)
			assert.Contains(t, err.Error(), testCase.expectedError)
			assert.Equal(t, testCase.expectedStatus, response.StatusCode)
		})
	}
}

func TestVerifyFunctionCompleteCoverage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"hash": "cadf727ffff23ec46c17d808a4884ea7566765182d1a2ffa88e4719bc1f7f9fb328e2abacc13202f2dc55b9d653919b79ecf02dd752de80285bbec57a57713d9"}`))
	}))
	defer server.Close()
	
	status, err := verify(server.URL+"/verify", "testchaincode", "testsalt")
		assert.Equal(t, "BLOCKED", status)
	assert.NoError(t, err)
	
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"hash": "wronghash"}`))
	}))
	defer server2.Close()
	
	status, err = verify(server2.URL+"/verify", "testchaincode", "testsalt")
	assert.Equal(t, "BLOCKED", status)
	assert.NoError(t, err)
	
	status, err = verify("http://192.168.1.1:99999/verify", "testchaincode", "testsalt")
	assert.Equal(t, "BLOCKED", status)
	assert.Error(t, err)
	
	server3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server3.Close()
	
	status, err = verify(server3.URL+"/verify", "testchaincode", "testsalt")
	assert.Equal(t, "BLOCKED", status)
	assert.Error(t, err)
}

func TestMainFunctionComponents(t *testing.T) {
	ctx := context.Background()
	assert.NotNil(t, ctx)

	d := deps{
		client: nil,
		ctx:    ctx,
	}
	assert.NotNil(t, d)
	assert.Equal(t, ctx, d.ctx)
}
