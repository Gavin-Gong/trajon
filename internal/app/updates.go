package app

import (
	"context"
	"errors"
	"time"

	"github.com/egoist/mygo"
	"github.com/oneo-me/onevra/source/components/markdown"
)

const projectGitHubURL = "https://github.com/Gavin-Gong/trajon"

type availableUpdate struct {
	version string
	notes   string
	install func(context.Context, func(downloaded, total int64)) error
}

type updateState struct {
	check     func(context.Context) (*availableUpdate, error)
	relaunch  func()
	available *availableUpdate
	notes     *markdown.MarkdownDocument

	checking, installing, installed bool
	checkError, installError        error
	downloaded, total               int64
	generation                      uint64
	cancel                          context.CancelFunc
	closed                          bool
	results                         chan updateResult
}

type updateResultKind uint8

const (
	updateChecked updateResultKind = iota
	updateProgress
	updateInstalled
)

type updateResult struct {
	kind              updateResultKind
	generation        uint64
	available         *availableUpdate
	notes             *markdown.MarkdownDocument
	err               error
	downloaded, total int64
}

func checkAppUpdate(ctx context.Context) (*availableUpdate, error) {
	update, err := mygo.Updater.Check(ctx)
	if err != nil || update == nil {
		return nil, err
	}
	return &availableUpdate{version: update.Version, notes: update.Notes, install: update.Install}, nil
}

func (a *application) checkForUpdates() {
	s := &a.updates
	if s.closed || s.checking || s.installing || s.installed || s.available != nil || s.check == nil {
		return
	}
	s.checking, s.checkError = true, nil
	check := s.check
	a.runUpdateJob(30*time.Second, func(ctx context.Context, publish func(updateResult)) {
		available, err := check(ctx)
		result := updateResult{kind: updateChecked, available: available, err: err}
		if err == nil && available != nil {
			result.notes = markdown.ParseMarkdown(available.notes)
		}
		publish(result)
	})
}

func (a *application) installUpdate() {
	s := &a.updates
	if s.closed || s.installing || s.installed || s.available == nil || s.available.install == nil {
		return
	}
	s.installing, s.installError = true, nil
	s.downloaded, s.total = 0, 0
	install := s.available.install
	a.runUpdateJob(15*time.Minute, func(ctx context.Context, publish func(updateResult)) {
		// Limit redraws without losing a delta-to-full-download progress reset.
		var last time.Time
		var previousTotal int64
		err := install(ctx, func(downloaded, total int64) {
			if total == previousTotal && downloaded != total && time.Since(last) < 100*time.Millisecond {
				return
			}
			last, previousTotal = time.Now(), total
			publish(updateResult{kind: updateProgress, downloaded: downloaded, total: total})
		})
		publish(updateResult{kind: updateInstalled, err: err})
	})
}

// Workers capture their inputs and publish only through Window.Update. The
// channel is the same headless fallback used by instrument search tests.
func (a *application) runUpdateJob(timeout time.Duration, work func(context.Context, func(updateResult))) {
	s := &a.updates
	if s.cancel != nil {
		s.cancel()
	}
	s.generation++
	if s.results == nil {
		s.results = make(chan updateResult, 32)
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	generation, window, results := s.generation, a.window, s.results
	publish := func(result updateResult) {
		if ctx.Err() != nil {
			return
		}
		result.generation = generation
		if window != nil {
			window.Update(func() { a.applyUpdateResult(result) })
			return
		}
		select {
		case results <- result:
		case <-ctx.Done():
		}
	}
	go func() {
		defer cancel()
		job, stop := context.WithTimeout(ctx, timeout)
		defer stop()
		work(job, publish)
	}()
}

func (a *application) applyUpdateResult(result updateResult) {
	s := &a.updates
	if s.closed || result.generation != s.generation {
		return
	}
	switch result.kind {
	case updateChecked:
		if !s.checking {
			return
		}
		s.checking = false
		s.checkError = result.err
		if errors.Is(result.err, mygo.ErrUpdatesDisabled) {
			// Development and externally managed installs still show app info.
			s.checkError = nil
		}
		if result.err == nil {
			s.available, s.notes = result.available, result.notes
		}
	case updateProgress:
		if s.installing {
			s.downloaded, s.total = result.downloaded, result.total
		}
	case updateInstalled:
		if !s.installing {
			return
		}
		s.installing, s.installError = false, result.err
		if result.err == nil {
			s.installed = true
			if s.relaunch != nil {
				s.relaunch()
			}
		}
	}
}

func (a *application) drainUpdateResults() {
	for a.updates.results != nil {
		select {
		case result := <-a.updates.results:
			a.applyUpdateResult(result)
		default:
			return
		}
	}
}

func (a *application) closeUpdates() {
	s := &a.updates
	s.closed = true
	s.generation++
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.checking, s.installing = false, false
}
