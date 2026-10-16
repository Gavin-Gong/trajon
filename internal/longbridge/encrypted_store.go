package longbridge

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
)

const (
	encryptionKeyName = "longbridge-storage-key-v1"
	encryptedPrefix   = "trajon-encrypted-credentials-v1\n"
	deletedPayload    = encryptedPrefix + "deleted\n"
)

// NewEncryptedStore stores AES-256-GCM ciphertext locally while keeping its
// only key in the app-scoped system vault. The key is loaded once per instance.
// legacy is read only when the corresponding new-format file does not exist.
// The directory callback and every vault/file operation run on the caller's
// worker, not during construction on the UI thread.
func NewEncryptedStore(vault, legacy Store, directory func() (string, error)) Store {
	return &encryptedStore{vault: vault, legacy: legacy, directory: directory, writeFile: writeCredentialFile}
}

type encryptedStore struct {
	mu        sync.Mutex
	vault     Store
	legacy    Store
	directory func() (string, error)
	key       []byte
	writeFile func(string, []byte) error
}

func (s *encryptedStore) Get(key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	path, err := s.path(key)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return s.migrate(key, path)
	}
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(io.LimitReader(file, maxChunkedSecretSize+1024))
	file.Close()
	if err != nil {
		return nil, err
	}
	if bytes.Equal(data, []byte(deletedPayload)) {
		// A durable absence marker prevents a failed legacy cleanup from
		// reviving old tokens. It contains no secret and needs no vault read.
		return nil, nil
	}
	sealed, ok := bytes.CutPrefix(data, []byte(encryptedPrefix))
	if !ok || len(data) >= maxChunkedSecretSize+1024 {
		return nil, errors.New("longbridge: invalid encrypted credential file")
	}
	aead, err := s.aead(false)
	if err != nil {
		return nil, err
	}
	if len(sealed) < aead.NonceSize()+aead.Overhead() {
		return nil, errors.New("longbridge: incomplete encrypted credential file")
	}
	nonce := sealed[:aead.NonceSize()]
	value, err := aead.Open(nil, nonce, sealed[aead.NonceSize():], []byte(encryptedPrefix+key))
	if err != nil {
		// Never fall back to stale legacy data after a new-format file exists.
		return nil, errors.New("longbridge: encrypted credential integrity check failed")
	}
	return value, nil
}

func (s *encryptedStore) Set(key string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	path, err := s.path(key)
	if err != nil {
		return err
	}
	return s.save(key, path, value)
}

func (s *encryptedStore) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	path, err := s.path(key)
	if err != nil {
		return err
	}
	// Removing the logical grant must not require unlocking its encryption
	// key. Commit absence before best-effort cleanup of old vault records.
	if err := s.writeFile(path, []byte(deletedPayload)); err != nil {
		return err
	}
	if s.legacy != nil {
		_ = s.legacy.Delete(key)
	}
	return nil
}

func (s *encryptedStore) migrate(key, path string) ([]byte, error) {
	if s.legacy == nil {
		return nil, nil
	}
	value, err := s.legacy.Get(key)
	if err != nil || len(value) == 0 {
		return nil, err
	}
	if err := s.save(key, path, value); err != nil {
		return nil, err
	}
	// Only retire the old reference after a complete, synced replacement.
	// Cleanup failure cannot undo a successfully migrated grant.
	_ = s.legacy.Delete(key)
	return value, nil
}

func (s *encryptedStore) save(key, path string, value []byte) error {
	if len(value) > maxChunkedSecretSize {
		return errors.New("longbridge: credential payload exceeds storage limit")
	}
	aead, err := s.aead(true)
	if err != nil {
		return err
	}
	nonce := make([]byte, aead.NonceSize())
	_, _ = rand.Read(nonce)
	sealed := aead.Seal(nil, nonce, value, []byte(encryptedPrefix+key))
	data := append([]byte(encryptedPrefix), nonce...)
	data = append(data, sealed...)
	return s.writeFile(path, data)
}

func (s *encryptedStore) aead(create bool) (cipher.AEAD, error) {
	if len(s.key) == 0 {
		key, err := s.vault.Get(encryptionKeyName)
		if err != nil {
			return nil, err
		}
		if len(key) == 0 && create {
			exists, err := s.hasEncryptedPayload()
			if err != nil {
				return nil, err
			}
			if exists {
				return nil, errors.New("longbridge: encryption key missing for saved credentials")
			}
			key = make([]byte, 32)
			_, _ = rand.Read(key)
			if err := s.vault.Set(encryptionKeyName, key); err != nil {
				return nil, err
			}
		}
		if len(key) != 32 {
			return nil, errors.New("longbridge: credential encryption key is missing or invalid")
		}
		s.key = bytes.Clone(key)
	}
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// A missing key must never be silently replaced while ciphertext still relies
// on it (including the other record). Empty tombstones contain no secret.
func (s *encryptedStore) hasEncryptedPayload() (bool, error) {
	for _, key := range []string{tokensKey, clientKey} {
		path, err := s.path(key)
		if err != nil {
			return false, err
		}
		file, err := os.Open(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return false, err
		}
		prefix, err := io.ReadAll(io.LimitReader(file, int64(len(deletedPayload)+1)))
		file.Close()
		if err != nil {
			return false, err
		}
		if !bytes.Equal(prefix, []byte(deletedPayload)) {
			return true, nil
		}
	}
	return false, nil
}

func (s *encryptedStore) path(key string) (string, error) {
	if key != tokensKey && key != clientKey {
		return "", errors.New("longbridge: unknown credential record")
	}
	dir, err := s.directory()
	if err != nil {
		return "", err
	}
	if dir == "" {
		return "", errors.New("longbridge: missing credential directory")
	}
	return filepath.Join(dir, key+".enc"), nil
}

func writeCredentialFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, ".credential-*") // 0600; plaintext is never written.
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
