package longbridge

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

const (
	chunkManifestPrefix  = "trajon-secret-chunks-v1\n"
	maxChunkedSecretSize = 1 << 20
)

// NewChunkedStore keeps OAuth JSON within the system store's per-item limit.
// Every part, including the small commit manifest, stays in the system vault;
// no credential data is written to files. Single-item entries remain readable.
func NewChunkedStore(store Store, maxSize int) Store {
	return &chunkedStore{store: store, maxSize: maxSize}
}

type chunkedStore struct {
	mu      sync.Mutex
	store   Store
	maxSize int
}

type chunkManifest struct {
	Generation string `json:"generation"`
	Size       int    `json:"size"`
	ChunkSize  int    `json:"chunk_size"`
	SHA256     string `json:"sha256"`
}

func (m chunkManifest) count() int { return (m.Size-1)/m.ChunkSize + 1 }
func (m chunkManifest) key(key string, index int) string {
	return fmt.Sprintf("%s.chunks-%s.%d", key, m.Generation, index)
}

func (s *chunkedStore) Get(key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.store.Get(key)
	if err != nil {
		return nil, err
	}
	manifest, chunked, err := decodeChunkManifest(data)
	if err != nil {
		return nil, err
	}
	if !chunked {
		return data, nil
	}
	value := make([]byte, 0, manifest.Size)
	for i := range manifest.count() {
		part, err := s.store.Get(manifest.key(key, i))
		if err != nil {
			return nil, err
		}
		expected := min(manifest.ChunkSize, manifest.Size-len(value))
		if len(part) != expected {
			return nil, errors.New("longbridge: incomplete secure credential")
		}
		value = append(value, part...)
	}
	digest := sha256.Sum256(value)
	if hex.EncodeToString(digest[:]) != manifest.SHA256 {
		return nil, errors.New("longbridge: secure credential integrity check failed")
	}
	return value, nil
}

func (s *chunkedStore) Set(key string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.maxSize <= 0 || len(value) > maxChunkedSecretSize {
		return errors.New("longbridge: invalid secure credential size")
	}
	previous, err := s.store.Get(key)
	if err != nil {
		return err
	}
	if len(value) <= s.maxSize {
		if err := s.store.Set(key, value); err != nil {
			return err
		}
		s.cleanup(key, previous)
		return nil
	}
	var generation [16]byte
	_, _ = rand.Read(generation[:])
	digest := sha256.Sum256(value)
	manifest := chunkManifest{
		Generation: hex.EncodeToString(generation[:]), Size: len(value),
		ChunkSize: s.maxSize, SHA256: hex.EncodeToString(digest[:]),
	}
	encoded, _ := json.Marshal(manifest)
	commit := append([]byte(chunkManifestPrefix), encoded...)
	if len(commit) > s.maxSize || manifest.count() > 1024 {
		return errors.New("longbridge: secure credential manifest exceeds item limit")
	}
	written := 0
	committed := false
	defer func() {
		if !committed {
			for i := range written {
				_ = s.store.Delete(manifest.key(key, i))
			}
		}
	}()
	for i := range manifest.count() {
		start := i * s.maxSize
		if err := s.store.Set(manifest.key(key, i), value[start:min(start+s.maxSize, len(value))]); err != nil {
			return err
		}
		written++
	}
	// This single atomic replacement is the commit point. Before it succeeds,
	// all readers still have the complete previous grant, even if a part fails.
	if err := s.store.Set(key, commit); err != nil {
		return err
	}
	committed = true
	s.cleanup(key, previous)
	return nil
}

func (s *chunkedStore) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous, err := s.store.Get(key)
	if err != nil {
		return err
	}
	// Remove the active reference first. A failed deletion leaves all active
	// parts intact; successful deletion can never restore an old grant.
	if err := s.store.Delete(key); err != nil {
		return err
	}
	s.cleanup(key, previous)
	return nil
}

// Obsolete parts are best-effort garbage collection, not part of the commit:
// cleanup failure must not report an otherwise successful grant save as failed.
func (s *chunkedStore) cleanup(key string, data []byte) {
	manifest, chunked, err := decodeChunkManifest(data)
	if err != nil || !chunked {
		return
	}
	for i := range manifest.count() {
		_ = s.store.Delete(manifest.key(key, i))
	}
}

func decodeChunkManifest(data []byte) (chunkManifest, bool, error) {
	var manifest chunkManifest
	encoded, chunked := bytes.CutPrefix(data, []byte(chunkManifestPrefix))
	if !chunked {
		return manifest, false, nil
	}
	if err := json.Unmarshal(encoded, &manifest); err != nil {
		return manifest, true, errors.New("longbridge: invalid secure credential manifest")
	}
	generation, generationErr := hex.DecodeString(manifest.Generation)
	digest, digestErr := hex.DecodeString(manifest.SHA256)
	if generationErr != nil || len(generation) != 16 || digestErr != nil || len(digest) != sha256.Size ||
		manifest.Size <= 0 || manifest.Size > maxChunkedSecretSize || manifest.ChunkSize <= 0 ||
		manifest.ChunkSize > maxChunkedSecretSize || manifest.count() > 1024 {
		return manifest, true, errors.New("longbridge: invalid secure credential manifest")
	}
	return manifest, true, nil
}
