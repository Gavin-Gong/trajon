package longbridge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestEncryptedCommitFailureKeepsDiskAndActiveGrant(t *testing.T) {
	vault, legacy, directory := encryptedFixture(t)
	store := NewEncryptedStore(vault, legacy, directory).(*encryptedStore)
	client := NewClient(store)
	if err := client.Save(testTokens()); err != nil {
		t.Fatal(err)
	}
	dir, _ := directory()
	path := filepath.Join(dir, tokensKey+".enc")
	before, _ := os.ReadFile(path)
	store.writeFile = func(string, []byte) error { return errors.New("disk full") }
	if err := client.Save(largeTokens()); !errors.Is(err, ErrStorage) {
		t.Fatalf("replacement error=%v", err)
	}
	if err := client.Disconnect(); !errors.Is(err, ErrStorage) {
		t.Fatalf("disconnect error=%v", err)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("failed commit changed the saved grant")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer old-access" {
			t.Error("failed commit changed the active grant")
		}
		io.WriteString(w, `{"code":0,"data":{}}`)
	}))
	defer server.Close()
	client.baseURL = server.URL
	if err := client.Request(context.Background(), "/quote/static-info", nil, &struct{}{}); err != nil {
		t.Fatal(err)
	}
}

func TestFailedMigrationDoesNotRemoveLegacyGrant(t *testing.T) {
	vault, legacy, directory := encryptedFixture(t)
	if err := NewClient(legacy).Save(largeTokens()); err != nil {
		t.Fatal(err)
	}
	before, _ := legacy.Get(tokensKey)
	store := NewEncryptedStore(vault, legacy, directory).(*encryptedStore)
	store.writeFile = func(string, []byte) error { return errors.New("disk full") }
	if _, err := store.Get(tokensKey); err == nil {
		t.Fatal("migration ignored its durable-save failure")
	}
	after, err := legacy.Get(tokensKey)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("failed migration destroyed the legacy grant")
	}
	store.writeFile = writeCredentialFile
	if migrated, err := store.Get(tokensKey); err != nil || !bytes.Equal(before, migrated) {
		t.Fatalf("migration retry failed: %v", err)
	}
}

func TestCiphertextTamperingOrMissingKeyNeverLoadsStaleLegacyData(t *testing.T) {
	for _, missingKey := range []bool{false, true} {
		t.Run(map[bool]string{false: "tamper", true: "missing-key"}[missingKey], func(t *testing.T) {
			vault, legacy, directory := encryptedFixture(t)
			if err := NewClient(NewEncryptedStore(vault, legacy, directory)).Save(largeTokens()); err != nil {
				t.Fatal(err)
			}
			if err := NewClient(legacy).Save(testTokens()); err != nil {
				t.Fatal(err)
			}
			dir, _ := directory()
			path := filepath.Join(dir, tokensKey+".enc")
			data, _ := os.ReadFile(path)
			if missingKey {
				delete(vault.data, encryptionKeyName)
			} else {
				data[len(data)-1] ^= 1
				if err := os.WriteFile(path, data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			client := NewClient(NewEncryptedStore(vault, legacy, directory))
			status, err := client.Restore(context.Background())
			if !errors.Is(err, ErrStorage) || status.Authorized {
				t.Fatalf("status=%+v, err=%v", status, err)
			}
			if missingKey {
				if err := client.Save(testTokens()); !errors.Is(err, ErrStorage) {
					t.Fatalf("saved over a missing encryption key: %v", err)
				}
				if _, exists := vault.data[encryptionKeyName]; exists {
					t.Fatal("silently regenerated a missing key")
				}
			}
			after, _ := os.ReadFile(path)
			if !bytes.Equal(after, data) {
				t.Fatal("error handling overwrote the encrypted grant")
			}
			legacyData, _ := legacy.Get(tokensKey)
			var old Tokens
			if err := json.Unmarshal(legacyData, &old); err != nil || old.AccessToken != "old-access" {
				t.Fatal("failed read destroyed unrelated legacy data")
			}
		})
	}
}
