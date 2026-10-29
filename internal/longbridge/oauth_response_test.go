package longbridge

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestTokenExchangeAcceptsProviderResponseVariants(t *testing.T) {
	for _, tc := range []struct {
		name     string
		response string
		lifetime time.Duration
	}{
		{"quoted expiry", `{"access_token":"access","refresh_token":"refresh","expires_in":"3600","token_type":"Bearer"}`, time.Hour},
		{"omitted expiry", `{"access_token":"access","refresh_token":"refresh","token_type":"Bearer"}`, time.Hour},
		{"access only", `{"access_token":"access","expires_in":3600,"token_type":"Bearer"}`, time.Hour},
		{"long lived", `{"access_token":"access","refresh_token":"refresh","expires_in":315360000,"token_type":"Bearer"}`, 10 * 365 * 24 * time.Hour},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				io.WriteString(w, tc.response)
			}))
			defer server.Close()
			store := &memoryStore{data: map[string][]byte{}}
			client := NewClient(store)
			client.baseURL = server.URL
			before := time.Now()
			tokens, err := client.exchange(context.Background(), "client", url.Values{"grant_type": {"authorization_code"}})
			if err != nil {
				t.Fatal(err)
			}
			if tokens.ClientID != "client" || tokens.AccessToken != "access" ||
				tokens.ExpiresAt.Before(before.Add(tc.lifetime)) || tokens.ExpiresAt.After(time.Now().Add(tc.lifetime)) {
				t.Fatalf("incorrect token identity or expiry")
			}
			if err := client.Save(tokens); err != nil {
				t.Fatal(err)
			}
			if status, err := NewClient(store).Restore(context.Background()); err != nil || !status.Authorized {
				t.Fatalf("restart status=%+v, err=%v", status, err)
			}
		})
	}
}

func TestRestoreAccessOnlyGrantUsesExpiryInsteadOfRefreshing(t *testing.T) {
	for _, tc := range []struct {
		name      string
		remaining time.Duration
	}{
		{"near expiry", 30 * time.Second},
		{"expired", -time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Error("attempted refresh without a refresh token")
				w.WriteHeader(http.StatusBadRequest)
			}))
			defer server.Close()
			tokens := testTokens()
			tokens.RefreshToken = ""
			tokens.ExpiresAt = time.Now().Add(tc.remaining)
			data, err := json.Marshal(tokens)
			if err != nil {
				t.Fatal(err)
			}
			client := NewClient(&memoryStore{data: map[string][]byte{tokensKey: data}})
			client.baseURL = server.URL
			status, err := client.Restore(context.Background())
			expired := tc.remaining <= 0
			if errors.Is(err, ErrAuthorization) != expired || status.Authorized == expired || status.NeedsAuthorization != expired || !expired && err != nil {
				t.Fatalf("status=%+v, err=%v", status, err)
			}
		})
	}
}

func TestTokenExchangeRejectsMalformedResponsesWithoutLeakingCredentials(t *testing.T) {
	for _, tc := range []struct {
		name     string
		response string
	}{
		{"missing access token", `{"refresh_token":"secret-refresh","expires_in":3600}`},
		{"unsupported token type", `{"access_token":"secret-access","expires_in":3600,"token_type":"MAC"}`},
		{"invalid expiry", `{"access_token":"secret-access","expires_in":"secret-echo"}`},
		{"zero expiry", `{"access_token":"secret-access","expires_in":0}`},
		{"negative expiry", `{"access_token":"secret-access","expires_in":-1}`},
		{"overflowing expiry", `{"access_token":"secret-access","expires_in":9223372037}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				io.WriteString(w, tc.response)
			}))
			defer server.Close()
			client := NewClient(&memoryStore{data: map[string][]byte{}})
			client.baseURL = server.URL
			_, err := client.exchange(context.Background(), "client", url.Values{"grant_type": {"authorization_code"}})
			if !errors.Is(err, ErrResponse) || strings.Contains(err.Error(), "secret-") {
				t.Fatalf("incorrect or unsafe error: %v", err)
			}
		})
	}
}
