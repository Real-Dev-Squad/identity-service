package utils

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"
)

const DefaultHTTPTimeout = 30 * time.Second

var (
	defaultClient     *http.Client
	defaultClientOnce sync.Once
	
	clientCache     = make(map[time.Duration]*http.Client)
	clientCacheLock sync.RWMutex
	
	// Shared HTTP client without timeout - relies on context for timeout control
	sharedHTTPClient = &http.Client{}
)

func getDefaultClient() *http.Client {
	defaultClientOnce.Do(func() {
		defaultClient = &http.Client{
			Timeout: DefaultHTTPTimeout,
		}
	})
	return defaultClient
}

func CreateHTTPClient(timeout time.Duration) *http.Client {
	if timeout == 0 {
		timeout = DefaultHTTPTimeout
	}
	
	if timeout == DefaultHTTPTimeout {
		return getDefaultClient()
	}
	
	clientCacheLock.RLock()
	if client, exists := clientCache[timeout]; exists {
		clientCacheLock.RUnlock()
		return client
	}
	clientCacheLock.RUnlock()
	
	clientCacheLock.Lock()
	defer clientCacheLock.Unlock()
	
	if client, exists := clientCache[timeout]; exists {
		return client
	}
	
	client := &http.Client{
		Timeout: timeout,
	}
	clientCache[timeout] = client
	return client
}

func DoRequestWithContext(ctx context.Context, client *http.Client, req *http.Request) (*http.Response, error) {
	return client.Do(req.WithContext(ctx))
}

func setupRequest(ctx context.Context, method string, url string, body io.Reader, timeout time.Duration) (context.CancelFunc, *http.Request, error) {
	if timeout == 0 {
		timeout = DefaultHTTPTimeout
	}
	
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	req, err := http.NewRequestWithContext(reqCtx, method, url, body)
	if err != nil {
		cancel()
		return nil, nil, err
	}
	
	return cancel, req, nil
}

func PostWithContext(ctx context.Context, url string, contentType string, body io.Reader, timeout time.Duration) (*http.Response, error) {
	cancel, req, err := setupRequest(ctx, "POST", url, body, timeout)
	if err != nil {
		return nil, err
	}
	defer cancel()
	
	req.Header.Set("Content-Type", contentType)
	return sharedHTTPClient.Do(req)
}

func GetWithContext(ctx context.Context, url string, timeout time.Duration) (*http.Response, error) {
	cancel, req, err := setupRequest(ctx, "GET", url, nil, timeout)
	if err != nil {
		return nil, err
	}
	defer cancel()
	
	return sharedHTTPClient.Do(req)
}
