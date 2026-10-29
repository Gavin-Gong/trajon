package journal

import (
	"errors"
	"slices"
	"testing"
	"time"
)

func TestWorkflowGatesActionsAndRequiresAFullClose(t *testing.T) {
	now := time.Now()
	book := New(now)
	r := book.Active()
	r.Draft.Text = "等待确认"
	if r.Phase() != Planning || !slices.Equal(r.AvailableKinds(), []Kind{Plan}) || r.CanReview() {
		t.Fatal("typing a plan unlocked trading or review")
	}
	if err := r.SelectKind(Open); !errors.Is(err, ErrPhase) {
		t.Fatal("opening unlocked before submitting the plan")
	}
	if err := r.Commit(now); err != nil {
		t.Fatal(err)
	}
	if r.Phase() != Ready || !slices.Equal(r.AvailableKinds(), []Kind{Note, Open}) {
		t.Fatal("submitted plan did not unlock note/open")
	}
	r.Draft = Draft{Kind: Open, Quantity: "100"}
	if !r.Draft.Empty() {
		t.Fatal("execution without price is ready to submit")
	}
	if err := r.Commit(now); !errors.Is(err, ErrExecutionFields) || len(r.Entries) != 1 {
		t.Fatal("incomplete open changed the record")
	}
	r.Draft.Price = "10"
	if err := r.Commit(now.Add(time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.AvailableKinds(), []Kind{Note, Add, Reduce, Close}) || r.CanReview() {
		t.Fatal("holding exposed wrong actions")
	}
	r.Draft = Draft{Kind: Add, Quantity: "-20", Price: "11"}
	if err := r.Commit(now); !errors.Is(err, ErrAddDirection) || r.NetPosition().Quantity != "100" {
		t.Fatal("opposing add changed the position")
	}
	r.Draft.Quantity = "20"
	if err := r.Commit(now.Add(2 * time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	for _, quantity := range []string{"20", "-120", "-121"} {
		r.Draft = Draft{Kind: Reduce, Quantity: quantity, Price: "11"}
		if err := r.Commit(now); !errors.Is(err, ErrReduceQuantity) || r.NetPosition().Quantity != "120" {
			t.Fatalf("invalid reduction %s changed the position", quantity)
		}
	}
	r.Draft = Draft{Kind: Reduce, Quantity: "-30", Price: "11"}
	if err := r.Commit(now.Add(3 * time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if err := r.SelectKind(Close); err != nil {
		t.Fatal(err)
	}
	if r.Draft.Quantity != "-90" || !r.Draft.Empty() {
		t.Fatalf("close was not prefilled or did not require a price: %+v", r.Draft)
	}
	r.Draft.Quantity, r.Draft.Price = "-80", "12"
	if err := r.Commit(now); !errors.Is(err, ErrCloseQuantity) || r.CanReview() {
		t.Fatal("partial close unlocked review")
	}
	r.ReconcileEditor()
	if r.Draft.Quantity != "-90" || r.Draft.Price != "12" {
		t.Fatal("close reconciliation lost price or did not follow the position")
	}
	if err := r.Commit(now.Add(4 * time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if !r.CanReview() || r.NetPosition().Quantity != "0" || len(r.AvailableKinds()) != 0 {
		t.Fatal("full close did not unlock review")
	}
	if err := r.SelectKind(Note); !errors.Is(err, ErrPhase) {
		t.Fatal("note selection remained available after close")
	}
	r.Draft = Draft{Kind: Note, Text: "平仓后不允许新增"}
	count := len(r.Entries)
	if err := r.Commit(now.Add(5 * time.Millisecond)); !errors.Is(err, ErrPhase) || len(r.Entries) != count {
		t.Fatal("note committed after close")
	}
	r.ReconcileEditor() // A completed record has no new-entry kinds.
	if err := r.RemoveEntry(r.Entries[len(r.Entries)-1].ID); err != nil {
		t.Fatal(err)
	}
	if r.CanReview() || r.Phase() != Holding {
		t.Fatal("removing close left review unlocked")
	}
}

func TestHistoricalChangesCannotInvalidateLaterTrades(t *testing.T) {
	now := time.Now()
	book := New(now)
	r := book.Active()
	for i, draft := range []Draft{
		{Kind: Plan, Text: "计划"},
		{Kind: Open, Quantity: "100", Price: "10"},
		{Kind: Reduce, Quantity: "-20", Price: "11"},
		{Kind: Close, Quantity: "-80", Price: "12"},
	} {
		r.Draft = draft
		if err := r.Commit(now.Add(time.Duration(i) * time.Millisecond)); err != nil {
			t.Fatal(err)
		}
	}
	r.Draft = Draft{Kind: Note, Text: "未提交的下一条"}
	openID, created := r.Entries[1].ID, r.Entries[1].At
	r.EditEntry(openID)
	r.Edit.Quantity = "200"
	if err := r.Commit(now.Add(time.Minute)); !errors.Is(err, ErrFollowingTrade) {
		t.Fatalf("edit should invalidate the later close: %v", err)
	}
	if r.Entries[1].Quantity != "100" || r.Edit.Quantity != "200" || r.Draft.Text != "未提交的下一条" {
		t.Fatal("failed edit mutated the execution or discarded drafts")
	}
	r.Edit.Quantity, r.Edit.Price = "100", "9.5"
	if err := r.Commit(now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if !r.Entries[1].At.Equal(created) || !r.CanReview() {
		t.Fatal("valid edit changed creation time or completion")
	}
	if err := r.RemoveEntry(r.Entries[0].ID); !errors.Is(err, ErrFollowingTrade) || len(r.Entries) != 4 {
		t.Fatal("deleting the plan left an invalid trading sequence")
	}
}
