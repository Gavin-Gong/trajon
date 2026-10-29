// Package journal owns manual trading notes, independently of market data,
// simulated positions and live orders. Recording an action never executes it.
package journal

import (
	"crypto/rand"
	"errors"
	"math"
	"math/big"
	"slices"
	"strconv"
	"strings"
	"time"
)

type Kind string

const (
	Plan   Kind = "plan"
	Note   Kind = "note"
	Open   Kind = "open"
	Add    Kind = "add"
	Reduce Kind = "reduce"
	Close  Kind = "close"
)

var Kinds = []Kind{Plan, Note, Open, Add, Reduce, Close}

func (k Kind) IsText() bool { return k == Plan || k == Note }

var (
	ErrEmptyEntry = errors.New("empty journal entry")
	ErrNumber     = errors.New("quantity must be nonzero and finite; price must be positive and finite")
)

type Entry struct {
	ID         string    `json:"id"`
	Kind       Kind      `json:"kind"`
	At         time.Time `json:"at"`
	Text       string    `json:"text,omitempty"`
	Quantity   string    `json:"quantity,omitempty"`
	Price      string    `json:"price,omitempty"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

// Draft keeps incomplete inputs as strings; switching records or restarting
// must not destroy a half-entered number or unsubmitted note. Time is retained
// only for reading old drafts; new entries always use their creation time.
type Draft struct {
	EntryID  string `json:"entryId,omitempty"`
	Kind     Kind   `json:"kind"`
	Time     string `json:"time,omitempty"`
	Text     string `json:"text,omitempty"`
	Quantity string `json:"quantity,omitempty"`
	Price    string `json:"price,omitempty"`
}

type Screenshot struct {
	File string `json:"file"`
	Name string `json:"name"`
}

type Record struct {
	ID          string       `json:"id"`
	Instrument  string       `json:"instrument"`
	Direction   string       `json:"direction,omitempty"` // Legacy v1 manual direction.
	Plan        string       `json:"plan,omitempty"`      // Legacy separate plan, migrated on load.
	Summary     string       `json:"summary,omitempty"`
	Tags        []string     `json:"tags,omitempty"`
	TagDraft    string       `json:"tagDraft,omitempty"`
	Entries     []Entry      `json:"entries,omitempty"`
	Draft       Draft        `json:"draft"`
	Edit        *Draft       `json:"edit,omitempty"`
	Screenshots []Screenshot `json:"screenshots,omitempty"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
	SavedAt     time.Time    `json:"savedAt,omitempty"`
}

type Book struct {
	Version  int       `json:"version"`
	ActiveID string    `json:"activeId"`
	Records  []*Record `json:"records"`
}

func New(now time.Time) Book {
	b := Book{Version: 2}
	b.AddRecord(now)
	return b
}

func (b *Book) AddRecord(now time.Time) *Record {
	r := &Record{ID: rand.Text(), CreatedAt: now, UpdatedAt: now, Draft: Draft{Kind: Plan}}
	b.Records = append(b.Records, r)
	b.ActiveID = r.ID
	return r
}

func (b *Book) Record(id string) *Record {
	for _, r := range b.Records {
		if r.ID == id {
			return r
		}
	}
	return nil
}

func (b *Book) Active() *Record { return b.Record(b.ActiveID) }

func (b Book) Clone() Book {
	b.Records = slices.Clone(b.Records)
	for i, original := range b.Records {
		r := *original
		r.Entries = slices.Clone(r.Entries)
		r.Tags = slices.Clone(r.Tags)
		r.Screenshots = slices.Clone(r.Screenshots)
		if r.Edit != nil {
			edit := *r.Edit
			r.Edit = &edit
		}
		b.Records[i] = &r
	}
	return b
}

func (r *Record) Editor() *Draft {
	if r.Edit != nil {
		return r.Edit
	}
	return &r.Draft
}

func (r *Record) EditEntry(id string) {
	for _, e := range r.Entries {
		if e.ID == id {
			r.Edit = &Draft{EntryID: e.ID, Kind: e.Kind, Text: e.Text, Quantity: e.Quantity, Price: e.Price}
			return
		}
	}
}

func (r *Record) RemoveEntry(id string) error {
	entries := slices.DeleteFunc(slices.Clone(r.Entries), func(e Entry) bool { return e.ID == id })
	if err := r.validateReplacement(entries, ""); err != nil {
		return err
	}
	r.Entries = entries
	if r.Edit != nil && r.Edit.EntryID == id {
		r.Edit = nil
	}
	r.ReconcileEditor()
	return nil
}

func (d Draft) Empty() bool {
	if d.Kind.IsText() {
		return strings.TrimSpace(d.Text) == ""
	}
	return strings.TrimSpace(d.Quantity) == "" || strings.TrimSpace(d.Price) == ""
}

func (d Draft) Entry(now time.Time) (Entry, error) {
	if !slices.Contains(Kinds, d.Kind) {
		return Entry{}, ErrEmptyEntry
	}
	if d.Empty() {
		if !d.Kind.IsText() {
			return Entry{}, ErrExecutionFields
		}
		return Entry{}, ErrEmptyEntry
	}
	e := Entry{ID: d.EntryID, Kind: d.Kind, At: now, Text: strings.TrimSpace(d.Text), ModifiedAt: now}
	if !d.Kind.IsText() {
		e.Quantity, e.Price = strings.TrimSpace(d.Quantity), strings.TrimSpace(d.Price)
		if e.Quantity != "" {
			if _, ok := signedQuantity(e.Quantity); !ok {
				return Entry{}, ErrNumber
			}
		}
		if e.Price != "" {
			n, err := strconv.ParseFloat(e.Price, 64)
			if err != nil || n <= 0 || math.IsInf(n, 0) || math.IsNaN(n) {
				return Entry{}, ErrNumber
			}
		}
	}
	return e, nil
}

