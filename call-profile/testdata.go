package main

import (
	"identity-service/layer/utils"

	"github.com/aws/aws-lambda-go/events"
)

	var TestRequests = []struct {
	Name        string
	Request     events.APIGatewayProxyRequest
	ExpectedErr bool
	Description string
}{
	{
		Name: "EmptyBody",
		Request: events.APIGatewayProxyRequest{
			Body: "",
		},
		ExpectedErr: true,
		Description: "Empty request body should fail at Firestore initialization",
	},
	{
		Name: "InvalidJSON",
		Request: events.APIGatewayProxyRequest{
			Body: "invalid json",
		},
		ExpectedErr: true,
		Description: "Invalid JSON should fail at Firestore initialization",
	},
	{
		Name: "ValidUserIdOnly",
		Request: events.APIGatewayProxyRequest{
			Body: `{"userId": "test-user-123"}`,
		},
		ExpectedErr: true,
		Description: "Valid userId should fail at Firestore initialization",
	},
	{
		Name: "ValidUserIdAndSession",
		Request: events.APIGatewayProxyRequest{
			Body: `{"userId": "test-user-123", "sessionId": "session-456"}`,
		},
		ExpectedErr: true,
		Description: "Valid userId and sessionId should fail at Firestore initialization",
	},
}

var GetDataFromBodyTests = []struct {
	Name              string
	Body              string
	ExpectedUserId    string
	ExpectedSessionId string
	Description       string
}{
	{
		Name:              "ValidBothFields",
		Body:              `{"userId": "user123", "sessionId": "session456"}`,
		ExpectedUserId:    "user123",
		ExpectedSessionId: "session456",
		Description:       "Valid JSON with both userId and sessionId",
	},
	{
		Name:              "OnlyUserId",
		Body:              `{"userId": "user123"}`,
		ExpectedUserId:    "user123",
		ExpectedSessionId: "",
		Description:       "Valid JSON with only userId",
	},
	{
		Name:              "OnlySessionId",
		Body:              `{"sessionId": "session456"}`,
		ExpectedUserId:    "",
		ExpectedSessionId: "session456",
		Description:       "Valid JSON with only sessionId",
	},
	{
		Name:              "EmptyJSON",
		Body:              `{}`,
		ExpectedUserId:    "",
		ExpectedSessionId: "",
		Description:       "Empty JSON object",
	},
	{
		Name:              "InvalidJSON",
		Body:              `invalid json`,
		ExpectedUserId:    "",
		ExpectedSessionId: "",
		Description:       "Invalid JSON should return empty strings",
	},
	{
		Name:              "EmptyString",
		Body:              "",
		ExpectedUserId:    "",
		ExpectedSessionId: "",
		Description:       "Empty string should return empty strings",
	},
	{
		Name:              "ExtraFields",
		Body:              `{"userId": "user123", "sessionId": "session456", "extra": "field", "timestamp": 1234567890}`,
		ExpectedUserId:    "user123",
		ExpectedSessionId: "session456",
		Description:       "JSON with extra fields should extract only userId and sessionId",
	},
}

var URLFormattingTests = []struct {
	Name        string
	Input       string
	Expected    string
	Description string
}{
	{
		Name:        "WithTrailingSlash",
		Input:       "https://example.com/",
		Expected:    "https://example.com/health",
		Description: "URL with trailing slash",
	},
	{
		Name:        "WithoutTrailingSlash",
		Input:       "https://example.com",
		Expected:    "https://example.com/health",
		Description: "URL without trailing slash",
	},
	{
		Name:        "LocalURL",
		Input:       "http://localhost:3000",
		Expected:    "http://localhost:3000/health",
		Description: "Local development URL",
	},
	{
		Name:        "URLWithPath",
		Input:       "https://api.example.com/v1",
		Expected:    "https://api.example.com/v1/health",
		Description: "URL with existing path",
	},
	{
		Name:        "URLWithPort",
		Input:       "https://service.example.com:8080",
		Expected:    "https://service.example.com:8080/health",
		Description: "URL with custom port",
	},
}

