package journal

import (
	"slices"
	"testing"
	"time"
)

func TestListKeepsCreationOrderAndSearchesDraftsWithoutMutatingBook(t *testing.T) {
	now := time.Now()
	book := New(now)
	old := book.Active()
	old.Instrument, old.Summary = "标的 A", "耐心等待"
	old.CreatedAt = now.AddDate(0, 0, -2)
	old.UpdatedAt = now.Add(time.Hour) // Editing must not move it above a newer trade.
	newer := book.AddRecord(now.AddDate(0, 0, -1))
	newer.Instrument, newer.Tags = "Sample", []string{"突破"}
	newer.Draft.Text = "未提交的观察"
	newer.Edit = &Draft{Text: "正在修正"}
	blank := book.AddRecord(now)

	if got := book.List("", AllRecords); !slices.Equal(got, []*Record{newer, old}) {
		t.Fatalf("list should hide blanks and use creation time: %+v", got)
	}
	if got := book.List("SAMPLE 突破 修正", AllRecords); !slices.Equal(got, []*Record{newer}) {
		t.Fatalf("search did not include metadata and entry edits: %+v", got)
	}
	if got := book.List("未提交", AllRecords); !slices.Equal(got, []*Record{newer}) {
		t.Fatalf("search lost pending text: %+v", got)
	}
	if book.ActiveID != blank.ID || !slices.Equal(book.Records, []*Record{old, newer, blank}) || newer.Draft.Text != "未提交的观察" {
		t.Fatal("listing changed selection, book order or drafts")
	}
	blank.Draft.Quantity = "-" // A partial execution is data, not an empty record.
	if got := book.List("", AllRecords); len(got) != 3 || got[0] != blank {
		t.Fatal("partial input disappeared from the list")
	}
}

func TestListClosedFilterRequiresCommittedFullClose(t *testing.T) {
	now := time.Now()
	book := New(now)
	r := book.Active()
	for i, draft := range []Draft{
		{Kind: Plan, Text: "计划"},
		{Kind: Open, Quantity: "-0.3", Price: "10"},
		{Kind: Reduce, Quantity: "0.1", Price: "9"},
	} {
		r.Draft = draft
		if err := r.Commit(now.Add(time.Duration(i) * time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	r.Draft = Draft{Kind: Close, Quantity: "0.2", Price: "8"}
	if len(book.List("", ClosedRecords)) != 0 || len(book.List("", ActiveRecords)) != 1 {
		t.Fatal("an unsubmitted close changed the filter")
	}
	if err := r.Commit(now.Add(3 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if len(book.List("", ClosedRecords)) != 1 || len(book.List("", ActiveRecords)) != 0 {
		t.Fatal("completed short did not move to the closed filter")
	}
	if err := r.RemoveEntry(r.Entries[len(r.Entries)-1].ID); err != nil {
		t.Fatal(err)
	}
	if len(book.List("", ClosedRecords)) != 0 || len(book.List("", ActiveRecords)) != 1 {
		t.Fatal("removing close left a trade in the closed filter")
	}
}
