// Package longbridge manages browser authorization independently of market
// adapters. An OAuth grant is not evidence of real-time quote entitlement.
package longbridge

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var (
	ErrNotConnected   = errors.New("longbridge: account is not connected")
	ErrSessionChanged = errors.New("longbridge: authorization session changed")
	ErrAuthorization  = errors.New("longbridge: authorization expired or rejected")
	ErrDenied         = errors.New("longbridge: authorization denied")
	ErrCallback       = errors.New("longbridge: callback listener unavailable")
	ErrStorage        = errors.New("longbridge: credential store unavailable")
	ErrResponse       = errors.New("longbridge: invalid OAuth response")
)

const (
	clientKey = "longbridge-oauth-client"
	tokensKey = "longbridge-oauth-tokens"
)

// Store persists credential payloads securely. Get returns nil, nil for a
// missing entry; Set must atomically replace an entry. Local payload files must
// be authenticated ciphertext with their key protected by the system vault.
type Store interface {
	Get(string) ([]byte, error)
	Set(string, []byte) error
	Delete(string) error
}

type Tokens struct {
	ClientID     string    `json:"client_id"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

func (t Tokens) valid() bool {
	return t.ClientID != "" && t.AccessToken != "" && t.RefreshToken != "" && !t.ExpiresAt.IsZero()
}

type Status struct {
	Authorized         bool
	NeedsAuthorization bool
}

type registration struct {
	ClientID    string `json:"client_id"`
	Scope       string `json:"scope,omitempty"`
	RedirectURI string `json:"redirect_uri"`
}

type Client struct {
	authorizationMu sync.Mutex // A canceled listener must close before a retry binds.
	mu              sync.Mutex // Serializes credential writes and refresh-token rotations.
	store           Store
	http            *http.Client
	baseURL         string
	address         string
	pendingRefresh  *Tokens // Retain a rotation if the secure store temporarily fails.
	cachedTokens    *Tokens // This Client owns the in-process grant; never reread per API call.
	tokensLoaded    bool
	generation      uint64 // Changes only when a grant is replaced or disconnected.
	apiRateMu       sync.Mutex
	nextAPIRequest  time.Time
}

func NewClient(store Store) *Client {
	return &Client{
		store: store,
		http: &http.Client{
			Timeout: 20 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		baseURL: "https://openapi.longbridge.com",
		// Keep the registered redirect stable, and do not compete with Pulse's
		// listener. Bind IPv4 loopback only, never a public interface.
		address: "127.0.0.1:41927",
	}
}

// Restore refreshes near-expiry grants. Rotated refresh tokens are saved before
// returning, even if the caller dismisses the UI after the response arrives.
func (c *Client) Restore(ctx context.Context) (Status, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	tokens, err := c.freshTokensLocked(ctx)
	if errors.Is(err, ErrNotConnected) {
		return Status{}, nil
	}
	if errors.Is(err, ErrAuthorization) {
		return Status{NeedsAuthorization: true}, err
	}
	return Status{Authorized: tokens.valid() && time.Now().Before(tokens.ExpiresAt)}, err
}

// freshTokensLocked is shared by settings and market requests: only this owner
// refreshes/rotates tokens, and no SDK writes them into a plaintext cache.
func (c *Client) freshTokensLocked(ctx context.Context) (Tokens, error) {
	if err := ctx.Err(); err != nil {
		return Tokens{}, err
	}
	if c.pendingRefresh != nil {
		if err := c.write(tokensKey, *c.pendingRefresh); err != nil {
			return Tokens{}, err
		}
		c.cachedTokens = c.pendingRefresh
		c.tokensLoaded = true
		c.pendingRefresh = nil
	}
	if !c.tokensLoaded {
		var tokens Tokens
		found, err := c.read(tokensKey, &tokens)
		if err != nil {
			return Tokens{}, err
		}
		if found {
			c.cachedTokens = &tokens
		}
		c.tokensLoaded = true
	}
	if c.cachedTokens == nil {
		return Tokens{}, ErrNotConnected
	}
	tokens := *c.cachedTokens
	if !tokens.valid() {
		return Tokens{}, ErrAuthorization
	}
	if time.Until(tokens.ExpiresAt) > time.Minute {
		return tokens, nil
	}
	refreshed, err := c.exchange(ctx, tokens.ClientID, url.Values{
		"grant_type": {"refresh_token"}, "client_id": {tokens.ClientID},
		"refresh_token": {tokens.RefreshToken},
	})
	if err != nil {
		return tokens, err
	}
	if refreshed.RefreshToken == "" {
		refreshed.RefreshToken = tokens.RefreshToken
	}
	c.pendingRefresh = &refreshed
	if err := c.write(tokensKey, refreshed); err != nil {
		return Tokens{}, err
	}
	c.cachedTokens = &refreshed
	c.tokensLoaded = true
	c.pendingRefresh = nil
	return refreshed, nil
}

// Authorize returns a grant without committing it. The UI rejects canceled or
// stale results before starting Save; a failed reauthorization keeps the old
// grant. callbackMessage is localized by the presentation layer.
func (c *Client) Authorize(ctx context.Context, openURL func(string), callbackMessage string) (Tokens, error) {
	c.authorizationMu.Lock()
	defer c.authorizationMu.Unlock()
	if err := ctx.Err(); err != nil {
		return Tokens{}, err
	}
	listener, err := net.Listen("tcp4", c.address)
	if err != nil {
		return Tokens{}, fmt.Errorf("%w: %v", ErrCallback, err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port
	redirect := fmt.Sprintf("http://localhost:%d/oauth/callback", port)
	client, err := c.ensureClient(ctx, redirect)
	if err != nil {
		return Tokens{}, err
	}
	verifier, state := randomURLSafe(), randomURLSafe()
	digest := sha256.Sum256([]byte(verifier))
	query := url.Values{
		"response_type": {"code"}, "client_id": {client.ClientID},
		"redirect_uri": {redirect}, "state": {state},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(digest[:])},
		"code_challenge_method": {"S256"},
	}
	if client.Scope != "" {
		query.Set("scope", client.Scope)
	}
	type callback struct {
		code string
		err  error
	}
	callbacks := make(chan callback, 1)
	redirectURL, _ := url.Parse(redirect)
	server := &http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       5 * time.Second,
		MaxHeaderBytes:    8192,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Referrer-Policy", "no-referrer")
			w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
			q := r.URL.Query()
			if r.Method != http.MethodGet || r.URL.Path != redirectURL.Path || r.Host != redirectURL.Host ||
				len(q["state"]) != 1 || q.Get("state") != state {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			result := callback{}
			switch {
			case len(q["error"]) == 1 && q.Get("error") != "":
				result.err = ErrDenied
			case len(q["code"]) == 1 && q.Get("code") != "" && !q.Has("error"):
				result.code = q.Get("code")
			default:
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, "<!doctype html><html><head><meta charset=utf-8><title>Trajon</title></head><body><p>%s</p></body></html>", html.EscapeString(callbackMessage))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			select {
			case callbacks <- result:
			default:
			}
		}),
	}
	defer server.Close()
	serveError := make(chan error, 1)
	go func() { serveError <- server.Serve(listener) }()
	if err := ctx.Err(); err != nil {
		return Tokens{}, err
	}
	openURL(c.baseURL + "/oauth2/authorize?" + query.Encode())
	select {
	case result := <-callbacks:
		if result.err != nil {
			return Tokens{}, result.err
		}
		return c.exchange(ctx, client.ClientID, url.Values{
			"grant_type": {"authorization_code"}, "client_id": {client.ClientID},
			"redirect_uri": {redirect}, "code": {result.code}, "code_verifier": {verifier},
		})
	case <-ctx.Done():
		return Tokens{}, ctx.Err()
	case err := <-serveError:
		return Tokens{}, fmt.Errorf("%w: %v", ErrCallback, err)
	}
}

func (c *Client) Save(tokens Tokens) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !tokens.valid() || !time.Now().Before(tokens.ExpiresAt) {
		return ErrAuthorization
	}
	if err := c.write(tokensKey, tokens); err != nil {
		return err
	}
	c.cachedTokens = &tokens
	c.tokensLoaded = true
	c.pendingRefresh = nil
	c.generation++
	return nil
}

// Disconnect removes this app's saved grant, not the authorization on the
// provider's website. Keep the client registration stable for future grants.
func (c *Client) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.store.Delete(tokensKey); err != nil {
		return fmt.Errorf("%w: %v", ErrStorage, err)
	}
	c.cachedTokens = nil
	c.tokensLoaded = true
	c.pendingRefresh = nil
	c.generation++
	return nil
}

func (c *Client) ensureClient(ctx context.Context, redirect string) (registration, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var cached registration
	found, err := c.read(clientKey, &cached)
	if err != nil {
		return registration{}, err
	}
	if found && cached.ClientID != "" && cached.RedirectURI == redirect {
		return cached, nil
	}
	body, _ := json.Marshal(map[string]any{
		"client_name": "Trajon", "redirect_uris": []string{redirect},
		"grant_types":    []string{"authorization_code", "refresh_token"},
		"response_types": []string{"code"}, "token_endpoint_auth_method": "none",
	})
	var registered registration
	if err := c.request(ctx, "/oauth2/register", "application/json", bytes.NewReader(body), &registered); err != nil {
		return registration{}, err
	}
	if registered.ClientID == "" {
		return registration{}, ErrResponse
	}
	registered.RedirectURI = redirect
	if err := c.write(clientKey, registered); err != nil {
		return registration{}, err
	}
	return registered, nil
}

func (c *Client) exchange(ctx context.Context, clientID string, form url.Values) (Tokens, error) {
	var response struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}
	if err := c.request(ctx, "/oauth2/token", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()), &response); err != nil {
		return Tokens{}, err
	}
	if response.AccessToken == "" || response.ExpiresIn <= 0 || response.ExpiresIn > int64((365*24*time.Hour)/time.Second) ||
		(response.TokenType != "" && !strings.EqualFold(response.TokenType, "Bearer")) {
		return Tokens{}, ErrResponse
	}
	if form.Get("grant_type") == "authorization_code" && response.RefreshToken == "" {
		return Tokens{}, ErrResponse
	}
	return Tokens{ClientID: clientID, AccessToken: response.AccessToken, RefreshToken: response.RefreshToken,
		ExpiresAt: time.Now().Add(time.Duration(response.ExpiresIn) * time.Second)}, nil
}

func (c *Client) request(ctx context.Context, path, contentType string, body io.Reader, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		var failure struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(data, &failure)
		if res.StatusCode == http.StatusUnauthorized || failure.Error == "invalid_grant" || failure.Error == "invalid_client" {
			return ErrAuthorization
		}
		if failure.Error == "access_denied" {
			return ErrDenied
		}
		// Never surface response bodies: OAuth failures can echo credentials.
		return fmt.Errorf("longbridge: %s returned HTTP %d", path, res.StatusCode)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return ErrResponse
	}
	return nil
}

func (c *Client) read(key string, out any) (bool, error) {
	data, err := c.store.Get(key)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrStorage, err)
	}
	if len(data) == 0 {
		return false, nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return false, ErrStorage
	}
	return true, nil
}

func (c *Client) write(key string, value any) error {
	data, err := json.Marshal(value)
	if err == nil {
		err = c.store.Set(key, data)
	}
	if err != nil {
		return fmt.Errorf("%w: %v", ErrStorage, err)
	}
	return nil
}

func randomURLSafe() string {
	var value [32]byte
	_, _ = rand.Read(value[:])
	return base64.RawURLEncoding.EncodeToString(value[:])
}
