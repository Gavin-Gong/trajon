package longbridge

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

type memoryStore struct {
	data     map[string][]byte
	failSave bool
}

func (s *memoryStore) Get(key string) ([]byte, error) { return s.data[key], nil }
func (s *memoryStore) Set(key string, value []byte) error {
	if s.failSave && key == tokensKey {
		return errors.New("keychain locked")
	}
	s.data[key] = append([]byte(nil), value...)
	return nil
}
func (s *memoryStore) Delete(key string) error { delete(s.data, key); return nil }

func testTokens() Tokens {
	return Tokens{ClientID: "client", AccessToken: "old-access", RefreshToken: "old-refresh", ExpiresAt: time.Now().Add(time.Hour)}
}

func TestBrowserAuthorizationChecksStateAndPKCEBeforeCommit(t *testing.T) {
	store := &memoryStore{data: map[string][]byte{}}
	var challenge, redirect string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/register":
			var body struct {
				Name      string   `json:"client_name"`
				Redirects []string `json:"redirect_uris"`
				Method    string   `json:"token_endpoint_auth_method"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if body.Name != "Trajon" || body.Method != "none" || len(body.Redirects) != 1 {
				t.Errorf("registration: %+v", body)
			}
			redirect = body.Redirects[0]
			io.WriteString(w, `{"client_id":"client","scope":"quote"}`)
		case "/oauth2/token":
			if err := r.ParseForm(); err != nil {
				t.Error(err)
			}
			digest := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
			if len(r.Form.Get("code_verifier")) < 43 || base64.RawURLEncoding.EncodeToString(digest[:]) != challenge ||
				r.Form.Get("code") != "approved-code" || r.Form.Get("redirect_uri") != redirect || r.Form.Get("client_id") != "client" {
				t.Errorf("incorrect PKCE exchange")
			}
			io.WriteString(w, `{"access_token":"new-access","refresh_token":"new-refresh","expires_in":3600,"token_type":"Bearer"}`)
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	client := NewClient(store)
	client.baseURL, client.address = server.URL, "127.0.0.1:0"
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	tokens, err := client.Authorize(ctx, func(raw string) {
		u, _ := url.Parse(raw)
		q := u.Query()
		challenge = q.Get("code_challenge")
		if q.Get("code_challenge_method") != "S256" || q.Get("scope") != "quote" || q.Get("response_type") != "code" {
			t.Errorf("authorization query: %v", q)
		}
		callback, _ := url.Parse(q.Get("redirect_uri"))
		for _, state := range []string{"wrong-state", q.Get("state")} {
			callback.RawQuery = url.Values{"state": {state}, "code": {"approved-code"}}.Encode()
			res, err := http.Get(callback.String())
			if err != nil {
				t.Error(err)
				return
			}
			body, _ := io.ReadAll(res.Body)
			res.Body.Close()
			if state == "wrong-state" {
				if res.StatusCode != 400 {
					t.Errorf("wrong-state response: %d", res.StatusCode)
				}
			} else if res.StatusCode != 200 || !strings.Contains(string(body), "Return &lt;Trajon&gt;") || res.Header.Get("Cache-Control") != "no-store" {
				t.Errorf("callback page: %d %s", res.StatusCode, body)
			}
		}
	}, "Return <Trajon>")
	if err != nil {
		t.Fatal(err)
	}
	if tokens.AccessToken != "new-access" || tokens.RefreshToken != "new-refresh" || len(store.data[tokensKey]) != 0 {
		t.Fatal("grant was incorrect or committed before the UI accepted it")
	}
	if err := client.Save(tokens); err != nil {
		t.Fatal(err)
	}
	if status, err := client.Restore(ctx); err != nil || !status.Authorized {
		t.Fatalf("restored status=%+v, err=%v", status, err)
	}
	if err := client.Disconnect(); err != nil {
		t.Fatal(err)
	}
	if len(store.data[tokensKey]) != 0 || len(store.data[clientKey]) == 0 {
		t.Fatal("disconnect removed registration or retained tokens")
	}
}

func TestCanceledAndDeclinedAuthorizationKeepPreviousGrant(t *testing.T) {
	for _, decline := range []bool{false, true} {
		t.Run(map[bool]string{false: "cancel", true: "decline"}[decline], func(t *testing.T) {
			store := &memoryStore{data: map[string][]byte{}}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/oauth2/register" {
					t.Error("canceled flow exchanged a token")
				}
				io.WriteString(w, `{"client_id":"client"}`)
			}))
			defer server.Close()
			client := NewClient(store)
			client.baseURL, client.address = server.URL, "127.0.0.1:0"
			if err := client.Save(testTokens()); err != nil {
				t.Fatal(err)
			}
			before := string(store.data[tokensKey])
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_, err := client.Authorize(ctx, func(raw string) {
				if !decline {
					cancel()
					return
				}
				u, _ := url.Parse(raw)
				q := u.Query()
				res, err := http.Get(q.Get("redirect_uri") + "?" + url.Values{"state": {q.Get("state")}, "error": {"access_denied"}}.Encode())
				if err != nil {
					t.Error(err)
					return
				}
				res.Body.Close()
			}, "Return to Trajon")
			want := context.Canceled
			if decline {
				want = ErrDenied
			}
			if !errors.Is(err, want) || string(store.data[tokensKey]) != before {
				t.Fatalf("err=%v; previous grant changed", err)
			}
		})
	}
}

func TestRefreshRotationSurvivesSaveFailureAndConcurrentRestore(t *testing.T) {
	store := &memoryStore{data: map[string][]byte{}}
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		r.ParseForm()
		if r.Form.Get("refresh_token") != "old-refresh" {
			t.Error("wrong refresh token")
		}
		io.WriteString(w, `{"access_token":"rotated-access","refresh_token":"rotated-refresh","expires_in":3600}`)
	}))
	defer server.Close()
	client := NewClient(store)
	client.baseURL = server.URL
	tokens := testTokens()
	tokens.ExpiresAt = time.Now().Add(30 * time.Second)
	if err := client.Save(tokens); err != nil {
		t.Fatal(err)
	}
	store.failSave = true
	if _, err := client.Restore(context.Background()); !errors.Is(err, ErrStorage) {
		t.Fatalf("save failure: %v", err)
	}
	store.failSave = false
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			if status, err := client.Restore(context.Background()); err != nil || !status.Authorized {
				t.Errorf("retry status=%+v, err=%v", status, err)
			}
		})
	}
	wg.Wait()
	var restored Tokens
	json.Unmarshal(store.data[tokensKey], &restored)
	if requests != 1 || restored.RefreshToken != "rotated-refresh" {
		t.Fatal("retry reused a dead refresh token or lost its rotation")
	}
}

func TestRestoreDistinguishesExpiredGrantFromNetworkFailure(t *testing.T) {
	for _, rejected := range []bool{false, true} {
		t.Run(map[bool]string{false: "network", true: "rejected"}[rejected], func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if rejected {
					w.WriteHeader(400)
					io.WriteString(w, `{"error":"invalid_grant","detail":"secret-echo"}`)
				} else {
					w.WriteHeader(503)
					io.WriteString(w, "secret-echo")
				}
			}))
			defer server.Close()
			store := &memoryStore{data: map[string][]byte{}}
			client := NewClient(store)
			client.baseURL = server.URL
			tokens := testTokens()
			tokens.ExpiresAt = time.Now().Add(30 * time.Second)
			client.Save(tokens)
			status, err := client.Restore(context.Background())
			if err == nil || errors.Is(err, ErrAuthorization) != rejected || status.NeedsAuthorization != rejected || status.Authorized == rejected || strings.Contains(err.Error(), "secret-echo") {
				t.Fatalf("status=%+v, err=%v", status, err)
			}
		})
	}
}

func TestCanceledListenerClosesBeforeImmediateRetry(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	_, port, _ := net.SplitHostPort(address)
	client := NewClient(&memoryStore{data: map[string][]byte{}})
	client.address = address
	if err := client.write(clientKey, registration{ClientID: "client", RedirectURI: "http://localhost:" + port + "/oauth/callback"}); err != nil {
		t.Fatal(err)
	}
	first, cancelFirst := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelFirst()
	opened, finished := make(chan struct{}), make(chan error, 1)
	go func() {
		_, err := client.Authorize(first, func(string) { close(opened) }, "Return")
		finished <- err
	}()
	select {
	case <-opened:
	case <-first.Done():
		t.Fatal("first listener did not open")
	}
	cancelFirst()
	second, cancelSecond := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelSecond()
	_, err = client.Authorize(second, func(string) { cancelSecond() }, "Return")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("retry failed before it could open: %v", err)
	}
	if err := <-finished; !errors.Is(err, context.Canceled) {
		t.Fatalf("first cancellation: %v", err)
	}
}

func TestBusyCallbackPortDoesNotOpenBrowser(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	client := NewClient(&memoryStore{data: map[string][]byte{}})
	client.address = listener.Addr().String()
	_, err = client.Authorize(context.Background(), func(string) { t.Error("opened browser without callback listener") }, "Return")
	if !errors.Is(err, ErrCallback) {
		t.Fatalf("err=%v", err)
	}
}
