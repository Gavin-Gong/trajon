package longbridge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

const systemSecretLimit = 2560 // The pinned MyGo cross-platform limit.

type boundedSecrets struct {
	data       map[string][]byte
	writes     int
	failWrite  int
	failKey    string
	failDelete string
}

func (s *boundedSecrets) Get(key string) ([]byte, error) { return s.data[key], nil }
func (s *boundedSecrets) Set(key string, value []byte) error {
	if len(value) > systemSecretLimit {
		return errors.New("value exceeds system credential store limit")
	}
	s.writes++
	if s.writes == s.failWrite || key == s.failKey {
		return errors.New("credential write failed")
	}
	s.data[key] = bytes.Clone(value)
	return nil
}
func (s *boundedSecrets) Delete(key string) error {
	if key == s.failDelete {
		return errors.New("credential deletion failed")
	}
	delete(s.data, key)
	return nil
}

func largeTokens() Tokens {
	tokens := testTokens()
	tokens.AccessToken = strings.Repeat("a", 1600)
	tokens.RefreshToken = strings.Repeat("r", 1600)
	return tokens
}

func TestLargeOAuthGrantSurvivesSystemLimitAndRestart(t *testing.T) {
	vault := &boundedSecrets{data: map[string][]byte{}}
	store := NewChunkedStore(vault, systemSecretLimit)
	client := NewClient(store)
	// Existing single-item grants remain readable before migrating to chunks.
	if err := client.Save(testTokens()); err != nil {
		t.Fatal(err)
	}
	if status, err := NewClient(store).Restore(context.Background()); err != nil || !status.Authorized {
		t.Fatalf("legacy restore status=%+v, err=%v", status, err)
	}
	tokens := largeTokens()
	data, _ := json.Marshal(tokens)
	if len(data) <= systemSecretLimit {
		t.Fatal("regression fixture does not exceed the actual per-item limit")
	}
	if err := client.Save(tokens); err != nil {
		t.Fatalf("large OAuth grant: %v", err)
	}
	for key, item := range vault.data {
		if len(item) > systemSecretLimit {
			t.Errorf("entry %s exceeds the native size limit", key)
		}
	}
	// A fresh wrapper/client has no in-memory grant to hide persistence errors.
	reopened := NewChunkedStore(vault, systemSecretLimit)
	saved, err := reopened.Get(tokensKey)
	if err != nil || !bytes.Equal(saved, data) {
		t.Fatal("large grant did not round-trip through secure storage")
	}
	if status, err := NewClient(reopened).Restore(context.Background()); err != nil || !status.Authorized {
		t.Fatalf("restart status=%+v, err=%v", status, err)
	}
	if err := NewClient(reopened).Disconnect(); err != nil {
		t.Fatal(err)
	}
	if len(vault.data) != 0 {
		t.Fatal("disconnect retained the grant or its chunks")
	}
}

func TestFailedChunkCommitKeepsPreviousGrant(t *testing.T) {
	for _, failure := range []string{"chunk", "pointer"} {
		t.Run(failure, func(t *testing.T) {
			vault := &boundedSecrets{data: map[string][]byte{}}
			store := NewChunkedStore(vault, systemSecretLimit)
			client := NewClient(store)
			if err := client.Save(largeTokens()); err != nil {
				t.Fatal(err)
			}
			previous, _ := store.Get(tokensKey)
			entries := len(vault.data)
			if failure == "chunk" {
				vault.failWrite = vault.writes + 2
			} else {
				vault.failKey = tokensKey
			}
			next := largeTokens()
			next.AccessToken += strings.Repeat("n", 1500)
			if err := client.Save(next); !errors.Is(err, ErrStorage) {
				t.Fatalf("save error=%v", err)
			}
			saved, err := store.Get(tokensKey)
			if err != nil || !bytes.Equal(saved, previous) || len(vault.data) != entries {
				t.Fatal("failed update destroyed the committed grant or left new chunks")
			}
			vault.failKey, vault.failWrite = "", 0
			vault.failDelete = tokensKey
			if err := client.Disconnect(); !errors.Is(err, ErrStorage) {
				t.Fatalf("delete error=%v", err)
			}
			if saved, err := store.Get(tokensKey); err != nil || !bytes.Equal(saved, previous) {
				t.Fatal("failed removal destroyed the committed grant")
			}
			vault.failDelete = ""
			if err := client.Save(testTokens()); err != nil {
				t.Fatal(err)
			}
			if len(vault.data) != 1 {
				t.Fatal("replacing a chunked grant with a small one retained old chunks")
			}
		})
	}
}

func TestIncompleteChunkedGrantIsNeverAuthorized(t *testing.T) {
	for _, missing := range []bool{false, true} {
		t.Run(map[bool]string{false: "corrupt", true: "missing"}[missing], func(t *testing.T) {
			vault := &boundedSecrets{data: map[string][]byte{}}
			store := NewChunkedStore(vault, systemSecretLimit)
			if err := NewClient(store).Save(largeTokens()); err != nil {
				t.Fatal(err)
			}
			for key, value := range vault.data {
				if key == tokensKey {
					continue
				}
				if missing {
					delete(vault.data, key)
				} else {
					value[0] ^= 1
				}
				break
			}
			status, err := NewClient(NewChunkedStore(vault, systemSecretLimit)).Restore(context.Background())
			if !errors.Is(err, ErrStorage) || status.Authorized {
				t.Fatalf("status=%+v, err=%v", status, err)
			}
		})
	}
}
