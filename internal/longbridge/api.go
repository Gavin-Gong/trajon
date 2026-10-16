package longbridge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// APIError preserves provider details without including credentials or raw
// response bodies. A quote-entitlement failure is not an expired OAuth grant.
type APIError struct {
	Status  int
	Code    int
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("Longbridge (%d / HTTP %d): %s", e.Code, e.Status, e.Message)
}
func (e *APIError) Unwrap() error {
	if e.Status == http.StatusUnauthorized || e.Code >= 401000 && e.Code < 402000 {
		return ErrAuthorization
	}
	return nil
}

// Request calls the official quote HTTP transport used by openapi-go v0.29.1
// (quote/transport.go). It reuses our vault-backed OAuth owner rather than the
// SDK's separate plaintext token cache. Only read-only quote routes are allowed.
func (c *Client) Request(ctx context.Context, path string, input, output any) error {
	if path != "/quote/history-candlesticks" && path != "/quote/static-info" {
		return fmt.Errorf("longbridge: unsupported quote route")
	}
	if err := c.waitForAPI(ctx); err != nil {
		return err
	}
	c.mu.Lock()
	tokens, err := c.freshTokensLocked(ctx)
	generation := c.generation
	c.mu.Unlock()
	if err != nil {
		return err
	}
	body, err := json.Marshal(input)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	token := strings.TrimPrefix(tokens.AccessToken, "Bearer ")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Api-Key", tokens.ClientID)
	region := "ap"
	if strings.HasPrefix(token, "us_") {
		region = "us"
	}
	req.Header.Set("X-Dc-Region", region)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	response, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	const maxBody = 16 << 20
	data, err := io.ReadAll(io.LimitReader(response.Body, maxBody+1))
	if err != nil {
		return err
	}
	c.mu.Lock()
	stale := generation != c.generation
	c.mu.Unlock()
	if stale {
		return ErrSessionChanged
	}
	if len(data) > maxBody {
		return ErrResponse
	}
	var envelope struct {
		Code    json.Number     `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	decodeErr := json.Unmarshal(data, &envelope)
	code, codeErr := strconv.Atoi(string(envelope.Code))
	if response.StatusCode < 200 || response.StatusCode >= 300 || decodeErr == nil && codeErr == nil && code != 0 {
		message := envelope.Message
		if decodeErr != nil || message == "" {
			message = http.StatusText(response.StatusCode)
		}
		for _, secret := range []string{tokens.AccessToken, token, tokens.RefreshToken} {
			if secret != "" {
				message = strings.ReplaceAll(message, secret, "[redacted]")
			}
		}
		if len(message) > 512 {
			message = strings.ToValidUTF8(message[:512], "") + "…"
		}
		return &APIError{Status: response.StatusCode, Code: code, Message: message}
	}
	if decodeErr != nil || codeErr != nil {
		return ErrResponse
	}
	if len(envelope.Data) == 0 || bytes.Equal(bytes.TrimSpace(envelope.Data), []byte("null")) {
		return nil
	}
	if err := json.Unmarshal(envelope.Data, output); err != nil {
		return ErrResponse
	}
	return nil
}

// The public HTTP limit is ten requests per second. Reserve cancellable slots
// independently of OAuth's lock so browsing cannot race refresh-token rotation.
func (c *Client) waitForAPI(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.apiRateMu.Lock()
	at := c.nextAPIRequest
	if at.Before(time.Now()) {
		at = time.Now()
	}
	c.nextAPIRequest = at.Add(120 * time.Millisecond)
	c.apiRateMu.Unlock()
	timer := time.NewTimer(time.Until(at))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
