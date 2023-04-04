package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"testing"
	"time"
	"verify/utils/mocks"

	"cloud.google.com/go/firestore"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
)

func init() {
	Client = &mocks.MockClient{}
}

func newFirestoreMockClient(ctx context.Context) *firestore.Client {
	client, err := firestore.NewClient(ctx, "test")
	if err != nil {
		log.Fatalf("firebase.NewClient err: %v", err)
	}

	return client
}

func TestHandler(t *testing.T) {
	os.Setenv("environment", "test")
	defer os.Unsetenv("environment")

	mocks.PostFunc = func(url string, contentType string, body io.Reader) (*http.Response, error) {
		if url == "https://96phoonyw3.execute-api.us-east-2.amazonaws.com/Prod/verification" {
			return &http.Response{
				StatusCode: 200,
				Body:       http.NoBody,
			}, nil
		}
		if url == "https://test.com/verification" {
			return &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(bytes.NewBufferString(`{"hash": "$2a$12$ScGc2Q0t0rqqSJK1E2W/WuaRVAchaVWdUqb1hQi21cFTnOVvlIdry"}`)),
			}, nil
		}

		return &http.Response{
			StatusCode: 500,
		}, fmt.Errorf("custom error")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	client := newFirestoreMockClient(ctx)
	defer cancel()

	verifiedUserId := "123"
	client.Collection("users").Doc(verifiedUserId).Set(ctx, map[string]interface{}{
		"chaincode":     "abcdefgh",
		"profileURL":    "https://96phoonyw3.execute-api.us-east-2.amazonaws.com/Prod",
		"profileStatus": "VERIFIED",
	})
	unverifiedUserId := "321"
	client.Collection("users").Doc(unverifiedUserId).Set(ctx, map[string]interface{}{
		"chaincode":     "testchaincode",
		"profileURL":    "https://test.com",
		"profileStatus": "BLOCKED",
	})

	testCases := []struct {
		name    string
		request events.APIGatewayProxyRequest
		expect  string
		err     error
	}{
		{
			name:    "verified user",
			request: events.APIGatewayProxyRequest{Body: fmt.Sprintf(`{ "userId": "%s" }`, verifiedUserId)},
			expect:  "Already Verified",
			err:     nil,
		},
		{
			name:    "unverified user",
			request: events.APIGatewayProxyRequest{Body: fmt.Sprintf(`{ "userId": "%s" }`, unverifiedUserId)},
			expect:  "Verification Process Done",
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
			assert.IsType(t, testCase.err, err)
			assert.Equal(t, testCase.expect, response.Body)
		})
	}

}

func TestSetProfileStatus(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	client := newFirestoreMockClient(ctx)
	defer cancel()

	// When status is BLOCKED, expect to set chaincode to empty string
	ID := "1234"
	profileStatus := "BLOCKED"
	client.Collection("users").Doc(ID).Set(ctx, map[string]interface{}{
		"profileStatus": profileStatus,
	})
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

	// if status is NOT BLOCKED, expect to set profile status without errors
	ID = "abcd"
	profileStatus = "VERIFIED"
	client.Collection("users").Doc(ID).Set(ctx, map[string]interface{}{
		"profileStatus": profileStatus,
	})
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	client := newFirestoreMockClient(ctx)
	defer cancel()

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
	profileURL     string
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
			profileURL:     "http://test.com",
			chaincode:      "testchaincode",
			mockStatusCode: 200,
			mockResBody:    `{"hash": "$2a$12$ScGc2Q0t0rqqSJK1E2W/WuaRVAchaVWdUqb1hQi21cFTnOVvlIdry"}`,
			expectedStatus: "VERIFIED",
			expectedErr:    nil,
		},
		{
			name:           "BLOCKED",
			profileURL:     "http://test.com",
			chaincode:      "invalid",
			mockStatusCode: 403,
			mockResBody:    `{"hash": "abcdefghijklmnopqrstuvwxyz"}`,
			expectedStatus: "BLOCKED",
			expectedErr:    nil,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			mockResponse := &http.Response{
				StatusCode: testCase.mockStatusCode,
				Body:       ioutil.NopCloser(bytes.NewBufferString(testCase.mockResBody)),
				Header:     make(http.Header),
			}

			mocks.PostFunc = func(url string, contentType string, body io.Reader) (*http.Response, error) {
				if url != testCase.profileURL {
					return nil, fmt.Errorf("unknown profile URL: %s", url)
				}
				return mockResponse, nil
			}

			status, err := verify(testCase.profileURL, testCase.chaincode)

			assert.Equal(t, testCase.expectedStatus, status)
			assert.Equal(t, testCase.expectedErr, err)
		})
	}
}
