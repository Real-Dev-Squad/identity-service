package mockclient

import "net/http"

type MockClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

var (
	DoFunc func(req *http.Request) (*http.Response, error)
)

func (m *MockClient) Do(req *http.Request) (*http.Response, error) {
	return DoFunc(req)
}
