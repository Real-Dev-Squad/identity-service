package main

import (
	"net/http"
	"time"

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
			Path:       "/health-check",
		},
		ExpectedErr: true,
		Description: "Basic GET request should fail at Firestore initialization",
	},
	{
		Name: "BasicPOSTRequest",
		Request: events.APIGatewayProxyRequest{
			HTTPMethod: "POST",
			Path:       "/health-check",
		},
		ExpectedErr: true,
		Description: "Basic POST request should fail at Firestore initialization",
	},
	{
		Name: "PUTRequest",
		Request: events.APIGatewayProxyRequest{
			HTTPMethod: "PUT",
			Path:       "/health-check",
		},
		ExpectedErr: true,
		Description: "PUT request should fail at Firestore initialization",
	},
	{
		Name: "RequestWithQueryParams",
		Request: events.APIGatewayProxyRequest{
			HTTPMethod:            "GET",
			Path:                  "/health-check",
			QueryStringParameters: map[string]string{"filter": "active", "limit": "10"},
		},
		ExpectedErr: true,
		Description: "Request with query parameters should fail at Firestore initialization",
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
	{
		Name:        "URLWithComplexPath",
		Input:       "https://api.example.com/v1/services/user",
		Expected:    "https://api.example.com/v1/services/user/health",
		Description: "URL with complex path",
	},
}

var MockServerTests = []struct {
	Name           string
	ServerResponse int
	ServerDelay    time.Duration
	ExpectedError  bool
	Description    string
}{
	{
		Name:           "SuccessfulHealthCheck",
		ServerResponse: 200,
		ServerDelay:    0,
		ExpectedError:  false,
		Description:    "Successful health check response",
	},
	{
		Name:           "ServerError500",
		ServerResponse: 500,
		ServerDelay:    0,
		ExpectedError:  false,
		Description:    "Server returns 500 error",
	},
	{
		Name:           "ServerError404",
		ServerResponse: 404,
		ServerDelay:    0,
		ExpectedError:  false,
		Description:    "Server returns 404 not found",
	},
	{
		Name:           "ServerError503",
		ServerResponse: 503,
		ServerDelay:    0,
		ExpectedError:  false,
		Description:    "Server returns 503 service unavailable",
	},
	{
		Name:           "ServerTimeout",
		ServerResponse: 200,
		ServerDelay:    3 * time.Second,
		ExpectedError:  false,
		Description:    "Server response timeout",
	},
	{
		Name:           "SlowButSuccessful",
		ServerResponse: 200,
		ServerDelay:    1 * time.Second,
		ExpectedError:  false,
		Description:    "Slow but successful response",
	},
}

var HTTPClientConfigTests = []struct {
	Name        string
	Timeout     time.Duration
	Description string
}{
	{
		Name:        "StandardTimeout",
		Timeout:     2 * time.Second,
		Description: "Standard 2 second timeout",
	},
	{
		Name:        "ShortTimeout",
		Timeout:     500 * time.Millisecond,
		Description: "Short 500ms timeout",
	},
	{
		Name:        "LongTimeout",
		Timeout:     10 * time.Second,
		Description: "Long 10 second timeout",
	},
}

var HTTPRequestTests = []struct {
	Name        string
	Method      string
	URL         string
	Description string
}{
	{
		Name:        "HTTPSRequest",
		Method:      "GET",
		URL:         "https://example.com/health",
		Description: "HTTPS GET request",
	},
	{
		Name:        "HTTPRequest",
		Method:      "GET",
		URL:         "http://localhost:3000/health",
		Description: "HTTP GET request to localhost",
	},
	{
		Name:        "URLWithPort",
		Method:      "GET",
		URL:         "https://api.example.com:8080/health",
		Description: "HTTPS request with custom port",
	},
	{
		Name:        "POSTRequest",
		Method:      "POST",
		URL:         "https://example.com/health",
		Description: "HTTPS POST request",
	},
	{
		Name:        "PUTRequest",
		Method:      "PUT",
		URL:         "https://example.com/health",
		Description: "HTTPS PUT request",
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
		ProfilesCount:  10,
		ExpectedFormat: "Total Profiles called in session is 10",
		Description:    "Response format for multiple profiles",
	},
	{
		Name:           "LargeNumberOfProfiles",
		ProfilesCount:  1000,
		ExpectedFormat: "Total Profiles called in session is 1000",
		Description:    "Response format for large number of profiles",
	},
}

