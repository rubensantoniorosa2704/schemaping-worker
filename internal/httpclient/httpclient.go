package httpclient

import (
	"fmt"
	"io"
	"net/http"

	"github.com/rubensantoniorosa2704/schemaping-worker/pkg/types"
)

// Do executes an HTTP request for the given monitor and returns the status code, body, and any transport error.
func Do(m types.Monitor) (statusCode int, body []byte, err error) {
	client := &http.Client{Timeout: m.Timeout}

	req, err := http.NewRequest(m.Method, m.URL, nil)
	if err != nil {
		return 0, nil, fmt.Errorf("httpclient: build request: %w", err)
	}

	for k, v := range m.Headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("httpclient: execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("httpclient: read body: %w", err)
	}

	return resp.StatusCode, body, nil
}
