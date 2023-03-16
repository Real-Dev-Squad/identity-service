package main

import (
	"net/http"
	"testing"

	"profile/utils/mockclient"

	"github.com/stretchr/testify/assert"
)

func init() {
	Client = &mockclient.MockClient{}
}

func TestCheckIsServiceRunning(t *testing.T) {
	// hurray! we are ready to mock the api responses now
	mockclient.DoFunc = func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
		}, nil
	}

	result := checkIfServiceIsRunning("http://ruthvik.dev/")

	assert.NotNil(t, result)
	assert.EqualValues(t, true, result)
}
