package journal

import (
	"context"
	"math"
	"math/big"
	"slices"
	"strconv"
	"strings"
	"time"
)

// PerformanceFilter selects completed trades by their recorded closing time,
// not creation/review time. Before is exclusive. All amounts are USD.
type PerformanceFilter struct {
	From, Before time.Time
	Instrument   string
	Location     *time.Location
}

type Trade struct {
	ID, Instrument, Direction string
	OpenedAt, ClosedAt        time.Time
	Tags                      []string
	PnL                       float64
	Outcome                   int
	amount                    *big.Rat
}

type PerformanceIssue struct {
	ID, Instrument string
}

type PerformanceSummary struct {
	Count, Wins, Losses, Breakeven int
	PnL, GrossProfit, GrossLoss    float64
	MaxDrawdown                    float64 // Amount, not a percentage of fictitious capital.
}

func (s PerformanceSummary) WinRate() (float64, bool) {
	if s.Count == 0 {
		return 0, false
	}
	return float64(s.Wins) / float64(s.Count), true
}

func (s PerformanceSummary) PayoffRatio() (float64, bool) {
	if s.Losses == 0 {
		return math.Inf(1), s.Wins > 0
	}
	if s.Wins == 0 {
		return 0, true
	}
	if !finiteAmount(s.GrossProfit) || !finiteAmount(s.GrossLoss) {
		return 0, false
	}
	return (s.GrossProfit / float64(s.Wins)) / (s.GrossLoss / float64(s.Losses)), true
}

type PerformanceDay struct {
	Date time.Time
	PerformanceSummary
	Cumulative float64
}

type PerformancePoint struct {
	At              time.Time
	PnL, Cumulative float64
	Count           int
}

type PerformanceGroup struct {
	Name string
	PerformanceSummary
}

type Performance struct {
	Instruments []string
	Trades      []Trade // Oldest closing first; ties retain their book order.
	Issues      []PerformanceIssue
	PerformanceSummary
	Days                []PerformanceDay   // Trading days only, in the requested location.
	Points              []PerformancePoint // Equal closing timestamps are aggregated.
	ByInstrument, ByTag []PerformanceGroup
}

// closedTrade uses exact decimal cash flows so fractional cancellations and
// breakeven classification do not depend on floating point roundoff. Prices
// are recorded in USD, with no conversion, contract multipliers or unrecorded fees.
func closedTrade(r *Record) (Trade, bool) {
	entries := slices.Clone(r.Entries)
	slices.SortStableFunc(entries, func(a, b Entry) int { return a.At.Compare(b.At) })
	w := inspectWorkflow(entries)
	if !w.closed || w.net.Sign() != 0 || len(w.issues) != 0 {
		return Trade{}, false
	}
	trade := Trade{ID: r.ID, Instrument: strings.TrimSpace(r.Instrument), Tags: slices.Clone(r.Tags), amount: new(big.Rat)}
	for _, entry := range entries {
		if entry.Kind.IsText() {
			continue
		}
		quantity, ok := signedQuantity(entry.Quantity)
		// Reject non-finite/oversized exponents before exact rational parsing.
		priceValue, err := strconv.ParseFloat(entry.Price, 64)
		if !ok || err != nil || !finiteAmount(priceValue) || priceValue <= 0 || entry.At.IsZero() {
			return Trade{}, false
		}
		price, valid := new(big.Rat).SetString(entry.Price)
		if !valid {
			return Trade{}, false
		}
		if trade.OpenedAt.IsZero() {
			trade.OpenedAt = entry.At
			trade.Direction = "long"
			if quantity.Sign() < 0 {
				trade.Direction = "short"
			}
		}
		trade.amount.Sub(trade.amount, new(big.Rat).Mul(quantity, price))
		trade.ClosedAt = entry.At
	}
	trade.PnL, _ = trade.amount.Float64()
	trade.Outcome = trade.amount.Sign()
	return trade, finiteAmount(trade.PnL)
}

func finiteAmount(n float64) bool { return !math.IsNaN(n) && !math.IsInf(n, 0) }

type performanceAccumulator struct {
	summary           PerformanceSummary
	net, profit, loss big.Rat
}

func (a *performanceAccumulator) add(t Trade) {
	a.summary.Count++
	a.net.Add(&a.net, t.amount)
	switch t.Outcome {
	case 1:
		a.summary.Wins++
		a.profit.Add(&a.profit, t.amount)
	case -1:
		a.summary.Losses++
		a.loss.Sub(&a.loss, t.amount)
	default:
		a.summary.Breakeven++
	}
}

func (a *performanceAccumulator) value() PerformanceSummary {
	s := a.summary
	s.PnL, _ = a.net.Float64()
	s.GrossProfit, _ = a.profit.Float64()
	s.GrossLoss, _ = a.loss.Float64()
	return s
}

