package mocks

import (
	"io"
	"net/http"
)

type MockClient struct {
	PostFunc func(url string, contentType string, body io.Reader) (*http.Response, error)
}

var (
	PostFunc func(url string, contentType string, body io.Reader) (*http.Response, error)
)

func (m *MockClient) Post(url string, contentType string, body io.Reader) (*http.Response, error) {
	return PostFunc(url, contentType, body)
}
