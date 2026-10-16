package longbridge

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestQuoteRequestSharesVaultRefreshAndUsesOAuthHeaders(t *testing.T) {
	store := &memoryStore{data: map[string][]byte{}}
	var client *Client
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth2/token" {
			r.ParseForm()
			if r.Form.Get("refresh_token") != "old-refresh" {
				t.Error("quote request did not use the saved refresh token")
			}
			io.WriteString(w, `{"access_token":"us_rotated","refresh_token":"new-refresh","expires_in":3600}`)
			return
		}
		if r.URL.Path != "/quote/static-info" || r.Method != "POST" || r.Header.Get("Authorization") != "Bearer us_rotated" || r.Header.Get("X-Api-Key") != "client" || r.Header.Get("X-Dc-Region") != "us" {
			t.Error("incorrect quote endpoint or OAuth headers")
		}
		client.mu.Lock()
		var saved Tokens
		json.Unmarshal(store.data[tokensKey], &saved)
		client.mu.Unlock()
		if saved.RefreshToken != "new-refresh" {
			t.Error("market request ran before the rotated grant was persisted")
		}
		io.WriteString(w, `{"code":"0","data":{"value":"ok"}}`)
	}))
	defer server.Close()
	client = NewClient(store)
	client.baseURL = server.URL
	tokens := testTokens()
	tokens.ExpiresAt = time.Now().Add(30 * time.Second)
	if err := client.Save(tokens); err != nil {
		t.Fatal(err)
	}
	var result struct {
		Value string `json:"value"`
	}
	if err := client.Request(context.Background(), "/quote/static-info", map[string]any{"symbol": []string{"AAPL.US"}}, &result); err != nil || result.Value != "ok" {
		t.Fatalf("quote result=%+v, err=%v", result, err)
	}
}

func TestQuoteRequestRejectsDisconnectedAndReplacedSessions(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		io.WriteString(w, `{"code":0,"data":{}}`)
	}))
	defer server.Close()
	client := NewClient(&memoryStore{data: map[string][]byte{}})
	client.baseURL = server.URL
	if err := client.Request(context.Background(), "/quote/static-info", nil, &struct{}{}); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("disconnected error=%v", err)
	}
	if err := client.Save(testTokens()); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() { result <- client.Request(context.Background(), "/quote/static-info", nil, &struct{}{}) }()
	<-started
	if err := client.Disconnect(); err != nil {
		t.Fatal(err)
	}
	close(release)
	if err := <-result; !errors.Is(err, ErrSessionChanged) {
		t.Fatalf("late result after disconnect: %v", err)
	}
}

func TestQuoteErrorsDoNotLeakCredentialsOrMisclassifyEntitlements(t *testing.T) {
	for _, unauthorized := range []bool{false, true} {
		t.Run(map[bool]string{false: "entitlement", true: "authorization"}[unauthorized], func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if unauthorized {
					w.WriteHeader(401)
				} else {
					w.WriteHeader(403)
				}
				io.WriteString(w, `{"code":301403,"message":"denied old-access old-refresh"}`)
			}))
			defer server.Close()
			client := NewClient(&memoryStore{data: map[string][]byte{}})
			client.baseURL = server.URL
			client.Save(testTokens())
			err := client.Request(context.Background(), "/quote/static-info", nil, &struct{}{})
			var apiError *APIError
			if !errors.As(err, &apiError) || errors.Is(err, ErrAuthorization) != unauthorized || strings.Contains(err.Error(), "old-access") || strings.Contains(err.Error(), "old-refresh") {
				t.Fatalf("incorrect or unsafe error=%v", err)
			}
		})
	}
}
