package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"trajon/internal/longbridge"
)

type testLongbridgeClient struct {
	grant     chan longbridge.Tokens
	status    longbridge.Status
	saveError error
	saved     longbridge.Tokens
}

func (c *testLongbridgeClient) Restore(context.Context) (longbridge.Status, error) {
	return c.status, nil
}
func (c *testLongbridgeClient) Authorize(ctx context.Context, open func(string), _ string) (longbridge.Tokens, error) {
	open("https://openapi.longbridge.com/oauth2/authorize")
	select {
	case tokens := <-c.grant:
		return tokens, nil
	case <-ctx.Done():
		return longbridge.Tokens{}, ctx.Err()
	}
}
func (c *testLongbridgeClient) Save(tokens longbridge.Tokens) error {
	if c.saveError != nil {
		return c.saveError
	}
	c.saved, c.status = tokens, longbridge.Status{Authorized: true}
	return nil
}
func (c *testLongbridgeClient) Disconnect() error {
	c.saved, c.status = longbridge.Tokens{}, longbridge.Status{}
	return nil
}

func waitLongbridge(t *testing.T, app *application, ready func() bool) {
	t.Helper()
	timeout := time.After(3 * time.Second)
	for !ready() {
		select {
		case result := <-app.longbridge.results:
			app.applyLongbridgeResult(result)
		case <-timeout:
			t.Fatal("timed out waiting for authorization work")
		}
	}
}

func TestLongbridgeSettingsSaveFailureRetriesWithoutAuthorizingAgain(t *testing.T) {
	app, tt := newLocaleSettingsTester()
	client := &testLongbridgeClient{grant: make(chan longbridge.Tokens, 1), saveError: longbridge.ErrStorage}
	app.longbridge.client = client
	t.Cleanup(app.closeLongbridge)
	clickReplaySetting(t, tt, "Settings")
	waitLongbridge(t, app, func() bool { return app.longbridge.job == longbridgeIdle })
	tt.Frame()
	clickReplaySetting(t, tt, "Connect via browser")
	waitLongbridge(t, app, func() bool { return len(tt.OpenedURLs()) == 1 }) // UI-thread browser opening.
	tt.Frame()
	if len(tt.OpenedURLs()) != 1 || !tt.HasText("Waiting for browser authorization…") {
		t.Fatal("authorization did not open the browser or show its pending state")
	}
	client.grant <- longbridge.Tokens{ClientID: "client", AccessToken: "access", RefreshToken: "refresh", ExpiresAt: time.Now().Add(time.Hour)}
	waitLongbridge(t, app, func() bool { return app.longbridge.job == longbridgeIdle })
	tt.Frame()
	if app.longbridge.status.Authorized || !tt.HasText("Could not access the system credential store") || app.longbridge.pending.AccessToken == "" {
		t.Fatal("failed save claimed success or discarded the retryable grant")
	}
	client.saveError = nil
	clickReplaySetting(t, tt, "Retry")
	waitLongbridge(t, app, func() bool { return app.longbridge.job == longbridgeIdle })
	tt.Frame()
	if !app.longbridge.status.Authorized || app.longbridge.pending.AccessToken != "" || client.saved.AccessToken != "access" || len(tt.OpenedURLs()) != 1 {
		t.Fatal("retry lost the grant or unnecessarily opened another authorization flow")
	}
	clickReplaySetting(t, tt, "Disconnect")
	waitLongbridge(t, app, func() bool { return app.longbridge.job == longbridgeIdle })
	tt.Frame()
	if app.longbridge.status.Authorized || client.saved.AccessToken != "" {
		t.Fatal("disconnect retained authorization")
	}
}

func TestLongbridgeSettingsDismissalRejectsLateGrantsAndRestoresFocus(t *testing.T) {
	app, tt := newLocaleSettingsTester()
	client := &testLongbridgeClient{grant: make(chan longbridge.Tokens, 1)}
	app.longbridge.client = client
	t.Cleanup(app.closeLongbridge)
	clickReplaySetting(t, tt, "Settings")
	waitLongbridge(t, app, func() bool { return app.longbridge.job == longbridgeIdle })
	tt.Frame()
	clickReplaySetting(t, tt, "Connect via browser")
	waitLongbridge(t, app, func() bool { return len(tt.OpenedURLs()) == 1 })
	generation := app.longbridge.generation
	tt.Key(0, ui.KeyEscape)
	for range 3 {
		tt.Frame()
	}
	if app.settingsOpen || app.longbridge.job != longbridgeIdle || !tt.Focused("Settings") {
		t.Fatal("dismissal retained authorization work or lost opener focus")
	}
	app.applyLongbridgeResult(longbridgeResult{generation: generation, kind: longbridgeAuthorizing, tokens: longbridge.Tokens{AccessToken: "late"}})
	app.applyLongbridgeResult(longbridgeResult{generation: generation, kind: longbridgeOpenBrowser, url: "https://example.com/late"})
	if client.saved.AccessToken != "" || app.longbridge.pending.AccessToken != "" || len(tt.OpenedURLs()) != 1 {
		t.Fatal("dismissed settings accepted a late grant or opened a late URL")
	}
	app.closeLongbridge()
	app.applyLongbridgeResult(longbridgeResult{generation: app.longbridge.generation, kind: longbridgeSaving})
	if app.longbridge.status.Authorized {
		t.Fatal("closed window accepted authorization")
	}
}

func TestLongbridgeFailedReauthorizationAndDisconnectKeepCommittedState(t *testing.T) {
	app := &application{longbridge: longbridgeState{status: longbridge.Status{Authorized: true}, generation: 1}}
	for _, kind := range []longbridgeJob{longbridgeAuthorizing, longbridgeDisconnecting} {
		app.longbridge.job = kind
		app.applyLongbridgeResult(longbridgeResult{generation: 1, kind: kind, err: errors.New("network or storage unavailable")})
		if !app.longbridge.status.Authorized || app.longbridge.job != longbridgeIdle {
			t.Fatal("failed operation destroyed the committed grant")
		}
	}
}
