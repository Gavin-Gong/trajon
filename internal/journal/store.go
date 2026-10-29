package journal

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"time"
)

const maxBookBytes = 32 << 20

func Directory() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "trajon", "journal"), nil
}

// Load rejects invalid documents rather than silently replacing the user's
// journal. The UI must offer retry without enabling writes over a failed load.
func Load(dir string, now time.Time) (Book, error) {
	file, err := os.Open(filepath.Join(dir, "records.json"))
	if errors.Is(err, os.ErrNotExist) {
		return New(now), nil
	}
	if err != nil {
		return Book{}, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxBookBytes+1))
	if err != nil {
		return Book{}, err
	}
	if len(data) > maxBookBytes {
		return Book{}, errors.New("journal document is too large")
	}
	var book Book
	if err := json.Unmarshal(data, &book); err != nil {
		return Book{}, err
	}
	if err := book.validate(); err != nil {
		return Book{}, err
	}
	book.migrate()
	return book, nil
}

func (b Book) validate() error {
	if (b.Version != 1 && b.Version != 2) || len(b.Records) == 0 {
		return errors.New("unsupported or empty journal document")
	}
	ids := map[string]bool{}
	for _, r := range b.Records {
		if r == nil || r.ID == "" || ids[r.ID] || r.CreatedAt.IsZero() || r.UpdatedAt.IsZero() {
			return errors.New("invalid journal record")
		}
		ids[r.ID] = true
		if r.Direction != "" && r.Direction != "long" && r.Direction != "short" {
			return errors.New("invalid journal direction")
		}
		entryIDs := map[string]bool{}
		for _, e := range r.Entries {
			if e.ID == "" || entryIDs[e.ID] || e.At.IsZero() || !slices.Contains(Kinds, e.Kind) {
				return errors.New("invalid journal entry")
			}
			entryIDs[e.ID] = true
		}
		if !slices.Contains(Kinds, r.Draft.Kind) || r.Edit != nil && (!entryIDs[r.Edit.EntryID] || !slices.Contains(Kinds, r.Edit.Kind)) {
			return errors.New("invalid journal draft")
		}
		if len(r.Screenshots) > MaxScreenshots {
			return errors.New("too many screenshot references")
		}
		files := map[string]bool{}
		for _, s := range r.Screenshots {
			if !validScreenshotFile(s.File) || files[s.File] {
				return errors.New("invalid screenshot reference")
			}
			files[s.File] = true
		}
		tags := map[string]bool{}
		for _, tag := range r.Tags {
			if tag == "" || tags[tag] {
				return errors.New("invalid journal tag")
			}
			tags[tag] = true
		}
	}
	if !ids[b.ActiveID] {
		return errors.New("active journal record is missing")
	}
	return nil
}

func Save(dir string, book Book) error {
	if err := book.validate(); err != nil {
		return err
	}
	data, err := json.Marshal(book)
	if err != nil {
		return err
	}
	if len(data) > maxBookBytes {
		return errors.New("journal document is too large")
	}
	return atomicWrite(filepath.Join(dir, "records.json"), data)
}

// A synced replacement is committed only after all bytes have been written.
// A failed write never truncates the previous journal or image.
func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".journal-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

type Snapshot struct {
	Revision  uint64
	Book      Book
	Immediate bool
}

type Result struct {
	Revision uint64
	Err      error
}

// Store has one writer, debounces drafts, and drains the most recent immutable
// snapshot on Close. Save/Close belong to the UI owner; callbacks run on the
// worker and must be published through Window.Update.
type Store struct {
	updates chan Snapshot
	done    chan struct{}
	closed  bool
	err     error
}

func NewStore(dir string, onResult func(Result)) *Store {
	s := &Store{updates: make(chan Snapshot, 1), done: make(chan struct{})}
	go func() {
		defer close(s.done)
		for snapshot := range s.updates {
			closing := false
			if !snapshot.Immediate {
				timer := time.NewTimer(300 * time.Millisecond)
			wait:
				for {
					select {
					case next, ok := <-s.updates:
						if !ok {
							closing = true
							break wait
						}
						snapshot = next
						if next.Immediate {
							break wait
						}
						timer.Reset(300 * time.Millisecond)
					case <-timer.C:
						break wait
					}
				}
				timer.Stop()
			}
			s.err = Save(dir, snapshot.Book)
			if onResult != nil {
				onResult(Result{Revision: snapshot.Revision, Err: s.err})
			}
			if closing {
				return
			}
		}
	}()
	return s
}

func (s *Store) Save(snapshot Snapshot) {
	if s == nil || s.closed {
		return
	}
	snapshot.Book = snapshot.Book.Clone()
	select {
	case s.updates <- snapshot:
		return
	default:
	}
	select {
	case <-s.updates:
	default:
	}
	s.updates <- snapshot
}

func (s *Store) Close() error {
	if s == nil {
		return nil
	}
	if !s.closed {
		s.closed = true
		close(s.updates)
	}
	<-s.done
	if s.err != nil {
		return fmt.Errorf("save journal: %w", s.err)
	}
	return nil
}