var ResValidationTests = []struct {
	Name        string
	Res         utils.Res
	IsValid     bool
	Description string
}{
	{
		Name: "ValidRes",
		Res: utils.Res{
			FirstName:   "John",
			LastName:    "Doe",
			Email:       "john.doe@example.com",
			Phone:       "1234567890",
			YOE:         5,
			Company:     "Tech Corp",
			Designation: "Senior Developer",
			GithubId:    "johndoe",
			LinkedIn:    "johndoe",
			TwitterId:   "johndoe",
			InstagramId: "johndoe",
			Website:     "https://johndoe.com",
		},
		IsValid:     true,
		Description: "Complete valid Res struct",
	},
	{
		Name: "MissingRequiredFields",
		Res: utils.Res{
			FirstName: "John",
			Email:     "john.doe@example.com",
		},
		IsValid:     false,
		Description: "Missing required fields should fail validation",
	},
	{
		Name: "InvalidEmail",
		Res: utils.Res{
			FirstName:   "John",
			LastName:    "Doe",
			Email:       "invalid-email",
			Phone:       "1234567890",
			YOE:         5,
			Company:     "Tech Corp",
			Designation: "Developer",
			GithubId:    "johndoe",
			LinkedIn:    "johndoe",
		},
		IsValid:     false,
		Description: "Invalid email format should fail validation",
	},
	{
		Name: "NegativeYOE",
		Res: utils.Res{
			FirstName:   "John",
			LastName:    "Doe",
			Email:       "john.doe@example.com",
			Phone:       "1234567890",
			YOE:         -1,
			Company:     "Tech Corp",
			Designation: "Developer",
			GithubId:    "johndoe",
			LinkedIn:    "johndoe",
		},
		IsValid:     false,
		Description: "Negative years of experience should fail validation",
	},
	{
		Name: "InvalidPhone",
		Res: utils.Res{
			FirstName:   "John",
			LastName:    "Doe",
			Email:       "john.doe@example.com",
			Phone:       "invalid-phone",
			YOE:         5,
			Company:     "Tech Corp",
			Designation: "Developer",
			GithubId:    "johndoe",
			LinkedIn:    "johndoe",
		},
		IsValid:     false,
		Description: "Non-digit phone number should fail validation",
	},
	{
		Name: "InvalidWebsite",
		Res: utils.Res{
			FirstName:   "John",
			LastName:    "Doe",
			Email:       "john.doe@example.com",
			Phone:       "1234567890",
			YOE:         5,
			Company:     "Tech Corp",
			Designation: "Developer",
			GithubId:    "johndoe",
			LinkedIn:    "johndoe",
			Website:     "not-a-url",
		},
		IsValid:     false,
		Description: "Invalid website URL should fail validation",
	},
}

var MockResponses = struct {
	ProfileSkippedNoUserID    string
	ProfileSkippedNoURL       string
	ProfileSkippedBlocked     string
	ProfileSkippedServiceDown string
	ProfileSaved              string
}{
	ProfileSkippedNoUserID:    "Profile Skipped No UserID",
	ProfileSkippedNoURL:       "Profile Skipped No Profile URL",
	ProfileSkippedBlocked:     "Profile Skipped Profile Service Blocked",
	ProfileSkippedServiceDown: "Profile Skipped Service Down",
	ProfileSaved:              "Profile Saved",
}

var EmptyUserIdTests = []struct {
	Name           string
	Body           string
	ExpectedBody   string
	ExpectedStatus int
	Description    string
}{
	{
		Name:           "EmptyUserId",
		Body:           `{"userId": "", "sessionId": "session123"}`,
		ExpectedBody:   MockResponses.ProfileSkippedNoUserID,
		ExpectedStatus: 200,
		Description:    "Empty userId should return skip message",
	},
	{
		Name:           "MissingUserId",
		Body:           `{"sessionId": "session123"}`,
		ExpectedBody:   MockResponses.ProfileSkippedNoUserID,
		ExpectedStatus: 200,
		Description:    "Missing userId field should return skip message",
	},
	{
		Name:           "EmptyJSON",
		Body:           `{}`,
		ExpectedBody:   MockResponses.ProfileSkippedNoUserID,
		ExpectedStatus: 200,
		Description:    "Empty JSON should return skip message",
	},
	{
		Name:           "NullUserId",
		Body:           `{"userId": null, "sessionId": "session123"}`,
		ExpectedBody:   MockResponses.ProfileSkippedNoUserID,
		ExpectedStatus: 200,
		Description:    "Null userId should return skip message",
	},
}
