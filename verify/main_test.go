package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
)

func newFirestoreMockClient(ctx context.Context) *firestore.Client {
	client, err := firestore.NewClient(ctx, "test")
	if err != nil {
		log.Fatalf("firebase.NewClient err: %v", err)
	}

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

func TestHandler(t *testing.T) {
	os.Setenv("environment", "test")
	defer os.Unsetenv("environment")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	client := newFirestoreMockClient(ctx)
	defer cancel()

	// Mock servers for profile verification
	verifiedMockServer := startMockServer(`{"hash":"correcthash"}`, http.StatusOK)
	defer verifiedMockServer.Close()

	unverifiedMockServer := startMockServer(`{"hash":"incorrecthash"}`, http.StatusOK)
	defer unverifiedMockServer.Close()

	verifiedUserId := "123"
	unverifiedUserId := "321"
	users := []map[string]interface{}{
		{"userId": verifiedUserId, "chaincode": "TESTCHAIN", "profileURL": verifiedMockServer.URL, "profileStatus": "VERIFIED"},
		{"userId": unverifiedUserId, "chaincode": "TESTCHAINCODE", "profileURL": unverifiedMockServer.URL, "profileStatus": "BLOCKED"},
	}

	if err := addUsers(ctx, client, users); err != nil {
		t.Fatalf("failed to add users: %v", err)
	}

	testCases := []struct {
		name    string
		request events.APIGatewayProxyRequest
		expect  string
		err     error
	}{
		{
			name:    "unverified user",
			request: events.APIGatewayProxyRequest{Body: fmt.Sprintf(`{ "userId": "%s" }`, unverifiedUserId)},
			expect:  "Verification Process Done",
			err:     nil,
		},
		{
			name:    "verified user",
			request: events.APIGatewayProxyRequest{Body: fmt.Sprintf(`{ "userId": "%s" }`, verifiedUserId)},
			expect:  "Already Verified",
			err:     nil,
		},
		{
			name:    "no userId",
			request: events.APIGatewayProxyRequest{Body: `{}`},
			expect:  "",
			err:     errors.New("no userId provided"),
		},
	}

	d := deps{
		client: client,
		ctx:    ctx,
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			response, err := d.handler(testCase.request)
			assert.Equal(t, testCase.err, err)
			assert.Equal(t, testCase.expect, response.Body)
		})
	}
}
