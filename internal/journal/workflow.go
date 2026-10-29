package journal

import (
	"errors"
	"math/big"
	"slices"
	"strings"
)

type Phase uint8

const (
	Planning Phase = iota
	Ready
	Holding
	Completed
)

var (
	ErrExecutionFields = errors.New("quantity and price are required")
	ErrPhase           = errors.New("entry is unavailable in this trading phase")
	ErrAddDirection    = errors.New("addition must have the same sign as the position")
	ErrReduceQuantity  = errors.New("reduction must oppose and be smaller than the position")
	ErrCloseQuantity   = errors.New("close must exactly offset the position")
	ErrFollowingTrade  = errors.New("change would invalidate a later trading action")
)

type workflow struct {
	net                      *big.Rat
	planned, started, closed bool
	issues                   map[string]error
}

func inspectWorkflow(entries []Entry) workflow {
	w := workflow{net: new(big.Rat), issues: map[string]error{}}
	opened := false
	for _, entry := range entries {
		if entry.Kind.IsText() {
			if entry.Kind == Plan && strings.TrimSpace(entry.Text) != "" {
				w.planned = true
			}
			continue
		}
		w.started = true
		if entry.Kind == Open {
			if !w.planned || opened || w.closed {
				w.issues[entry.ID] = ErrPhase
			}
			opened = true
		} else if w.closed {
			w.issues[entry.ID] = ErrPhase
		}
		quantity, ok := signedQuantity(entry.Quantity)
		if !ok {
			continue
		} // Old incomplete entries remain readable and editable.
		if err := validateDelta(entry.Kind, w.net, quantity); err != nil {
			w.issues[entry.ID] = err
		}
		w.net.Add(w.net, quantity)
		w.closed = entry.Kind == Close && w.net.Sign() == 0
	}
	return w
}

func (r *Record) Phase() Phase {
	w := inspectWorkflow(r.Entries)
	if w.started {
		if w.closed && w.net.Sign() == 0 {
			return Completed
		}
		return Holding
	}
	if w.planned {
		return Ready
	}
	return Planning
}

func (r *Record) CanReview() bool { return r.Phase() == Completed }

func (r *Record) AvailableKinds() []Kind {
	switch r.Phase() {
	case Ready:
		return []Kind{Note, Open}
	case Holding:
		return []Kind{Note, Add, Reduce, Close}
	case Completed:
		return nil
	default:
		return []Kind{Plan}
	}
}

// Historical edits keep their original action kind; the phase picker controls
// new entries, not a rewrite of an existing execution's meaning.
func (r *Record) EditorKinds() []Kind {
	if r.Edit != nil {
		return []Kind{r.Edit.Kind}
	}
	return r.AvailableKinds()
}

func (r *Record) editorPrefix() []Entry {
	if r.Edit != nil {
		for i, entry := range r.Entries {
			if entry.ID == r.Edit.EntryID {
				return r.Entries[:i]
			}
		}
	}
	return r.Entries
}

func (r *Record) ClosingQuantity() string {
	net := inspectWorkflow(r.editorPrefix()).net
	net.Neg(net)
	precision, _ := net.FloatPrec()
	return net.FloatString(precision)
}

func (r *Record) ReconcileEditor() {
	if r.Edit != nil {
		for _, entry := range r.Entries {
			if entry.ID == r.Edit.EntryID {
				r.Edit.Kind = entry.Kind
				break
			}
		}
	} else if kinds := r.AvailableKinds(); len(kinds) > 0 && !slices.Contains(kinds, r.Draft.Kind) {
		r.Draft.Kind = kinds[0]
	}
	if r.Editor().Kind == Close {
		r.Editor().Quantity = r.ClosingQuantity()
	}
}

func (r *Record) SelectKind(kind Kind) error {
	if !slices.Contains(r.EditorKinds(), kind) {
		return ErrPhase
	}
	draft := r.Editor()
	if draft.Kind == Close && kind != Close {
		draft.Quantity = ""
	}
	draft.Kind = kind
	if kind == Close {
		draft.Quantity = r.ClosingQuantity()
	}
	return nil
}

func validateDelta(kind Kind, net, quantity *big.Rat) error {
	switch kind {
	case Open:
		if net.Sign() != 0 {
			return ErrPhase
		}
	case Add:
		if net.Sign() == 0 {
			return ErrPhase
		}
		if quantity.Sign() != net.Sign() {
			return ErrAddDirection
		}
	case Reduce:
		if net.Sign() == 0 || quantity.Sign() == net.Sign() || new(big.Rat).Abs(quantity).Cmp(new(big.Rat).Abs(net)) >= 0 {
			return ErrReduceQuantity
		}
	case Close:
		if net.Sign() == 0 || new(big.Rat).Add(net, quantity).Sign() != 0 {
			return ErrCloseQuantity
		}
	}
	return nil
}

// Preserve pre-existing legacy inconsistencies, but never introduce new broken
// downstream actions when a trader edits or deletes an earlier entry.
func (r *Record) validateReplacement(entries []Entry, changedID string) error {
	before, after := inspectWorkflow(r.Entries), inspectWorkflow(entries)
	if err := after.issues[changedID]; err != nil && before.issues[changedID] != err {
		return err
	}
	for id, err := range after.issues {
		if before.issues[id] != err {
			return ErrFollowingTrade
		}
	}
	return nil
}