func (r *Record) Commit(now time.Time) error {
	e, err := r.Editor().Entry(now)
	if err != nil {
		return err
	}
	entries := slices.Clone(r.Entries)
	editing := r.Edit != nil
	if editing {
		i := slices.IndexFunc(entries, func(old Entry) bool { return old.ID == e.ID })
		if i < 0 {
			return ErrEmptyEntry
		}
		if e.Kind != entries[i].Kind {
			return ErrPhase
		}
		if !e.Kind.IsText() {
			quantity, _ := signedQuantity(e.Quantity)
			if err := validateDelta(e.Kind, inspectWorkflow(entries[:i]).net, quantity); err != nil {
				return err
			}
		}
		e.At = entries[i].At // Editing never changes when the entry was created.
		entries[i] = e
	} else {
		if !slices.Contains(r.AvailableKinds(), e.Kind) {
			return ErrPhase
		}
		if !e.Kind.IsText() {
			quantity, _ := signedQuantity(e.Quantity)
			if err := validateDelta(e.Kind, inspectWorkflow(r.Entries).net, quantity); err != nil {
				return err
			}
		}
		e.ID = rand.Text()
		entries = append(entries, e)
	}
	slices.SortStableFunc(entries, func(a, b Entry) int { return a.At.Compare(b.At) })
	if err := r.validateReplacement(entries, e.ID); err != nil {
		return err
	}
	r.Entries = entries
	if editing {
		r.Edit = nil
	} else {
		r.Draft = Draft{Kind: Note}
	}
	r.UpdatedAt = now
	r.ReconcileEditor()
	return nil
}

// Position is inferred from recorded signed executions, not from a manually
// selected direction or an operation label. Missing quantities remain notes.
type Position struct {
	Quantity  string
	Direction string
	Recorded  bool
}

func signedQuantity(raw string) (*big.Rat, bool) {
	n, err := strconv.ParseFloat(raw, 64)
	if err != nil || n == 0 || math.IsInf(n, 0) || math.IsNaN(n) {
		return nil, false
	}
	return new(big.Rat).SetString(raw)
}

func (r *Record) NetPosition() Position {
	net := new(big.Rat)
	position := Position{Quantity: "0", Direction: "flat"}
	for _, entry := range r.Entries {
		if entry.Kind.IsText() {
			continue
		}
		quantity, ok := signedQuantity(entry.Quantity)
		if !ok {
			continue
		}
		net.Add(net, quantity)
		position.Recorded = true
	}
	if net.Sign() > 0 {
		position.Direction = "long"
	}
	if net.Sign() < 0 {
		position.Direction = "short"
	}
	// Decimal input has a terminating decimal sum. Preserve exact cancellation
	// (0.1 + 0.2 - 0.3 is flat), without binary floating-point dust positions.
	precision, _ := net.FloatPrec()
	position.Quantity = net.FloatString(precision)
	return position
}

// Migration changes only the loaded snapshot; the previous document remains
// intact until the next atomic save. V1 quantities were unsigned sizes paired
// with a manual direction. Convert their meaning, not merely their raw sign.
func (b *Book) migrate() {
	if b.Version == 1 {
		for _, r := range b.Records {
			short := r.Direction == "short"
			for i := range r.Entries {
				e := &r.Entries[i]
				e.Quantity = legacySignedQuantity(e.Quantity, e.Kind, short)
			}
			r.Draft.Quantity = legacySignedQuantity(r.Draft.Quantity, r.Draft.Kind, short)
			if r.Edit != nil {
				r.Edit.Quantity = legacySignedQuantity(r.Edit.Quantity, r.Edit.Kind, short)
			}
			r.Direction = ""
		}
	}
	b.migratePlans()
	b.Version = 2
}

func legacySignedQuantity(raw string, kind Kind, short bool) string {
	if kind.IsText() {
		return raw
	}
	n, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || n <= 0 || math.IsInf(n, 0) || math.IsNaN(n) {
		return raw
	}
	sell := kind == Reduce || kind == Close
	if short {
		sell = !sell
	}
	if sell {
		return "-" + strings.TrimPrefix(strings.TrimSpace(raw), "+")
	}
	return raw
}

// A separate legacy plan becomes one ordinary timeline entry.
func (b *Book) migratePlans() {
	for _, r := range b.Records {
		if strings.TrimSpace(r.Plan) == "" {
			continue
		}
		r.Entries = append(r.Entries, Entry{ID: rand.Text(), Kind: Plan, At: r.CreatedAt, Text: r.Plan, ModifiedAt: r.CreatedAt})
		r.Plan = ""
		slices.SortStableFunc(r.Entries, func(a, b Entry) int { return a.At.Compare(b.At) })
	}
}

func (r *Record) AddTag() {
	tag := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(r.TagDraft), "#"))
	if tag == "" {
		return
	}
	if !slices.Contains(r.Tags, tag) {
		r.Tags = append(r.Tags, tag)
	}
	r.TagDraft = ""
}

func (r *Record) AddScreenshot(s Screenshot) {
	if !slices.ContainsFunc(r.Screenshots, func(old Screenshot) bool { return old.File == s.File }) {
		r.Screenshots = append(r.Screenshots, s)
	}
}
