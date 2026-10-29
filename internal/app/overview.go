package app

import (
	"context"
	"time"

	"trajon/internal/journal"
)

type overviewRequest struct {
	revision        uint64
	period          int
	instrument, day string
}

type overviewState struct {
	period                   int // This month, last month, this year, all time.
	instrument               string
	request                  overviewRequest
	generation               uint64
	started, closed, loading bool
	cancel                   context.CancelFunc
	result                   *journal.Performance
	selectedDay              string
	heatmapYear              int
	byTag, showIssues        bool
	trendIndex               int
}

func overviewRange(period int, now time.Time) (time.Time, time.Time) {
	month := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	switch period {
	case 0:
		return month, month.AddDate(0, 1, 0)
	case 1:
		return month.AddDate(0, -1, 0), month
	case 2:
		start := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		return start, start.AddDate(1, 0, 0)
	default:
		return time.Time{}, time.Time{}
	}
}

func (a *application) overviewRequest(now time.Time) overviewRequest {
	return overviewRequest{revision: a.records.revision, period: a.overview.period, instrument: a.overview.instrument, day: now.Format("2006-01-02 MST -0700")}
}

// Like indicator calculations, native statistics use immutable snapshots off
// the UI thread. Standalone/headless views compute synchronously.
func (a *application) ensureOverview(now time.Time) {
	s := &a.overview
	if s.closed || !a.records.ready {
		return
	}
	request := a.overviewRequest(now)
	if s.started && s.request == request {
		return
	}
	if s.cancel != nil {
		s.cancel()
	}
	s.request, s.started, s.loading = request, true, true
	s.selectedDay, s.trendIndex = "", -1
	s.generation++
	generation := s.generation
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	from, before := overviewRange(s.period, now)
	filter := journal.PerformanceFilter{From: from, Before: before, Instrument: s.instrument, Location: now.Location()}
	book := a.records.book.Clone()
	if a.window == nil {
		a.applyOverview(generation, request, journal.CalculatePerformance(ctx, book, filter))
		return
	}
	window := a.window
	go func() {
		result := journal.CalculatePerformance(ctx, book, filter)
		if ctx.Err() == nil {
			window.Update(func() { a.applyOverview(generation, request, result) })
		}
	}()
}

func (a *application) applyOverview(generation uint64, request overviewRequest, result journal.Performance) {
	s := &a.overview
	if s.closed || a.records.closed || generation != s.generation || request != a.overviewRequest(time.Now()) {
		return
	}
	s.result, s.loading = &result, false
}

func (a *application) closeOverview() {
	a.overview.closed = true
	if a.overview.cancel != nil {
		a.overview.cancel()
	}
}

func (a *application) openOverviewRecord(id string) {
	a.activePage = "journal"
	a.openJournalRecord(id)
}
