package app

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"trajon/internal/longbridge"
	"trajon/internal/market"
)

type longbridgeClient interface {
	Restore(context.Context) (longbridge.Status, error)
	Authorize(context.Context, func(string), string) (longbridge.Tokens, error)
	Save(longbridge.Tokens) error
	Disconnect() error
}

type longbridgeJob uint8

const (
	longbridgeIdle longbridgeJob = iota
	longbridgeRestoring
	longbridgeAuthorizing
	longbridgeSaving
	longbridgeDisconnecting
	longbridgeOpenBrowser
)

type longbridgeState struct {
	client     longbridgeClient
	status     longbridge.Status
	job        longbridgeJob
	failedJob  longbridgeJob
	err        error
	pending    longbridge.Tokens // Only retained for retrying a failed secure save.
	button     ui.Handle
	openURL    func(string)
	generation uint64
	cancel     context.CancelFunc
	results    chan longbridgeResult
	closed     bool
}

type longbridgeResult struct {
	generation uint64
	kind       longbridgeJob
	status     longbridge.Status
	tokens     longbridge.Tokens
	url        string
	err        error
}

// MyGo scopes the encryption key to the app identifier (including .dev).
// Payloads are authenticated ciphertext under the app's own data directory;
// neither plaintext grants nor keys are written into configuration JSON.
type longbridgeSecrets struct{}

func newLongbridgeClient() *longbridge.Client {
	vault := longbridgeSecrets{}
	legacy := longbridge.NewChunkedStore(vault, mygo.MaxSecretSize)
	store := longbridge.NewEncryptedStore(vault, legacy, func() (string, error) {
		dir, err := mygo.App.Path(mygo.PathUserData)
		if err != nil {
			return "", err
		}
		name := "longbridge"
		if !mygo.App.IsPackaged() {
			// go run defaults to "trajon", which can alias packaged "Trajon"
			// on case-insensitive macOS disks despite different vault scopes.
			name = "longbridge-standalone"
		}
		return filepath.Join(dir, name), nil
	})
	return longbridge.NewClient(store)
}

func (longbridgeSecrets) Get(key string) ([]byte, error) {
	data, err := mygo.Secrets.Get(key)
	if errors.Is(err, mygo.ErrSecretNotFound) {
		return nil, nil
	}
	return data, err
}
func (longbridgeSecrets) Set(key string, data []byte) error { return mygo.Secrets.Set(key, data) }
func (longbridgeSecrets) Delete(key string) error           { return mygo.Secrets.Delete(key) }

func (a *application) openDataSourceSettingsFromPicker() {
	a.stopInstrumentSearch()
	a.instrumentPickerOpen = false
	a.settingsReturnToPicker = true
	a.settingsOpen = true
	a.restoreLongbridge()
}

func (a *application) longbridgeAccessChanged() {
	if a.marketLoader != nil {
		a.marketLoader.InvalidateSourceCache(market.DataSourceLongbridge)
		if a.longbridge.status.Authorized && !a.chart.replayLocked() && a.pendingCandleLoad != nil &&
			a.retryMarketAfterAuthorization == a.pendingCandleLoad.CacheKey() {
			a.marketLoader.RetryLatest()
		}
	}
	a.retryMarketAfterAuthorization = ""
	if a.instrumentPickerOpen && a.instrumentPickerQuery != "" {
		a.searchInstruments()
	}
}

func (a *application) restoreLongbridge() {
	client := a.longbridge.client
	if client == nil || a.longbridge.closed || a.longbridge.job != longbridgeIdle {
		return
	}
	a.runLongbridgeJob(longbridgeRestoring, func(ctx context.Context, publish func(longbridgeResult)) {
		status, err := client.Restore(ctx)
		publish(longbridgeResult{kind: longbridgeRestoring, status: status, err: err})
	})
}

func (a *application) authorizeLongbridge(openURL func(string)) {
	s := &a.longbridge
	if s.client == nil || s.closed || s.job != longbridgeIdle {
		return
	}
	s.openURL = openURL
	s.pending = longbridge.Tokens{}
	client, message := s.client, a.text().LongbridgeBrowserReturn()
	a.runLongbridgeJob(longbridgeAuthorizing, func(ctx context.Context, publish func(longbridgeResult)) {
		tokens, err := client.Authorize(ctx, func(url string) {
			publish(longbridgeResult{kind: longbridgeOpenBrowser, url: url})
		}, message)
		publish(longbridgeResult{kind: longbridgeAuthorizing, tokens: tokens, err: err})
	})
}

