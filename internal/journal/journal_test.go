package journal

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestEntriesKeepCreationTimeAndUnsubmittedDraftWhileEditing(t *testing.T) {
	now := time.Date(2026, 10, 17, 10, 6, 0, 0, time.Local)
	book := New(now)
	r := book.Active()
	r.Draft = Draft{Kind: Plan, Text: "等待确认", Quantity: "100"}
	if err := r.Commit(now); err != nil {
		t.Fatal(err)
	}
	if r.Entries[0].Quantity != "" {
		t.Fatal("plan leaked execution fields")
	}
	r.Draft = Draft{Kind: Open, Quantity: ".5", Price: "10.24", Text: "按计划试仓", Time: "invalid legacy draft time"}
	created := now.Add(time.Minute)
	if err := r.Commit(created); err != nil {
		t.Fatal(err)
	}
	id := r.Entries[1].ID
	r.Draft = Draft{Kind: Note, Text: "还没提交的想法"}
	r.EditEntry(id)
	r.Edit.Time, r.Edit.Quantity = "2026-10-17 09:42", "0.75"
	if err := r.Commit(now.Add(2 * time.Minute)); err != nil {
		t.Fatal(err)
	}
	if r.Draft.Text != "还没提交的想法" || r.Entries[1].ID != id || r.Entries[1].Quantity != "0.75" ||
		!r.Entries[1].At.Equal(created) || !r.Entries[1].ModifiedAt.Equal(now.Add(2*time.Minute)) {
		t.Fatalf("edit lost creation time, identity or composer: %+v", r)
	}
	if err := r.Commit(now.Add(3 * time.Minute)); err != nil {
		t.Fatal(err)
	}
	r.Draft = Draft{Kind: Close, Text: "本次已平仓"}
	if err := r.Commit(now); !errors.Is(err, ErrExecutionFields) {
		t.Fatal("execution accepted without quantity and price")
	}
	for _, invalid := range []string{"NaN", "+Inf", "0", "unfinished"} {
		r.Draft = Draft{Kind: Add, Quantity: invalid, Price: "10"}
		if err := r.Commit(now); !errors.Is(err, ErrNumber) {
			t.Fatalf("quantity %q: %v", invalid, err)
		}
	}
	r.Draft = Draft{Kind: Open, Quantity: "-0.5", Price: "-1"}
	if err := r.Commit(now); !errors.Is(err, ErrNumber) {
		t.Fatal("negative price accepted")
	}
}

func TestNetPositionUsesSignedBuySellQuantitiesAndExactDecimals(t *testing.T) {
	now := time.Now()
	book := New(now)
	r := book.Active()
	r.Draft = Draft{Kind: Plan, Text: "交易计划"}
	if err := r.Commit(now); err != nil {
		t.Fatal(err)
	}
	for i, step := range []struct {
		kind                     Kind
		quantity, net, direction string
	}{
		{Open, "-0.1", "-0.1", "short"},
		{Add, "-0.2", "-0.3", "short"},
		{Close, "+0.3", "0", "flat"},
	} {
		r.Draft = Draft{Kind: step.kind, Quantity: step.quantity, Price: "10"}
		if err := r.Commit(now.Add(time.Duration(i+1) * time.Minute)); err != nil {
			t.Fatal(err)
		}
		position := r.NetPosition()
		if !position.Recorded || position.Quantity != step.net || position.Direction != step.direction {
			t.Fatalf("%s %s: %+v", step.kind, step.quantity, position)
		}
	}
	if err := r.RemoveEntry(r.Entries[len(r.Entries)-1].ID); err != nil {
		t.Fatal(err)
	}
	if position := r.NetPosition(); position.Quantity != "-0.3" || position.Direction != "short" {
		t.Fatalf("removal left stale position: %+v", position)
	}
	if r.CanReview() {
		t.Fatal("removing the close left review unlocked")
	}
}

