package main

import (
	"identity-service/layer/utils"
	"os"

	"github.com/aws/aws-lambda-go/events"
)

var TestRequests = []struct {
	Name        string
	Request     events.APIGatewayProxyRequest
	ExpectedErr bool
	Description string
}{
	{
		Name: "BasicGETRequest",
		Request: events.APIGatewayProxyRequest{
			HTTPMethod: "GET",
			Path:       "/call-profiles",
		},
		ExpectedErr: true,
		Description: "Basic GET request should fail at Firestore initialization",
	},
	{
		Name: "BasicPOSTRequest",
		Request: events.APIGatewayProxyRequest{
			HTTPMethod: "POST",
			Path:       "/call-profiles",
		},
		ExpectedErr: true,
		Description: "Basic POST request should fail at Firestore initialization",
	},
	{
		Name: "RequestWithHeaders",
		Request: events.APIGatewayProxyRequest{
			HTTPMethod: "GET",
			Path:       "/call-profiles",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer token123",
			},
		},
		ExpectedErr: true,
		Description: "Request with headers should fail at Firestore initialization",
	},
}

var ProfileLambdaCallPayloadTests = []struct {
	Name        string
	UserId      string
	SessionId   string
	Description string
}{
	{
		Name:        "StandardPayload",
		UserId:      "user123",
		SessionId:   "session456",
		Description: "Standard payload with both fields",
	},
	{
		Name:        "LongIDs",
		UserId:      "very-long-user-id-with-special-characters-and-numbers-123456789",
		SessionId:   "very-long-session-id-with-special-characters-and-numbers-987654321",
		Description: "Payload with very long IDs",
	},
	{
		Name:        "SpecialCharacters",
		UserId:      "user@domain.com",
		SessionId:   "session-with-dashes-and_underscores",
		Description: "Payload with special characters",
	},
	{
		Name:        "EmptyUserId",
		UserId:      "",
		SessionId:   "session456",
		Description: "Payload with empty userId",
	},
	{
		Name:        "EmptySessionId",
		UserId:      "user123",
		SessionId:   "",
		Description: "Payload with empty sessionId",
	},
	{
		Name:        "BothEmpty",
		UserId:      "",
		SessionId:   "",
		Description: "Payload with both fields empty",
	},
	{
		Name:        "UUIDFormat",
		UserId:      "550e8400-e29b-41d4-a716-446655440000",
		SessionId:   "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
		Description: "Payload with UUID format IDs",
	},
}

var InvokeProfileLambdaErrorTests = []struct {
	Name        string
	EnvVarValue string
	Payload     utils.ProfileLambdaCallPayload
	ExpectedErr string
	Description string
}{
	{
		Name:        "MissingEnvVar",
		EnvVarValue: "",
		Payload: utils.ProfileLambdaCallPayload{
			UserId:    "user123",
			SessionID: "session456",
		},
		ExpectedErr: "profileFunctionLambdaName is not set",
		Description: "Missing environment variable should return error",
	},
	{
		Name:        "EmptyEnvVar",
		EnvVarValue: "",
		Payload: utils.ProfileLambdaCallPayload{
			UserId:    "user123",
			SessionID: "session456",
		},
		ExpectedErr: "profileFunctionLambdaName is not set",
		Description: "Empty environment variable should return error",
	},
}

var APIGatewayProxyRequestWrapperTests = []struct {
	Name        string
	Body        string
	Description string
}{
	{
		Name:        "SimpleJSONBody",
		Body:        `{"userId": "user123", "sessionId": "session456"}`,
		Description: "Simple JSON body",
	},
	{
		Name:        "EmptyBody",
		Body:        "",
		Description: "Empty body",
	},
	{
		Name:        "ComplexJSONBody",
		Body:        `{"userId": "user123", "sessionId": "session456", "metadata": {"source": "test", "timestamp": 1234567890}}`,
		Description: "Complex JSON body with nested objects",
	},
	{
		Name:        "LargeJSONBody",
		Body:        `{"userId": "user123", "sessionId": "session456", "data": "` + generateLargeString(1000) + `"}`,
		Description: "Large JSON body",
	},
}

var ResponseFormatTests = []struct {
	Name           string
	ProfilesCount  int
	ExpectedFormat string
	Description    string
}{
	{
		Name:           "ZeroProfiles",
		ProfilesCount:  0,
		ExpectedFormat: "Total Profiles called in session is 0",
		Description:    "Response format for zero profiles",
	},
	{
		Name:           "OneProfile",
		ProfilesCount:  1,
		ExpectedFormat: "Total Profiles called in session is 1",
		Description:    "Response format for single profile",
	},
	{
		Name:           "MultipleProfiles",
		ProfilesCount:  5,
		ExpectedFormat: "Total Profiles called in session is 5",
		Description:    "Response format for multiple profiles",
	},
	{
		Name:           "LargeNumberOfProfiles",
		ProfilesCount:  100,
		ExpectedFormat: "Total Profiles called in session is 100",
		Description:    "Response format for large number of profiles",
	},
	{
		Name:           "VeryLargeNumber",
		ProfilesCount:  999999,
		ExpectedFormat: "Total Profiles called in session is 999999",
		Description:    "Response format for very large number",
	},
}

var SessionIdGenerationTests = []struct {
	Name        string
	Count       int
	Description string
}{
	{
		Name:        "SmallBatch",
		Count:       5,
		Description: "Generate small batch of session IDs",
	},
	{
		Name:        "MediumBatch",
		Count:       50,
		Description: "Generate medium batch of session IDs",
	},
	{
		Name:        "LargeBatch",
		Count:       100,
		Description: "Generate large batch of session IDs",
	},
}

var EnvironmentVariableTests = []struct {
	Name         string
	OriginalEnv  string
	TestEnv      string
	ShouldRestore bool
	Description  string
}{
	{
		Name:         "SetValidEnv",
		TestEnv:      "test-lambda-function-name",
		ShouldRestore: true,
		Description:  "Set valid environment variable",
	},
	{
		Name:         "SetEmptyEnv",
		TestEnv:      "",
		ShouldRestore: true,
		Description:  "Set empty environment variable",
	},
	{
		Name:         "UnsetEnv",
		TestEnv:      "",
		ShouldRestore: true,
		Description:  "Unset environment variable",
	},
}

func generateLargeString(size int) string {
	result := make([]byte, size)
	for i := range result {
		result[i] = 'a' + byte(i%26)
	}
	return string(result)
}

func SaveEnvVar(key string) string {
	return os.Getenv(key)
}

func RestoreEnvVar(key, value string) {
	if value != "" {
		os.Setenv(key, value)
	} else {
		os.Unsetenv(key)
	}
}

var ConcurrentTestData = struct {
	UserIds    []string
	SessionIds []string
	URLs       []string
}{
	UserIds: []string{
		"user1", "user2", "user3", "user4", "user5",
		"user6", "user7", "user8", "user9", "user10",
	},
	SessionIds: []string{
		"session1", "session2", "session3", "session4", "session5",
		"session6", "session7", "session8", "session9", "session10",
	},
	URLs: []string{
		"https://service1.example.com",
		"https://service2.example.com",
		"https://service3.example.com",
		"https://service4.example.com",
		"https://service5.example.com",
	},
}