func (a *application) saveLongbridge() {
	s := &a.longbridge
	client, tokens := s.client, s.pending
	a.runLongbridgeJob(longbridgeSaving, func(_ context.Context, publish func(longbridgeResult)) {
		err := client.Save(tokens)
		publish(longbridgeResult{kind: longbridgeSaving, err: err})
	})
}

func (a *application) disconnectLongbridge() {
	s := &a.longbridge
	if s.client == nil || s.closed || s.job != longbridgeIdle {
		return
	}
	client := s.client
	s.pending = longbridge.Tokens{}
	a.runLongbridgeJob(longbridgeDisconnecting, func(_ context.Context, publish func(longbridgeResult)) {
		publish(longbridgeResult{kind: longbridgeDisconnecting, err: client.Disconnect()})
	})
}

func (a *application) runLongbridgeJob(kind longbridgeJob, work func(context.Context, func(longbridgeResult))) {
	s := &a.longbridge
	if s.cancel != nil {
		s.cancel()
	}
	s.generation++
	s.job, s.err = kind, nil
	if s.results == nil {
		s.results = make(chan longbridgeResult, 16)
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	generation, window, results := s.generation, a.window, s.results
	publish := func(result longbridgeResult) {
		if ctx.Err() != nil {
			return
		}
		result.generation = generation
		if window != nil {
			window.Update(func() { a.applyLongbridgeResult(result) })
			return
		}
		select {
		case results <- result:
		case <-ctx.Done():
		}
	}
	go func() {
		defer cancel()
		job, stop := context.WithTimeout(ctx, 5*time.Minute)
		defer stop()
		work(job, publish)
	}()
}

func (a *application) applyLongbridgeResult(result longbridgeResult) {
	s := &a.longbridge
	if s.closed || result.generation != s.generation || s.job == longbridgeIdle {
		return
	}
	if result.kind == longbridgeOpenBrowser {
		if s.job == longbridgeAuthorizing && a.settingsOpen && s.openURL != nil {
			s.openURL(result.url)
		}
		return
	}
	if result.kind != s.job {
		return
	}
	s.job, s.err, s.failedJob = longbridgeIdle, result.err, result.kind
	s.openURL = nil
	if result.kind == longbridgeRestoring {
		s.status = result.status
	}
	if result.err != nil {
		return
	}
	switch result.kind {
	case longbridgeAuthorizing:
		s.pending = result.tokens
		// Cancellation/dismissal can reject the grant until this point. Once
		// saving starts it must finish atomically, even if settings is closed.
		a.saveLongbridge()
	case longbridgeSaving:
		s.pending = longbridge.Tokens{}
		s.status = longbridge.Status{Authorized: true}
		a.longbridgeAccessChanged()
	case longbridgeDisconnecting:
		s.status = longbridge.Status{}
		a.longbridgeAccessChanged()
	}
}

func (a *application) drainLongbridgeResults() {
	for a.longbridge.results != nil {
		select {
		case result := <-a.longbridge.results:
			a.applyLongbridgeResult(result)
		default:
			return
		}
	}
}

func (a *application) cancelLongbridgeAuthorization() {
	s := &a.longbridge
	if s.job != longbridgeAuthorizing {
		return
	}
	s.generation++
	if s.cancel != nil {
		s.cancel()
	}
	s.job, s.err, s.openURL = longbridgeIdle, nil, nil
	s.button.Focus()
}

func (a *application) closeLongbridge() {
	a.cancelLongbridgeAuthorization()
	s := &a.longbridge
	s.closed = true
	s.generation++
	if s.cancel != nil {
		s.cancel()
	}
	s.pending = longbridge.Tokens{}
	s.openURL = nil
}

func (a *application) retryLongbridge(openURL func(string)) {
	s := &a.longbridge
	if s.job != longbridgeIdle || s.closed {
		return
	}
	if s.status.NeedsAuthorization || errors.Is(s.err, longbridge.ErrAuthorization) {
		a.authorizeLongbridge(openURL)
		return
	}
	switch s.failedJob {
	case longbridgeRestoring:
		a.restoreLongbridge()
	case longbridgeSaving:
		a.saveLongbridge()
	case longbridgeDisconnecting:
		a.disconnectLongbridge()
	default:
		a.authorizeLongbridge(openURL)
	}
}

func (a *application) longbridgeSettings(c *ui.Context) {
	s, text := &a.longbridge, a.text()
	services := c.Services()
	ui.Box(c).FillWidth().Height(1).Background(c.Theme().Border)
	components.Text(c, text.DataSources()).FontSize(13).TextColor(c.Theme().TextMuted)
	ui.Column(c.Key("longbridge-settings")).FillWidth().Padding(12).Gap(12).
		Border(1, c.Theme().Border).Radius(c.Theme().Radius).Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(12).Children(func() {
			components.Text(c, text.Longbridge()).Bold().SingleLine()
			ui.Spacer(c)
			status := text.LongbridgeNotAuthorized()
			if s.status.NeedsAuthorization {
				status = text.LongbridgeAuthorizationExpired()
			} else if s.status.Authorized {
				status = text.LongbridgeAuthorized()
			}
			components.Text(c, status).FontSize(13).TextColor(c.Theme().TextMuted).SingleLine()
		})
		if s.job != longbridgeIdle {
			ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
				label := text.LongbridgeChecking()
				switch s.job {
				case longbridgeAuthorizing:
					label = text.LongbridgeWaiting()
				case longbridgeSaving:
					label = text.LongbridgeSaving()
				case longbridgeDisconnecting:
					label = text.LongbridgeDisconnecting()
				}
				components.Spinner(c.Key("longbridge-progress"), components.SpinnerRing).Size(14, 14).Label(label)
				components.Text(c, label).FontSize(13).TextColor(c.Theme().TextMuted).Wrap().Grow(1).MinWidth(0)
				if s.job == longbridgeAuthorizing {
					components.Button(c.Key("longbridge-cancel"), text.Cancel(), components.ButtonSecondary).
						OnClick(a.cancelLongbridgeAuthorization)
				}
			})
		} else {
			ui.Row(c).FillWidth().Justify(ui.End).Gap(8).Children(func() {
				label, variant := text.LongbridgeConnect(), components.ButtonPrimary
				if s.status.Authorized || s.status.NeedsAuthorization {
					label, variant = text.LongbridgeReauthorize(), components.ButtonSecondary
				}
				components.Button(c.Key("longbridge-connect"), label, variant).
					Bind(&s.button).Disabled(s.client == nil).
					OnClick(func() { a.authorizeLongbridge(services.OpenURL) })
				if s.status.Authorized || s.status.NeedsAuthorization {
					components.Button(c.Key("longbridge-disconnect"), text.LongbridgeDisconnect(), components.ButtonGhost).
						OnClick(a.disconnectLongbridge)
				}
			})
		}
		// An expired grant already has a status and a reauthorization action;
		// do not repeat the same message and action in a second error row.
		if s.err != nil && !(s.status.NeedsAuthorization && errors.Is(s.err, longbridge.ErrAuthorization)) {
			ui.Row(c.Key("longbridge-error")).FillWidth().Gap(8).AlignItems(ui.Center).
				Transition(ui.ElementTransition{Duration: 120 * time.Millisecond, Enter: &ui.Motion{Opacity: 0}}).
				Children(func() {
					components.Text(c, a.longbridgeErrorMessage()).FontSize(13).TextColor(c.Theme().TextMuted).
						Tooltip(s.err.Error()).Wrap().Grow(1).MinWidth(0)
					components.Button(c.Key("longbridge-retry"), text.Retry(), components.ButtonSecondary).
						Disabled(s.job != longbridgeIdle).OnClick(func() { a.retryLongbridge(services.OpenURL) })
				})
		}
	})
}

func (a *application) longbridgeErrorMessage() string {
	text := a.text()
	switch {
	case errors.Is(a.longbridge.err, longbridge.ErrStorage):
		return text.LongbridgeStorageFailed()
	case errors.Is(a.longbridge.err, longbridge.ErrCallback):
		return text.LongbridgeCallbackFailed()
	case errors.Is(a.longbridge.err, longbridge.ErrDenied):
		return text.LongbridgeAuthorizationDenied()
	case errors.Is(a.longbridge.err, longbridge.ErrAuthorization):
		return text.LongbridgeAuthorizationExpired()
	case errors.Is(a.longbridge.err, context.DeadlineExceeded):
		return text.LongbridgeAuthorizationTimedOut()
	default:
		return text.LongbridgeConnectionFailed()
	}
}
