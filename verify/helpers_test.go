package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
)

var (
	ctx    context.Context
	cancel context.CancelFunc
	client *firestore.Client
)

func TestMain(m *testing.M) {
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	client = newFirestoreMockClient(ctx)

	code := m.Run()

	cancel()
	client.Close()

	os.Exit(code)
}

func TestSetProfileStatus(t *testing.T) {
	// When status is BLOCKED, expect to set chaincode to empty string
	ID := "1234"
	profileStatus := "BLOCKED"
	err := setProfileStatus(client, ctx, ID, profileStatus)
	if err != nil {
		t.Errorf("setProfileStatus returned an error: %v", err)
	}

	userDoc, err := client.Collection("users").Doc(ID).Get(ctx)
	if err != nil {
		t.Errorf("Unable to fetch user document: %v", err)
	}

	assert.Equal(t, "", userDoc.Data()["chaincode"])
	assert.Equal(t, profileStatus, userDoc.Data()["profileStatus"])

	// if status is VERIFIED / PENDING, expect to set profile status without errors
	ID = "abcd"
	profileStatus = "VERIFIED"
	err = setProfileStatus(client, ctx, ID, profileStatus)
	if err != nil {
		t.Errorf("setProfileStatus returned an error: %v", err)
	}

	userDoc, err = client.Collection("users").Doc(ID).Get(ctx)
	if err != nil {
		t.Errorf("Unable to fetch user document: %v", err)
	}

	assert.Equal(t, profileStatus, userDoc.Data()["profileStatus"])
}

func TestGetUserData(t *testing.T) {
	testCases := []struct {
		name          string
		userId        string
		profileURL    interface{}
		profileStatus interface{}
		chaincode     interface{}
		expectedErr   error
	}{
		{
			name:          "success",
			userId:        "1",
			profileURL:    "http://test.com",
			profileStatus: "VERIFIED",
			chaincode:     "abc123",
			expectedErr:   nil,
		},
		{
			name:          "missing profile url",
			userId:        "2",
			profileURL:    nil,
			profileStatus: "BLOCKED",
			chaincode:     "abc123",
			expectedErr:   errors.New("profile url is not a string"),
		},
		{
			name:          "missing chaincode",
			userId:        "3",
			profileURL:    "http://test.com",
			profileStatus: "BLOCKED",
			chaincode:     "",
			expectedErr:   errors.New("chaincode is blocked"),
		},
		{
			name:          "invalid chaincode",
			userId:        "4",
			profileURL:    "http://test.com",
			profileStatus: "BLOCKED",
			chaincode:     123,
			expectedErr:   errors.New("chaincode is not a string"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			client.Collection("users").Doc(testCase.userId).Set(ctx, map[string]interface{}{
				"chaincode":     testCase.chaincode,
				"profileURL":    testCase.profileURL,
				"profileStatus": testCase.profileStatus,
			})

			profileURL, profileStatus, chaincode, err := getUserData(client, ctx, testCase.userId)

			assert.Equal(t, testCase.expectedErr, err)
			if testCase.expectedErr == nil {
				assert.Equal(t, testCase.profileURL.(string), profileURL)
				assert.Equal(t, testCase.profileStatus.(string), profileStatus)
				assert.Equal(t, testCase.chaincode.(string), chaincode)
			}
		})
	}
}

func TestGetUserIdFromBody(t *testing.T) {
	// valid request body
	body := []byte(`{"userId": "123"}`)
	expected := "123"

	actual := getUserIdFromBody(body)

	if actual != expected {
		t.Errorf("getUserIdFromBody returned %v, expected %v", actual, expected)
	}

	// empty request body
	body = []byte(``)
	expected = ""

	actual = getUserIdFromBody(body)

	if actual != expected {
		t.Errorf("getUserIdFromBody returned %v, expected %v", actual, expected)
	}

	// invalid request body
	body = []byte(`{"invalidProperty": ""}`)
	expected = ""

	actual = getUserIdFromBody(body)

	if actual != expected {
		t.Errorf("getUserIdFromBody returned %v, expected %v", actual, expected)
	}
}

type testVerifyData struct {
	name           string
	chaincode      string
	mockStatusCode int
	mockResBody    string
	expectedStatus string
	expectedErr    error
}

func TestVerify(t *testing.T) {
	testCases := []testVerifyData{
		{
			name:           "VERIFIED",
			chaincode:      "testchaincode",
			mockStatusCode: http.StatusOK,
			mockResBody:    `{"hash": "$2a$12$ScGc2Q0t0rqqSJK1E2W/WuaRVAchaVWdUqb1hQi21cFTnOVvlIdry"}`,
			expectedStatus: "VERIFIED",
			expectedErr:    nil,
		},
		{
			name:           "BLOCKED",
			chaincode:      "invalid",
			mockStatusCode: http.StatusForbidden,
			mockResBody:    `{"hash": "abcdefghijklmnopqrstuvwxyz"}`,
			expectedStatus: "BLOCKED",
			expectedErr:    nil,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(testCase.mockStatusCode)
				w.Write([]byte(testCase.mockResBody))
			}))

			status, err := verify(server.URL+"/", testCase.chaincode)

			assert.Equal(t, testCase.expectedStatus, status)
			assert.Equal(t, testCase.expectedErr, err)

			defer server.Close()
		})
	}
}
