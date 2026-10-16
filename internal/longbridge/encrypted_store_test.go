package longbridge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

type countingVault struct {
	boundedSecrets
	reads int
}

func (s *countingVault) Get(key string) ([]byte, error) {
	s.reads++
	return s.boundedSecrets.Get(key)
}

func encryptedFixture(t *testing.T) (*countingVault, Store, func() (string, error)) {
	t.Helper()
	vault := &countingVault{boundedSecrets: boundedSecrets{data: map[string][]byte{}}}
	legacy := NewChunkedStore(vault, systemSecretLimit)
	path := filepath.Join(t.TempDir(), "longbridge")
	return vault, legacy, func() (string, error) { return path, nil }
}

func TestEncryptedGrantUsesOneVaultReadPerProcess(t *testing.T) {
	vault, legacy, directory := encryptedFixture(t)
	store := NewEncryptedStore(vault, legacy, directory)
	client := NewClient(store)
	grant := largeTokens()
	if err := client.Save(grant); err != nil {
		t.Fatal(err)
	}
	if len(vault.data) != 1 {
		t.Fatalf("vault entries=%d, want one encryption key", len(vault.data))
	}
	vault.reads = 0
	reopened := NewClient(NewEncryptedStore(vault, legacy, directory))
	for range 3 {
		if status, err := reopened.Restore(context.Background()); err != nil || !status.Authorized {
			t.Fatalf("status=%+v, err=%v", status, err)
		}
	}
	if vault.reads != 1 {
		t.Fatalf("vault reads=%d, want one for the entire process", vault.reads)
	}
	dir, _ := directory()
	files, err := os.ReadDir(dir)
	if err != nil || len(files) != 1 {
		t.Fatalf("encrypted files=%d, err=%v", len(files), err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, files[0].Name()))
	if bytes.Contains(data, []byte(grant.AccessToken)) || bytes.Contains(data, []byte(grant.RefreshToken)) {
		t.Fatal("plaintext credentials reached disk")
	}
}

func TestLegacyMigrationPreservesGrantAndDoesNotRepeatVaultReads(t *testing.T) {
	vault, legacy, directory := encryptedFixture(t)
	grant := largeTokens()
	if err := NewClient(legacy).Save(grant); err != nil {
		t.Fatal(err)
	}
	store := NewEncryptedStore(vault, legacy, directory)
	first, err := store.Get(tokensKey)
	if err != nil {
		t.Fatal(err)
	}
	var migrated Tokens
	if err := json.Unmarshal(first, &migrated); err != nil || migrated.AccessToken != grant.AccessToken {
		t.Fatal("migration changed the grant")
	}
	vault.reads = 0
	second, err := NewEncryptedStore(vault, legacy, directory).Get(tokensKey)
	if err != nil || !bytes.Equal(first, second) || vault.reads != 1 {
		t.Fatalf("migration repeated legacy reads: reads=%d, err=%v", vault.reads, err)
	}
	if _, found := vault.data[tokensKey]; found {
		t.Fatal("legacy active reference was not removed after durable migration")
	}
}

func TestRestoreCachesGrantAndDisconnectDoesNotResurrectIt(t *testing.T) {
	vault, legacy, _ := encryptedFixture(t)
	if err := NewClient(legacy).Save(largeTokens()); err != nil {
		t.Fatal(err)
	}
	client := NewClient(legacy)
	vault.reads = 0
	if _, err := client.Restore(context.Background()); err != nil {
		t.Fatal(err)
	}
	first := vault.reads
	var jobs sync.WaitGroup
	for range 3 {
		jobs.Go(func() {
			if status, err := client.Restore(context.Background()); err != nil || !status.Authorized {
				t.Errorf("cached restore status=%+v, err=%v", status, err)
			}
		})
	}
	jobs.Wait()
	if vault.reads != first {
		t.Fatalf("cached grant reread the vault: first=%d, total=%d", first, vault.reads)
	}
	if err := client.Disconnect(); err != nil {
		t.Fatal(err)
	}
	afterDelete := vault.reads
	if status, err := client.Restore(context.Background()); err != nil || status.Authorized || vault.reads != afterDelete {
		t.Fatal("disconnect retained or reread the old grant")
	}
}

func TestEncryptedDisconnectCannotFallBackToStaleLegacyGrant(t *testing.T) {
	vault, legacy, directory := encryptedFixture(t)
	store := NewEncryptedStore(vault, legacy, directory)
	client := NewClient(store)
	if err := client.Save(largeTokens()); err != nil {
		t.Fatal(err)
	}
	// Simulate legacy cleanup being denied. A disconnect must still leave a
	// durable absence marker, never revive that older grant on the next run.
	if err := NewClient(legacy).Save(testTokens()); err != nil {
		t.Fatal(err)
	}
	vault.failDelete = tokensKey
	if err := client.Disconnect(); err != nil {
		t.Fatal(err)
	}
	vault.reads = 0
	status, err := NewClient(NewEncryptedStore(vault, legacy, directory)).Restore(context.Background())
	if err != nil || status.Authorized || vault.reads != 0 {
		t.Fatalf("stale grant revived or empty state read vault: status=%+v, reads=%d, err=%v", status, vault.reads, err)
	}
}

func TestDeniedKeyAccessDoesNotOverwriteSavedEncryptedGrant(t *testing.T) {
	vault, legacy, directory := encryptedFixture(t)
	store := NewEncryptedStore(vault, legacy, directory)
	if err := NewClient(store).Save(largeTokens()); err != nil {
		t.Fatal(err)
	}
	// Replacing the master key with an invalid value must fail closed, without
	// re-registering, migrating a stale copy, or overwriting the payload.
	for key := range vault.data {
		if key != tokensKey {
			vault.data[key] = []byte("invalid key")
		}
	}
	_, err := NewClient(NewEncryptedStore(vault, legacy, directory)).Restore(context.Background())
	if !errors.Is(err, ErrStorage) {
		t.Fatalf("wrong-key error=%v", err)
	}
}
