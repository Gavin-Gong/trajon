package market

import "sort"

// DefaultInstruments keeps the original crypto starters first and adds each
// provider's offline suggestions in deterministic source order.
func (r *Registry) DefaultInstruments() []Instrument {
	items := InstrumentCatalog()
	if r == nil {
		return items
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]DataSourceID, 0, len(r.sources))
	for id := range r.sources {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	seen := make(map[string]bool)
	for _, item := range items {
		seen[item.Identity()] = true
	}
	for _, id := range ids {
		source, ok := r.sources[id].(InstrumentSuggestionsSource)
		if !ok {
			continue
		}
		for _, item := range source.SuggestedInstruments() {
			if !seen[item.Identity()] {
				items = append(items, item)
				seen[item.Identity()] = true
			}
		}
	}
	return items
}
