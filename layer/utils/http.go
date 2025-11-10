package utils

import (
	"context"
	"io"
	"net/http"
	"time"
)

const DefaultHTTPTimeout = 30 * time.Second

func CreateHTTPClient(timeout time.Duration) *http.Client {
	if timeout == 0 {
		timeout = DefaultHTTPTimeout
	}
	return &http.Client{
		Timeout: timeout,
	}
}

func DoRequestWithContext(ctx context.Context, client *http.Client, req *http.Request) (*http.Response, error) {
	return client.Do(req)
}

func PostWithContext(ctx context.Context, url string, contentType string, body io.Reader, timeout time.Duration) (*http.Response, error) {
	if timeout == 0 {
		timeout = DefaultHTTPTimeout
	}
	
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	
	client := CreateHTTPClient(timeout)
	req, err := http.NewRequestWithContext(reqCtx, "POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	
	return client.Do(req)
}

func GetWithContext(ctx context.Context, url string, timeout time.Duration) (*http.Response, error) {
	if timeout == 0 {
		timeout = DefaultHTTPTimeout
	}
	
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	
	client := CreateHTTPClient(timeout)
	req, err := http.NewRequestWithContext(reqCtx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	return client.Do(req)
}
