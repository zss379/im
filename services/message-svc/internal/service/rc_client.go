package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// CheckChainReq maps to rc-svc CheckChain API request body.
type CheckChainReq struct {
	Content    string `json:"content"`
	SenderID   int64  `json:"sender_id"`
	SenderType int8   `json:"sender_type"` // 1=user, 2=bot
	FileType   string `json:"file_type"`
	FileSize   int    `json:"file_size"`
}

// PreflightResult is the parsed result from rc-svc CheckChain.
type PreflightResult struct {
	Passed bool

	// Sensitive word result
	Blocked  bool
	HitWords []string

	// Rate limit result
	RateLimited bool
	RetryAfter  int // seconds hint
}

// rcClient is an HTTP client for calling rc-svc CheckChain API.
type rcClient struct {
	httpClient *http.Client
	addr       string
}

func NewRCClient(addr string, timeout time.Duration) *rcClient {
	return &rcClient{
		httpClient: &http.Client{Timeout: timeout},
		addr:       addr,
	}
}

func (c *rcClient) CheckChain(ctx context.Context, req *CheckChainReq) (*PreflightResult, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("http://%s/api/v1/sensitive/check", c.addr),
		bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call rc-svc: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// rc-svc wraps data in standard response: {"code":0,"msg":"success","data":{...}}
	var wrapper struct {
		Code int            `json:"code"`
		Data *checkChainDTO `json:"data"`
	}
	if err := json.Unmarshal(respBody, &wrapper); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if wrapper.Code != 0 {
		return nil, fmt.Errorf("rc-svc error code: %d", wrapper.Code)
	}
	if wrapper.Data == nil {
		return nil, fmt.Errorf("rc-svc empty data")
	}

	result := &PreflightResult{Passed: wrapper.Data.Passed}

	if wrapper.Data.SensitiveCheck != nil {
		result.Blocked = wrapper.Data.SensitiveCheck.Blocked
		result.HitWords = wrapper.Data.SensitiveCheck.HitWords
	}

	if wrapper.Data.RateLimitCheck != nil && !wrapper.Data.RateLimitCheck.Passed {
		result.RateLimited = true
		result.RetryAfter = 1 // default 1s; rc-svc doesn't return window info
	}

	return result, nil
}

// checkChainDTO mirrors rc-svc model.CheckChainResp for decoding.
type checkChainDTO struct {
	Passed         bool               `json:"passed"`
	SensitiveCheck *sensitiveCheckDTO `json:"sensitive_check,omitempty"`
	RateLimitCheck *rateLimitCheckDTO `json:"rate_limit_check,omitempty"`
}

type sensitiveCheckDTO struct {
	Passed   bool     `json:"passed"`
	HitWords []string `json:"hit_words,omitempty"`
	Blocked  bool     `json:"blocked,omitempty"`
}

type rateLimitCheckDTO struct {
	Passed    bool `json:"passed"`
	Remaining int  `json:"remaining"`
}

// ErrBlocked is returned when a message is blocked by sensitive word check.
type ErrBlocked struct {
	Reason   string
	HitWords []string
}

func (e *ErrBlocked) Error() string {
	return fmt.Sprintf("message blocked: %s (hit: %v)", e.Reason, e.HitWords)
}

// ErrRateLimited is returned when rate limit is exceeded.
type ErrRateLimited struct {
	RetryAfter int
}

func (e *ErrRateLimited) Error() string {
	return fmt.Sprintf("rate limited, retry after %ds", e.RetryAfter)
}
