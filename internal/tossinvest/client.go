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
