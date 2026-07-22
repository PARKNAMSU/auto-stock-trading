// Package tossinvest는 토스증권 Open API와 통신하는 어댑터를 제공합니다.
package tossinvest

import (
	"net/http"
	"time"
)

const DefaultBaseURL = "https://openapi.tossinvest.com"

// Client is the boundary for the Toss Securities Open API.
// Endpoint implementations will be added after authentication is wired.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}