func TestJournalRoundTripDrainsImmutableSnapshotAndPreservesPreviousOnFailure(t *testing.T) {
	dir, now := t.TempDir(), time.Now().UTC()
	book := New(now)
	r := book.Active()
	r.Instrument, r.Plan, r.Summary = "手动标的", "等待确认", "## 总结\n\n- 按计划执行"
	r.Draft = Draft{Kind: Reduce, Quantity: "0.", Time: "2026-10-"}
	r.Tags = []string{"突破"}
	r.Entries = []Entry{{ID: "entry", Kind: Open, At: now, Text: "原始记录"}}
	r.EditEntry("entry")
	r.Edit.Text = "修改中"
	store := NewStore(dir, nil)
	store.Save(Snapshot{Revision: 1, Book: book})
	r.Plan, r.Tags[0] = "最新计划", "计划内"
	store.Save(Snapshot{Revision: 2, Book: book})
	r.Plan, r.Tags[0], r.Edit.Text = "不能泄露到写入快照", "不能泄露", "不能泄露到编辑快照"
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	// Older schema-v2 records may contain currency metadata. It is ignored
	// without changing the saved executions, drafts or schema version.
	path := filepath.Join(dir, "records.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data = bytes.Replace(data, []byte(`"instrument":`), []byte(`"currency":"USDT","instrument":`), 1)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(dir, now)
	if err != nil {
		t.Fatal(err)
	}
	got := loaded.Active()
	if got.Plan != "" || !slices.ContainsFunc(got.Entries, func(e Entry) bool { return e.Kind == Plan && e.Text == "最新计划" }) || got.Tags[0] != "计划内" || got.Draft.Quantity != "0." || got.Summary != r.Summary || got.Edit == nil || got.Edit.Text != "修改中" {
		t.Fatalf("round trip: %+v", got)
	}
	if err := Save(dir, loaded); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(path)
	if err != nil || bytes.Contains(data, []byte(`"currency"`)) {
		t.Fatalf("obsolete currency persisted: %v", err)
	}
	again, err := Load(dir, now)
	if err != nil || len(again.Active().Entries) != len(got.Entries) {
		t.Fatalf("plan migrated more than once: %v", err)
	}
	loaded.ActiveID = "missing"
	if err := Save(dir, loaded); err == nil {
		t.Fatal("invalid state overwrote journal")
	}
	if _, err := Load(dir, now); err != nil {
		t.Fatalf("previous document lost: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "records.json"), []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir, now); err == nil {
		t.Fatal("corruption silently reset journal")
	}
}

func TestLegacyUnsignedSizesMigrateOnceWithTheirDirection(t *testing.T) {
	for _, direction := range []string{"long", "short"} {
		t.Run(direction, func(t *testing.T) {
			dir, now := t.TempDir(), time.Now()
			book := New(now)
			book.Version = 1
			r := book.Active()
			r.Direction = direction
			r.Entries = []Entry{
				{ID: "open", Kind: Open, At: now, Quantity: "2"},
				{ID: "reduce", Kind: Reduce, At: now, Quantity: "0.5"},
			}
			r.Draft = Draft{Kind: Close, Quantity: "1.5", Price: "11"}
			if err := Save(dir, book); err != nil {
				t.Fatal(err)
			}
			loaded, err := Load(dir, now)
			if err != nil {
				t.Fatal(err)
			}
			expected := "1.5"
			if direction == "short" {
				expected = "-1.5"
			}
			if loaded.Version != 2 || loaded.Active().NetPosition().Quantity != expected {
				t.Fatalf("legacy %s meaning changed: %+v", direction, loaded.Active())
			}
			if err := loaded.Active().Commit(now.Add(time.Minute)); err != nil {
				t.Fatal(err)
			}
			if loaded.Active().NetPosition().Direction != "flat" {
				t.Fatal("legacy closing draft did not close the position")
			}
			if err := Save(dir, loaded); err != nil {
				t.Fatal(err)
			}
			again, err := Load(dir, now)
			if err != nil || again.Active().NetPosition().Direction != "flat" {
				t.Fatalf("v2 quantities converted twice: %v", err)
			}
		})
	}
}

func TestScreenshotsUseDurableValidatedLocalCopies(t *testing.T) {
	dir := t.TempDir()
	var raw bytes.Buffer
	if err := png.Encode(&raw, image.NewRGBA(image.Rect(0, 0, 8, 6))); err != nil {
		t.Fatal(err)
	}
	s, normalized, err := ImportImage(dir, "/external/screenshot.png", raw.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	stored, err := LoadImage(dir, s.File)
	if err != nil || !bytes.Equal(stored, normalized) || s.Name != "screenshot.png" {
		t.Fatalf("image round trip: %+v, %v", s, err)
	}
	if _, err := LoadImage(dir, "../../secret.png"); !errors.Is(err, ErrImage) {
		t.Fatal("unsafe image reference accepted")
	}
	if _, _, err := ImportImage(dir, "bad.png", []byte("not an image")); !errors.Is(err, ErrImage) {
		t.Fatal("non-image accepted")
	}
	book := New(time.Now())
	book.Active().AddScreenshot(s)
	book.Active().AddScreenshot(s)
	if len(book.Active().Screenshots) != 1 {
		t.Fatal("duplicate image added twice")
	}
	if err := Save(dir, book); err != nil {
		t.Fatal(err)
	}
	// References are relative content hashes, never a dependency on the source.
	b, err := os.ReadFile(filepath.Join(dir, "records.json"))
	if err != nil || strings.Contains(string(b), "/external/") {
		t.Fatal("external image path persisted")
	}
}
