// Package tossinvest는 토스증권 Open API와 통신하는 어댑터를 제공합니다.
package tossinvest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	DefaultBaseURL       = "https://openapi.tossinvest.com"
	defaultTimeout       = 10 * time.Second
	defaultMaxRetries    = 3
	defaultRetryBase     = 250 * time.Millisecond
	defaultRetryMax      = 4 * time.Second
	defaultRefreshBefore = 30 * time.Second
)

// Config contains API connection settings. Account is the accountSeq returned
// by GET /api/v1/accounts, not an account number.
type Config struct {
	ClientID      string
	ClientSecret  string
	Account       string
	BaseURL       string
	HTTPClient    *http.Client
	Logger        *slog.Logger
	MaxRetries    int
	RetryBase     time.Duration
	RetryMax      time.Duration
	RefreshBefore time.Duration
}

// APIError is the common Toss Securities error envelope.
type APIError struct {
	StatusCode int
	RequestID  string
	Code       string
	Message    string
	Data       json.RawMessage
	RetryAfter time.Duration
}

func (e *APIError) Error() string {
	if e.Code == "" {
		return fmt.Sprintf("tossinvest API returned HTTP %d", e.StatusCode)
	}
	return fmt.Sprintf("tossinvest API returned HTTP %d (%s)", e.StatusCode, e.Code)
}

// Client authenticates and sends requests to the Toss Securities Open API.
type Client struct {
	baseURL       *url.URL
	httpClient    *http.Client
	clientID      string
	clientSecret  string
	account       string
	logger        *slog.Logger
	maxRetries    int
	retryBase     time.Duration
	retryMax      time.Duration
	refreshBefore time.Duration
	now           func() time.Time
	sleep         func(context.Context, time.Duration) error

	tokenMu sync.Mutex
	token   accessToken
}

type accessToken struct {
	value     string
	tokenType string
	expiresAt time.Time
}

// NewClient validates settings and creates an authenticated common client.
func NewClient(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.ClientID) == "" {
		return nil, errors.New("tossinvest client ID is required")
	}
	if strings.TrimSpace(cfg.ClientSecret) == "" {
		return nil, errors.New("tossinvest client secret is required")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	baseURL, err := url.Parse(cfg.BaseURL)
	if err != nil || baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, errors.New("tossinvest base URL must be an absolute URL")
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: defaultTimeout}
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = defaultMaxRetries
	}
	if cfg.MaxRetries < 0 {
		return nil, errors.New("tossinvest max retries cannot be negative")
	}
	if cfg.RetryBase == 0 {
		cfg.RetryBase = defaultRetryBase
	}
	if cfg.RetryMax == 0 {
		cfg.RetryMax = defaultRetryMax
	}
	if cfg.RefreshBefore == 0 {
		cfg.RefreshBefore = defaultRefreshBefore
	}

	client := &Client{
		baseURL: baseURL, httpClient: cfg.HTTPClient,
		clientID: cfg.ClientID, clientSecret: cfg.ClientSecret, account: cfg.Account,
		logger: cfg.Logger, maxRetries: cfg.MaxRetries, retryBase: cfg.RetryBase,
		retryMax: cfg.RetryMax, refreshBefore: cfg.RefreshBefore, now: time.Now,
	}
	client.sleep = sleepContext
	return client, nil
}

// NewRequest creates a request relative to the configured API base URL.
func (c *Client) NewRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	reference, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("parse request path: %w", err)
	}
	if reference.IsAbs() || reference.Host != "" {
		return nil, errors.New("request path must be relative to the Toss Securities API")
	}
	requestURL := c.baseURL.ResolveReference(reference)
	req, err := http.NewRequestWithContext(ctx, method, requestURL.String(), body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	return req, nil
}

// Do sends an authenticated request. Successful callers own and must close the
// response body. Non-2xx responses are returned as *APIError.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, errors.New("request is required")
	}
	if req.URL.Scheme != c.baseURL.Scheme || req.URL.Host != c.baseURL.Host {
		return nil, errors.New("refusing to send credentials to an unconfigured host")
	}

	token, err := c.validToken(req.Context(), "")
	if err != nil {
		return nil, err
	}

	for authAttempt := 0; authAttempt < 2; authAttempt++ {
		resp, err := c.send(req, token)
		if err != nil {
			return nil, err
		}
		apiErr := decodeAPIError(resp)
		if apiErr == nil {
			return resp, nil
		}
		if authAttempt == 0 && resp.StatusCode == http.StatusUnauthorized &&
			(apiErr.Code == "expired-token" || apiErr.Code == "invalid-token") {
			token, err = c.validToken(req.Context(), token)
			if err != nil {
				return nil, err
			}
			continue
		}
		return nil, apiErr
	}
	return nil, errors.New("authentication retry exhausted")
}