// CalculatePerformance owns its output and never changes the book. Native
// callers use an immutable book snapshot and cancel/reject superseded work.
func CalculatePerformance(ctx context.Context, book Book, filter PerformanceFilter) Performance {
	result := Performance{}
	instruments := map[string]bool{}
	for _, r := range book.Records {
		if ctx.Err() != nil {
			return Performance{}
		}
		if r == nil || r.Empty() {
			continue
		}
		name := strings.TrimSpace(r.Instrument)
		if name != "" {
			instruments[name] = true
		}
		if !slices.ContainsFunc(r.Entries, func(e Entry) bool { return e.Kind == Close }) {
			continue
		}
		trade, ok := closedTrade(r)
		if !ok {
			var closedAt time.Time
			for _, entry := range r.Entries {
				if entry.Kind == Close && entry.At.After(closedAt) {
					closedAt = entry.At
				}
			}
			if filter.includes(name, closedAt) {
				result.Issues = append(result.Issues, PerformanceIssue{ID: r.ID, Instrument: name})
			}
			continue
		}
		if filter.includes(trade.Instrument, trade.ClosedAt) {
			result.Trades = append(result.Trades, trade)
		}
	}
	for name := range instruments {
		result.Instruments = append(result.Instruments, name)
	}
	slices.Sort(result.Instruments)
	slices.SortStableFunc(result.Trades, func(a, b Trade) int { return a.ClosedAt.Compare(b.ClosedAt) })
	location := filter.Location
	if location == nil {
		location = time.Local
	}
	var total performanceAccumulator
	var peak, drawdown big.Rat
	days := map[string]*performanceAccumulator{}
	groups, tags := map[string]*performanceAccumulator{}, map[string]*performanceAccumulator{}
	for i := 0; i < len(result.Trades); {
		if ctx.Err() != nil {
			return Performance{}
		}
		at := result.Trades[i].ClosedAt
		point := PerformancePoint{At: at}
		var batch big.Rat
		for i < len(result.Trades) && result.Trades[i].ClosedAt.Equal(at) {
			trade := result.Trades[i]
			total.add(trade)
			batch.Add(&batch, trade.amount)
			point.Count++
			date := trade.ClosedAt.In(location).Format("2006-01-02")
			addPerformanceGroup(days, date, trade)
			addPerformanceGroup(groups, trade.Instrument, trade)
			seen := map[string]bool{}
			for _, tag := range trade.Tags {
				if tag = strings.TrimSpace(tag); tag != "" && !seen[tag] {
					seen[tag] = true
					addPerformanceGroup(tags, tag, trade)
				}
			}
			i++
		}
		point.PnL, _ = batch.Float64()
		point.Cumulative, _ = total.net.Float64()
		result.Points = append(result.Points, point)
		if total.net.Cmp(&peak) > 0 {
			peak.Set(&total.net)
		}
		decline := new(big.Rat).Sub(&peak, &total.net)
		if decline.Cmp(&drawdown) > 0 {
			drawdown.Set(decline)
		}
	}
	result.PerformanceSummary = total.value()
	result.MaxDrawdown, _ = drawdown.Float64()
	var dates []string
	for date := range days {
		dates = append(dates, date)
	}
	slices.Sort(dates)
	var cumulative big.Rat
	for _, date := range dates {
		day, _ := time.ParseInLocation("2006-01-02", date, location)
		cumulative.Add(&cumulative, &days[date].net)
		value, _ := cumulative.Float64()
		result.Days = append(result.Days, PerformanceDay{Date: day, PerformanceSummary: days[date].value(), Cumulative: value})
	}
	result.ByInstrument = performanceGroups(groups)
	result.ByTag = performanceGroups(tags)
	return result
}

func (f PerformanceFilter) includes(instrument string, at time.Time) bool {
	return (f.Instrument == "" || f.Instrument == instrument) && (at.IsZero() ||
		(f.From.IsZero() || !at.Before(f.From)) && (f.Before.IsZero() || at.Before(f.Before)))
}

func addPerformanceGroup(groups map[string]*performanceAccumulator, key string, trade Trade) {
	if groups[key] == nil {
		groups[key] = &performanceAccumulator{}
	}
	groups[key].add(trade)
}

func performanceGroups(groups map[string]*performanceAccumulator) []PerformanceGroup {
	var result []PerformanceGroup
	for name, value := range groups {
		result = append(result, PerformanceGroup{Name: name, PerformanceSummary: value.value()})
	}
	slices.SortFunc(result, func(a, b PerformanceGroup) int {
		if a.PnL > b.PnL {
			return -1
		}
		if a.PnL < b.PnL {
			return 1
		}
		return strings.Compare(a.Name, b.Name)
	})
	return result
}
