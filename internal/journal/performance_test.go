package journal

import (
	"context"
	"math"
	"reflect"
	"testing"
	"time"
)

func performanceRecord(t *testing.T, book *Book, at time.Time, instrument string, executions ...Draft) *Record {
	t.Helper()
	r := book.AddRecord(at)
	r.Instrument = instrument
	r.Draft = Draft{Kind: Plan, Text: "plan"}
	if err := r.Commit(at); err != nil {
		t.Fatal(err)
	}
	for i, draft := range executions {
		r.Draft = draft
		if err := r.Commit(at.Add(time.Duration(i+1) * time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

func TestPerformanceUsesWholeTradesExactCashFlowsAndClosingTime(t *testing.T) {
	at := time.Date(2026, 1, 31, 23, 58, 0, 0, time.UTC)
	book := New(at)
	long := performanceRecord(t, &book, at, "A",
		Draft{Kind: Open, Quantity: ".1", Price: "100"},
		Draft{Kind: Add, Quantity: ".2", Price: "110"},
		Draft{Kind: Reduce, Quantity: "-.1", Price: "120"},
		Draft{Kind: Close, Quantity: "-.2", Price: "130"}) // +6, closes in February.
	long.Tags = []string{"trend", "trend", ""}
	long.Summary, long.SavedAt = "", time.Time{} // Review is irrelevant.
	performanceRecord(t, &book, at.Add(24*time.Hour), "B",
		Draft{Kind: Open, Quantity: "-2", Price: "100"},
		Draft{Kind: Close, Quantity: "2", Price: "110"}) // -20 short.
	performanceRecord(t, &book, at.Add(48*time.Hour), "A",
		Draft{Kind: Open, Quantity: ".3", Price: ".1"},
		Draft{Kind: Reduce, Quantity: "-.1", Price: ".1"},
		Draft{Kind: Close, Quantity: "-.2", Price: ".1"}) // Exactly breakeven.
	performanceRecord(t, &book, at.Add(-time.Hour), "outside period",
		Draft{Kind: Open, Quantity: "1", Price: "10"}, Draft{Kind: Close, Quantity: "-1", Price: "1000"})
	performanceRecord(t, &book, at, "holding",
		Draft{Kind: Open, Quantity: "2", Price: "10"}, Draft{Kind: Reduce, Quantity: "-1", Price: "20"})
	before := book.Clone()
	from := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	result := CalculatePerformance(context.Background(), book, PerformanceFilter{From: from, Before: from.AddDate(0, 1, 0), Location: time.UTC})
	if result.Count != 3 || result.Wins != 1 || result.Losses != 1 || result.Breakeven != 1 || result.PnL != -14 || result.MaxDrawdown != 20 {
		t.Fatalf("wrong performance: %+v", result.PerformanceSummary)
	}
	if rate, ok := result.WinRate(); !ok || rate != 1.0/3 {
		t.Fatalf("win rate=%v, %v", rate, ok)
	}
	if ratio, ok := result.PayoffRatio(); !ok || ratio != .3 {
		t.Fatalf("payoff=%v, %v", ratio, ok)
	}
	if len(result.ByTag) != 1 || result.ByTag[0].Count != 1 || result.ByTag[0].PnL != 6 {
		t.Fatalf("tags=%+v", result.ByTag)
	}
	if !reflect.DeepEqual(book, before) {
		t.Fatal("statistics mutated the journal")
	}
	if result.Days[0].Date != from || result.Days[len(result.Days)-1].Cumulative != -14 {
		t.Fatalf("days=%+v", result.Days)
	}
	// The closing boundary is exclusive, even though a record was created earlier.
	result = CalculatePerformance(context.Background(), book, PerformanceFilter{From: from, Before: long.Entries[len(long.Entries)-1].At, Location: time.UTC})
	if result.Count != 0 {
		t.Fatal("included an exclusive closing boundary")
	}
}

func TestPerformanceIncludesAllInstrumentsWithoutCurrencyAndRejectsIncompleteLogs(t *testing.T) {
	at := time.Date(2026, 3, 8, 0, 30, 0, 0, time.UTC)
	book := New(at)
	performanceRecord(t, &book, at, "legacy", Draft{Kind: Open, Quantity: "1", Price: "10"}, Draft{Kind: Close, Quantity: "-1", Price: "20"})
	invalid := performanceRecord(t, &book, at, "incomplete", Draft{Kind: Open, Quantity: "1", Price: "10"}, Draft{Kind: Close, Quantity: "-1", Price: "20"})
	invalid.Entries[1].Price = ""
	for _, instrument := range []string{"US stock", "HK stock"} {
		performanceRecord(t, &book, at, instrument, Draft{Kind: Open, Quantity: "-1", Price: "20"}, Draft{Kind: Close, Quantity: "1", Price: "10"})
	}
	result := CalculatePerformance(context.Background(), book, PerformanceFilter{})
	if result.Count != 3 || result.PnL != 30 || len(result.Issues) != 1 || result.Issues[0].ID != invalid.ID {
		t.Fatalf("excluded complete records or silently lost repairable records: %+v", result)
	}
	zone, _ := time.LoadLocation("America/New_York")
	result = CalculatePerformance(context.Background(), book, PerformanceFilter{Instrument: "US stock", Location: zone})
	if result.Count != 1 || len(result.Issues) != 0 || result.Days[0].Date.Day() != 7 || result.Days[0].Date.Hour() != 0 || result.Days[0].Date.Location() != zone {
		t.Fatalf("filter/local-day failure: %+v", result)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if cancelled := CalculatePerformance(ctx, book, PerformanceFilter{}); cancelled.Count != 0 {
		t.Fatal("cancelled job produced statistics")
	}
}

func TestPerformanceDrawdownIncludesZeroOriginAndBatchesEqualTimestamps(t *testing.T) {
	at := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	book := New(at)
	for _, closing := range []string{"5", "15"} {
		performanceRecord(t, &book, at, "A", Draft{Kind: Open, Quantity: "1", Price: "10"}, Draft{Kind: Close, Quantity: "-1", Price: closing})
	}
	result := CalculatePerformance(context.Background(), book, PerformanceFilter{})
	if result.PnL != 0 || result.MaxDrawdown != 0 || len(result.Points) != 1 || result.Points[0].Count != 2 {
		t.Fatalf("same-time trades created an artificial drawdown: %+v", result)
	}
	performanceRecord(t, &book, at.Add(time.Hour), "A", Draft{Kind: Open, Quantity: "1", Price: "10"}, Draft{Kind: Close, Quantity: "-1", Price: "8"})
	result = CalculatePerformance(context.Background(), book, PerformanceFilter{})
	if result.MaxDrawdown != 2 {
		t.Fatalf("ignored zero-origin decline: %v", result.MaxDrawdown)
	}
	if _, ok := (PerformanceSummary{}).WinRate(); ok {
		t.Fatal("empty win rate was defined")
	}
	if _, ok := (PerformanceSummary{Count: 1, Breakeven: 1}).PayoffRatio(); ok {
		t.Fatal("all-flat ratio was defined")
	}
	if value, ok := (PerformanceSummary{Count: 1, Wins: 1, GrossProfit: 1}).PayoffRatio(); !ok || !math.IsInf(value, 1) {
		t.Fatal("no-loss ratio should be infinite")
	}
}
