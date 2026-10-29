package journal

import (
	"slices"
	"strings"
)

type Filter uint8

const (
	AllRecords Filter = iota
	ActiveRecords
	ClosedRecords
)

// Empty ignores editor kind and timestamps, but never discards incomplete input.
func (r *Record) Empty() bool {
	return strings.TrimSpace(r.Instrument+r.Summary+r.Plan+r.TagDraft+
		r.Draft.Text+r.Draft.Quantity+r.Draft.Price+r.Draft.Time) == "" &&
		len(r.Entries) == 0 && len(r.Tags) == 0 && len(r.Screenshots) == 0 && r.Edit == nil
}

// List returns a new, newest-created-first slice. Editing an old record must
// not move it, and filtering must not change the active record or its drafts.
func (b Book) List(query string, filter Filter) []*Record {
	terms := strings.Fields(strings.ToLower(query))
	var records []*Record
	for _, r := range b.Records {
		if r.Empty() || filter == ActiveRecords && r.CanReview() || filter == ClosedRecords && !r.CanReview() {
			continue
		}
		fields := []string{r.Instrument, r.Summary, r.Plan, r.TagDraft, r.Draft.Text}
		fields = append(fields, r.Tags...)
		for _, e := range r.Entries {
			fields = append(fields, e.Text)
		}
		if r.Edit != nil {
			fields = append(fields, r.Edit.Text)
		}
		haystack := strings.ToLower(strings.Join(fields, "\n"))
		if !slices.ContainsFunc(terms, func(term string) bool { return !strings.Contains(haystack, term) }) {
			records = append(records, r)
		}
	}
	slices.SortStableFunc(records, func(a, b *Record) int { return b.CreatedAt.Compare(a.CreatedAt) })
	return records
}
