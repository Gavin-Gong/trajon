package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/egoist/mygo"
)

func receiveUpdateResult(t *testing.T, app *application, kind updateResultKind) {
	t.Helper()
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	for {
		select {
		case result := <-app.updates.results:
			app.applyUpdateResult(result)
			if result.kind == kind {
				return
			}
		case <-timer.C:
			t.Fatalf("timed out waiting for update result %d", kind)
		}
	}
}

func TestUpdateCheckResults(t *testing.T) {
	for _, tc := range []struct {
		name      string
		available *availableUpdate
		err       error
		wantError bool
	}{
		{name: "current"},
		{name: "new", available: &availableUpdate{version: "0.0.2", notes: "## Fixes\n\n- Chart zoom"}},
		{name: "disabled", err: mygo.ErrUpdatesDisabled},
		{name: "offline", err: errors.New("offline"), wantError: true},
		{name: "deadline", err: context.DeadlineExceeded, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := &application{updates: updateState{check: func(context.Context) (*availableUpdate, error) {
				return tc.available, tc.err
			}}}
			t.Cleanup(app.closeUpdates)
			app.checkForUpdates()
			if !app.updates.checking {
				t.Fatal("check did not enter loading state")
			}
			receiveUpdateResult(t, app, updateChecked)
			if app.updates.checking || (app.updates.checkError != nil) != tc.wantError || app.updates.available != tc.available {
				t.Fatalf("unexpected check result: %+v", &app.updates)
			}
		})
	}
}