var URLEdgeCaseTests = []struct {
	Name        string
	Input       string
	Expected    string
	ShouldPanic bool
	Description string
}{
	{
		Name:        "NormalURL",
		Input:       "https://example.com",
		Expected:    "https://example.com/health",
		ShouldPanic: false,
		Description: "Normal URL without trailing slash",
	},
	{
		Name:        "URLWithTrailingSlash",
		Input:       "https://example.com/",
		Expected:    "https://example.com/health",
		ShouldPanic: false,
		Description: "URL with trailing slash",
	},
	{
		Name:        "URLWithPath",
		Input:       "https://example.com/api/v1",
		Expected:    "https://example.com/api/v1/health",
		ShouldPanic: false,
		Description: "URL with existing path",
	},
	{
		Name:        "SingleCharacterURL",
		Input:       "h",
		Expected:    "h/health",
		ShouldPanic: false,
		Description: "Single character URL",
	},
	{
		Name:        "TwoCharacterURL",
		Input:       "ht",
		Expected:    "ht/health",
		ShouldPanic: false,
		Description: "Two character URL",
	},
	{
		Name:        "URLWithQueryParams",
		Input:       "https://example.com?param=value",
		Expected:    "https://example.com?param=value/health",
		ShouldPanic: false,
		Description: "URL with query parameters",
	},
}

var ConcurrentHealthCheckTests = []struct {
	Name        string
	URLs        []string
	Description string
}{
	{
		Name: "SmallBatch",
		URLs: []string{
			"https://service1.example.com",
			"https://service2.example.com",
			"https://service3.example.com",
		},
		Description: "Small batch of concurrent health checks",
	},
	{
		Name: "MediumBatch",
		URLs: []string{
			"https://service1.example.com",
			"https://service2.example.com",
			"https://service3.example.com",
			"https://service4.example.com",
			"https://service5.example.com",
			"https://service6.example.com",
			"https://service7.example.com",
			"https://service8.example.com",
		},
		Description: "Medium batch of concurrent health checks",
	},
	{
		Name: "LargeBatch",
		URLs: generateURLs(20),
		Description: "Large batch of concurrent health checks",
	},
}

var EmptyURLHandlingTests = []struct {
	Name        string
	URLs        []string
	Description string
}{
	{
		Name:        "MixedURLs",
		URLs:        []string{"", "h", "ht", "http", "https://example.com"},
		Description: "Mix of empty and valid URLs",
	},
	{
		Name:        "OnlyEmptyURLs",
		URLs:        []string{"", "", ""},
		Description: "Only empty URLs",
	},
	{
		Name:        "SingleCharURLs",
		URLs:        []string{"a", "b", "c", "d", "e"},
		Description: "Single character URLs",
	},
}

func generateURLs(count int) []string {
	urls := make([]string, count)
	for i := 0; i < count; i++ {
		urls[i] = "https://service" + string(rune('1'+i)) + ".example.com"
	}
	return urls
}

var MockHTTPResponses = struct {
	Success        string
	Error          string
	NotFound       string
	ServiceUnavail string
}{
	Success:        "OK",
	Error:          "Internal Server Error",
	NotFound:       "Not Found",
	ServiceUnavail: "Service Unavailable",
}

var HTTPStatusCodeTests = []struct {
	Name        string
	StatusCode  int
	Description string
}{
	{Name: "OK", StatusCode: http.StatusOK, Description: "200 OK"},
	{Name: "Created", StatusCode: http.StatusCreated, Description: "201 Created"},
	{Name: "BadRequest", StatusCode: http.StatusBadRequest, Description: "400 Bad Request"},
	{Name: "Unauthorized", StatusCode: http.StatusUnauthorized, Description: "401 Unauthorized"},
	{Name: "Forbidden", StatusCode: http.StatusForbidden, Description: "403 Forbidden"},
	{Name: "NotFound", StatusCode: http.StatusNotFound, Description: "404 Not Found"},
	{Name: "InternalServerError", StatusCode: http.StatusInternalServerError, Description: "500 Internal Server Error"},
	{Name: "BadGateway", StatusCode: http.StatusBadGateway, Description: "502 Bad Gateway"},
	{Name: "ServiceUnavailable", StatusCode: http.StatusServiceUnavailable, Description: "503 Service Unavailable"},
	{Name: "GatewayTimeout", StatusCode: http.StatusGatewayTimeout, Description: "504 Gateway Timeout"},
}
