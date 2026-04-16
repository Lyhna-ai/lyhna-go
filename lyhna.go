package lyhna

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	defaultBaseURL = "https://www.lyhna.com"
	defaultTimeout = 10 * time.Second
	version        = "0.1.0"
	userAgent      = "lyhna-go/" + version
)

// Client is a Lyhna API client.
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL overrides the default API base URL.
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = url }
}

// WithTimeout overrides the default request timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.http.Timeout = d }
}

// NewClient creates a Lyhna API client.
func NewClient(apiKey string, opts ...Option) *Client {
	c := &Client{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		http:    &http.Client{Timeout: defaultTimeout},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Bind calls POST /v1/bind and returns a Receipt.
func (c *Client) Bind(ctx context.Context, req BindRequest) (Receipt, error) {
	if req.AuthorityTier == "" {
		req.AuthorityTier = "tier_0"
	}

	body, err := json.Marshal(req)
	if err != nil {
		return Receipt{}, &LyhnaError{Message: fmt.Sprintf("marshal request: %v", err)}
	}

	raw, err := c.do(ctx, "POST", "/v1/bind", body)
	if err != nil {
		return Receipt{}, err
	}

	var resp bindResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return Receipt{}, &LyhnaError{Message: fmt.Sprintf("decode response: %v", err)}
	}

	return receiptFromMap(resp.Receipt), nil
}

// VerifyReceipt calls POST /v1/verify and returns the verification result.
func (c *Client) VerifyReceipt(ctx context.Context, receipt Receipt) (VerifyResult, error) {
	payload := receipt.Raw
	if payload == nil {
		b, err := json.Marshal(receipt)
		if err != nil {
			return VerifyResult{}, &LyhnaError{Message: fmt.Sprintf("marshal receipt: %v", err)}
		}
		if err := json.Unmarshal(b, &payload); err != nil {
			return VerifyResult{}, &LyhnaError{Message: fmt.Sprintf("convert receipt: %v", err)}
		}
	}

	body, err := json.Marshal(map[string]interface{}{"receipt": payload})
	if err != nil {
		return VerifyResult{}, &LyhnaError{Message: fmt.Sprintf("marshal request: %v", err)}
	}

	raw, err := c.do(ctx, "POST", "/v1/verify", body)
	if err != nil {
		return VerifyResult{}, err
	}

	var result VerifyResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return VerifyResult{}, &LyhnaError{Message: fmt.Sprintf("decode response: %v", err)}
	}
	return result, nil
}

func (c *Client) do(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, &LyhnaError{Message: fmt.Sprintf("create request: %v", err)}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		if os.IsTimeout(err) || ctx.Err() == context.DeadlineExceeded {
			return nil, &TimeoutError{LyhnaError{Message: err.Error()}}
		}
		return nil, &LyhnaError{Message: fmt.Sprintf("request failed: %v", err)}
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &LyhnaError{Message: fmt.Sprintf("read response: %v", err)}
	}

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return nil, &AuthError{LyhnaError{
			StatusCode: resp.StatusCode,
			Message:    "authentication failed",
			Body:       string(data),
		}}
	}

	if resp.StatusCode >= 400 {
		return nil, &LyhnaError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("API error: %s", string(data)),
		}
	}

	return data, nil
}

func receiptFromMap(m map[string]interface{}) Receipt {
	r := Receipt{Raw: m}
	r.ReceiptID = str(m, "receipt_id")
	r.Outcome = str(m, "outcome")
	r.ActionType = str(m, "action_type")
	r.ActionHash = str(m, "action_hash")
	r.AuthorityTier = str(m, "authority_tier")
	r.ExpiresAt = str(m, "expires_at")
	r.Signature = str(m, "signature")

	if v := str(m, "outcome_reason"); v != "" {
		r.OutcomeReason = v
	} else {
		r.OutcomeReason = str(m, "reason")
	}
	r.EscalateTo = str(m, "escalate_to")

	if v := str(m, "bound_at"); v != "" {
		r.BoundAt = v
	} else {
		r.BoundAt = str(m, "timestamp")
	}
	if v := str(m, "tenant_id"); v != "" {
		r.TenantID = v
	} else {
		r.TenantID = str(m, "tenant_hash")
	}
	r.CanonicalHash = str(m, "canonical_hash")

	return r
}

func str(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// --- Package-level convenience functions ---

var defaultClient *Client

func getDefault() *Client {
	if defaultClient == nil {
		key := os.Getenv("LYHNA_API_KEY")
		if key == "" {
			return nil
		}
		defaultClient = NewClient(key)
	}
	return defaultClient
}

// Bind is a package-level convenience that uses LYHNA_API_KEY from the environment.
func Bind(ctx context.Context, req BindRequest) (Receipt, error) {
	c := getDefault()
	if c == nil {
		return Receipt{}, &LyhnaError{Message: "LYHNA_API_KEY environment variable is not set"}
	}
	return c.Bind(ctx, req)
}

// VerifyReceipt is a package-level convenience that uses LYHNA_API_KEY from the environment.
func VerifyReceipt(ctx context.Context, receipt Receipt) (VerifyResult, error) {
	c := getDefault()
	if c == nil {
		return VerifyResult{}, &LyhnaError{Message: "LYHNA_API_KEY environment variable is not set"}
	}
	return c.VerifyReceipt(ctx, receipt)
}