func TestUpdateCheckDoesNotDuplicateAndCanRetry(t *testing.T) {
	started, release := make(chan struct{}, 2), make(chan struct{})
	app := &application{updates: updateState{check: func(ctx context.Context) (*availableUpdate, error) {
		started <- struct{}{}
		select {
		case <-release:
			return nil, errors.New("offline")
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}}}
	t.Cleanup(app.closeUpdates)
	app.checkForUpdates()
	<-started
	generation := app.updates.generation
	app.checkForUpdates()
	if app.updates.generation != generation {
		t.Fatal("duplicate check started while already checking")
	}
	close(release)
	receiveUpdateResult(t, app, updateChecked)
	app.updates.check = func(context.Context) (*availableUpdate, error) { return nil, nil }
	app.checkForUpdates()
	if app.updates.checkError != nil || !app.updates.checking {
		t.Fatal("retry did not clear the previous error")
	}
	receiveUpdateResult(t, app, updateChecked)
	if app.updates.checkError != nil || app.updates.checking {
		t.Fatal("successful retry retained the previous error")
	}
}

func TestUpdateInstallProgressRetryAndRelaunch(t *testing.T) {
	started := make(chan struct{}, 2)
	release := make(chan error, 2)
	relaunches := 0
	app := &application{updates: updateState{
		relaunch: func() { relaunches++ },
		available: &availableUpdate{version: "0.0.2", install: func(ctx context.Context, progress func(int64, int64)) error {
			started <- struct{}{}
			progress(25, 100)
			progress(10, 200) // A failed delta falls back to the full archive.
			select {
			case err := <-release:
				return err
			case <-ctx.Done():
				return ctx.Err()
			}
		}},
	}}
	t.Cleanup(app.closeUpdates)
	for attempt := range 2 {
		app.installUpdate()
		<-started
		generation := app.updates.generation
		app.installUpdate()
		app.checkForUpdates()
		if !app.updates.installing || app.updates.generation != generation {
			t.Fatal("installation was duplicated or interrupted by a check")
		}
		receiveUpdateResult(t, app, updateProgress)
		if app.updates.downloaded != 25 || app.updates.total != 100 {
			t.Fatal("download progress was not applied")
		}
		receiveUpdateResult(t, app, updateProgress)
		if app.updates.downloaded != 10 || app.updates.total != 200 {
			t.Fatal("delta fallback progress was not reset")
		}
		if attempt == 0 {
			release <- errors.New("signature mismatch")
		} else {
			release <- nil
		}
		receiveUpdateResult(t, app, updateInstalled)
		if attempt == 0 {
			if app.updates.installError == nil || app.updates.installing || relaunches != 0 {
				t.Fatal("failed install did not offer retry, or relaunched")
			}
		} else if app.updates.installError != nil || !app.updates.installed || relaunches != 1 {
			t.Fatal("successful retry did not relaunch exactly once")
		}
	}
	app.applyUpdateResult(updateResult{generation: app.updates.generation, kind: updateInstalled})
	app.installUpdate()
	if relaunches != 1 {
		t.Fatal("duplicate completion relaunched again")
	}
}

func TestUpdateJobsCancelOnWindowCloseAndRejectLateResults(t *testing.T) {
	for _, installing := range []bool{false, true} {
		t.Run(map[bool]string{false: "check", true: "install"}[installing], func(t *testing.T) {
			started, stopped := make(chan struct{}), make(chan struct{})
			work := func(ctx context.Context) error {
				close(started)
				<-ctx.Done()
				close(stopped)
				return ctx.Err()
			}
			relaunched := false
			app := &application{updates: updateState{
				check:    func(ctx context.Context) (*availableUpdate, error) { return nil, work(ctx) },
				relaunch: func() { relaunched = true },
			}}
			if installing {
				app.updates.available = &availableUpdate{install: func(ctx context.Context, _ func(int64, int64)) error { return work(ctx) }}
				app.installUpdate()
			} else {
				app.checkForUpdates()
			}
			<-started
			oldGeneration := app.updates.generation
			app.closeUpdates()
			select {
			case <-stopped:
			case <-time.After(3 * time.Second):
				t.Fatal("closing the window did not cancel background work")
			}
			app.applyUpdateResult(updateResult{kind: updateChecked, generation: oldGeneration, available: &availableUpdate{version: "stale"}})
			app.applyUpdateResult(updateResult{kind: updateInstalled, generation: oldGeneration})
			app.checkForUpdates()
			app.installUpdate()
			if app.updates.checking || app.updates.installing || relaunched || (app.updates.available != nil && app.updates.available.version == "stale") {
				t.Fatal("closed window accepted late results or restarted work")
			}
		})
	}
}

func TestUpdateResultsOnlyApplyOnUIThreadAndRejectOldGeneration(t *testing.T) {
	app := &application{updates: updateState{check: func(context.Context) (*availableUpdate, error) {
		return &availableUpdate{version: "0.0.2", notes: "- Fixed replay"}, nil
	}}}
	t.Cleanup(app.closeUpdates)
	app.checkForUpdates()
	var result updateResult
	select {
	case result = <-app.updates.results:
	case <-time.After(3 * time.Second):
		t.Fatal("check did not publish")
	}
	if !app.updates.checking || app.updates.available != nil {
		t.Fatal("worker mutated UI state before the result was applied")
	}
	old := result
	old.generation--
	app.applyUpdateResult(old)
	if !app.updates.checking || app.updates.available != nil {
		t.Fatal("stale check result was applied")
	}
	app.applyUpdateResult(result)
	if app.updates.checking || app.updates.available == nil {
		t.Fatal("current check result was not applied")
	}
	app.updates.installing = true
	app.applyUpdateResult(updateResult{kind: updateInstalled, generation: old.generation})
	app.applyUpdateResult(updateResult{kind: updateProgress, generation: old.generation, downloaded: 100, total: 100})
	if !app.updates.installing || app.updates.downloaded != 0 || app.updates.installed {
		t.Fatal("old installation result changed the current job")
	}
}

func TestUpdateJobTimeoutIsPublished(t *testing.T) {
	app := &application{updates: updateState{checking: true}}
	t.Cleanup(app.closeUpdates)
	app.runUpdateJob(time.Millisecond, func(ctx context.Context, publish func(updateResult)) {
		<-ctx.Done()
		publish(updateResult{kind: updateChecked, err: ctx.Err()})
	})
	receiveUpdateResult(t, app, updateChecked)
	if app.updates.checking || !errors.Is(app.updates.checkError, context.DeadlineExceeded) {
		t.Fatal("timeout left checking stuck instead of publishing an error")
	}
}