func (c *Client) send(req *http.Request, token string) (*http.Response, error) {
	maxAttempts := 1
	if isRetryableMethod(req.Method) {
		maxAttempts += c.maxRetries
	}
	for attempt := 0; attempt < maxAttempts; attempt++ {
		current, err := cloneRequest(req, attempt)
		if err != nil {
			return nil, err
		}
		current.Header.Set("Authorization", "Bearer "+token)
		if c.account != "" {
			current.Header.Set("X-Tossinvest-Account", c.account)
		}
		resp, err := c.httpClient.Do(current)
		if err != nil {
			if attempt+1 == maxAttempts {
				return nil, fmt.Errorf("send tossinvest request: %w", err)
			}
			if err := c.sleep(req.Context(), c.backoff(attempt, 0)); err != nil {
				return nil, err
			}
			continue
		}
		c.logger.Debug("tossinvest request completed", "method", req.Method, "path", req.URL.Path, "status", resp.StatusCode, "request_id", resp.Header.Get("X-Request-Id"))
		if attempt+1 < maxAttempts && (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500) {
			retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"), c.now())
			resp.Body.Close()
			if err := c.sleep(req.Context(), c.backoff(attempt, retryAfter)); err != nil {
				return nil, err
			}
			continue
		}
		return resp, nil
	}
	return nil, errors.New("request retry exhausted")
}

func (c *Client) validToken(ctx context.Context, rejected string) (string, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	if c.token.value != "" && c.token.value != rejected && c.now().Add(c.refreshBefore).Before(c.token.expiresAt) {
		return c.token.value, nil
	}
	token, err := c.issueToken(ctx)
	if err != nil {
		return "", err
	}
	c.token = token
	return token.value, nil
}

func (c *Client) issueToken(ctx context.Context) (accessToken, error) {
	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
	}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL.ResolveReference(&url.URL{Path: "/oauth2/token"}).String(), strings.NewReader(form))
	if err != nil {
		return accessToken{}, fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.sendTokenRequest(req)
	if err != nil {
		return accessToken{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return accessToken{}, decodeOAuthError(resp)
	}
	var payload struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil {
		return accessToken{}, fmt.Errorf("decode token response: %w", err)
	}
	if payload.AccessToken == "" || payload.TokenType != "Bearer" || payload.ExpiresIn <= 0 {
		return accessToken{}, errors.New("invalid token response")
	}
	return accessToken{value: payload.AccessToken, tokenType: payload.TokenType, expiresAt: c.now().Add(time.Duration(payload.ExpiresIn) * time.Second)}, nil
}

func (c *Client) sendTokenRequest(req *http.Request) (*http.Response, error) {
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		current, err := cloneRequest(req, attempt)
		if err != nil {
			return nil, err
		}
		resp, err := c.httpClient.Do(current)
		if err == nil && resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode < 500 {
			return resp, nil
		}
		if attempt == c.maxRetries {
			if err != nil {
				return nil, fmt.Errorf("issue tossinvest token: %w", err)
			}
			return resp, nil
		}
		retryAfter := time.Duration(0)
		if resp != nil {
			retryAfter = parseRetryAfter(resp.Header.Get("Retry-After"), c.now())
			resp.Body.Close()
		}
		if err := c.sleep(req.Context(), c.backoff(attempt, retryAfter)); err != nil {
			return nil, err
		}
	}
	return nil, errors.New("token retry exhausted")
}

func decodeAPIError(resp *http.Response) *APIError {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	defer resp.Body.Close()
	var envelope struct {
		Error struct {
			RequestID string          `json:"requestId"`
			Code      string          `json:"code"`
			Message   string          `json:"message"`
			Data      json.RawMessage `json:"data"`
		} `json:"error"`
	}
	_ = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&envelope)
	requestID := envelope.Error.RequestID
	if requestID == "" {
		requestID = resp.Header.Get("X-Request-Id")
	}
	return &APIError{StatusCode: resp.StatusCode, RequestID: requestID, Code: envelope.Error.Code, Message: envelope.Error.Message, Data: envelope.Error.Data, RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After"), time.Now())}
}

func decodeOAuthError(resp *http.Response) error {
	defer resp.Body.Close()
	var payload struct {
		Error       string `json:"error"`
		Description string `json:"error_description"`
	}
	_ = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload)
	return &APIError{StatusCode: resp.StatusCode, RequestID: resp.Header.Get("X-Request-Id"), Code: payload.Error, Message: payload.Description, RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After"), time.Now())}
}

func cloneRequest(req *http.Request, attempt int) (*http.Request, error) {
	clone := req.Clone(req.Context())
	if req.Body == nil {
		return clone, nil
	}
	if req.GetBody == nil {
		if attempt == 0 {
			return clone, nil
		}
		return nil, errors.New("request body cannot be replayed")
	}
	body, err := req.GetBody()
	if err != nil {
		return nil, fmt.Errorf("replay request body: %w", err)
	}
	clone.Body = body
	return clone, nil
}

func isRetryableMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut, http.MethodDelete:
		return true
	}
	return false
}

func (c *Client) backoff(attempt int, retryAfter time.Duration) time.Duration {
	delay := c.retryBase
	for i := 0; i < attempt && delay < c.retryMax; i++ {
		delay *= 2
	}
	if delay > c.retryMax {
		delay = c.retryMax
	}
	if retryAfter > delay {
		delay = retryAfter
	}
	return delay
}

func parseRetryAfter(value string, now time.Time) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	if at, err := http.ParseTime(value); err == nil && at.After(now) {
		return at.Sub(now)
	}
	return 0
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return nil
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
