package sqlite

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"pastebin/internal/paste"
)

func TestCreateAndGetRoundTripExactBytes(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t, withCodeGenerator(codeSequence(t, "AbC123")))
	defer closeStore(t, store)

	now := time.Date(2026, 6, 26, 12, 34, 56, 789, time.UTC)
	content := []byte("first line\r\n\tsecond line with trailing spaces  ")

	created, err := store.Create(ctx, paste.CreateRequest{
		Content: content,
		TTL:     time.Hour,
		Now:     now,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Code != "abc123" {
		t.Fatalf("Create() code = %q, want lowercase abc123", created.Code)
	}
	if !bytes.Equal(created.Content, content) {
		t.Fatalf("Create() content = %q, want exact bytes %q", created.Content, content)
	}
	if created.Size != int64(len(content)) {
		t.Fatalf("Create() size = %d, want %d", created.Size, len(content))
	}
	if !created.CreatedAt.Equal(now) {
		t.Fatalf("Create() CreatedAt = %v, want %v", created.CreatedAt, now)
	}
	if !created.ExpiresAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("Create() ExpiresAt = %v, want %v", created.ExpiresAt, now.Add(time.Hour))
	}

	got, err := store.Get(ctx, strings.ToUpper(created.Code), now.Add(time.Minute))
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Code != created.Code {
		t.Fatalf("Get() code = %q, want %q", got.Code, created.Code)
	}
	if !bytes.Equal(got.Content, content) {
		t.Fatalf("Get() content = %q, want exact bytes %q", got.Content, content)
	}
}

func TestOpenCreatesSchema(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "pastebin.db")

	store, err := open(ctx, path, withCodeGenerator(codeSequence(t, "schema1")))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer closeStore(t, store)

	if _, err := store.Create(ctx, paste.CreateRequest{
		Content: []byte("schema created"),
		TTL:     time.Hour,
		Now:     testNow(),
	}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
}

func TestGetReturnsExpiredForKnownExpiredPaste(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t, withCodeGenerator(codeSequence(t, "expires1")))
	defer closeStore(t, store)

	now := testNow()
	created, err := store.Create(ctx, paste.CreateRequest{
		Content: []byte("short lived"),
		TTL:     time.Hour,
		Now:     now,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if _, err := store.Get(ctx, created.Code, created.ExpiresAt.Add(-time.Nanosecond)); err != nil {
		t.Fatalf("Get() before expiration error = %v", err)
	}
	if _, err := store.Get(ctx, created.Code, created.ExpiresAt); !errors.Is(err, paste.ErrExpired) {
		t.Fatalf("Get() at expiration error = %v, want ErrExpired", err)
	}
	if _, err := store.Get(ctx, "missing", created.ExpiresAt); !errors.Is(err, paste.ErrNotFound) {
		t.Fatalf("Get() missing error = %v, want ErrNotFound", err)
	}
}

func TestCleanupExpiredDeletesExpiredRecords(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t, withCodeGenerator(codeSequence(t, "oldpaste", "newpaste")))
	defer closeStore(t, store)

	now := testNow()
	oldPaste, err := store.Create(ctx, paste.CreateRequest{
		Content: []byte("old"),
		TTL:     time.Hour,
		Now:     now,
	})
	if err != nil {
		t.Fatalf("Create(old) error = %v", err)
	}
	newPaste, err := store.Create(ctx, paste.CreateRequest{
		Content: []byte("new"),
		TTL:     24 * time.Hour,
		Now:     now,
	})
	if err != nil {
		t.Fatalf("Create(new) error = %v", err)
	}

	deleted, err := store.CleanupExpired(ctx, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("CleanupExpired() error = %v", err)
	}
	if deleted != 1 {
		t.Fatalf("CleanupExpired() deleted = %d, want 1", deleted)
	}
	if _, err := store.Get(ctx, oldPaste.Code, now.Add(time.Hour)); !errors.Is(err, paste.ErrNotFound) {
		t.Fatalf("Get(old) error = %v, want ErrNotFound after cleanup", err)
	}
	if _, err := store.Get(ctx, newPaste.Code, now.Add(time.Hour)); err != nil {
		t.Fatalf("Get(new) error = %v", err)
	}
}

func TestCreateRetriesOnCodeCollision(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t, withCodeGenerator(codeSequence(t, "samecode", "samecode", "freshcode")))
	defer closeStore(t, store)

	now := testNow()
	first, err := store.Create(ctx, paste.CreateRequest{
		Content: []byte("first"),
		TTL:     time.Hour,
		Now:     now,
	})
	if err != nil {
		t.Fatalf("Create(first) error = %v", err)
	}
	second, err := store.Create(ctx, paste.CreateRequest{
		Content: []byte("second"),
		TTL:     time.Hour,
		Now:     now,
	})
	if err != nil {
		t.Fatalf("Create(second) error = %v", err)
	}

	if first.Code != "samecode" {
		t.Fatalf("Create(first) code = %q, want samecode", first.Code)
	}
	if second.Code != "freshcode" {
		t.Fatalf("Create(second) code = %q, want freshcode after retry", second.Code)
	}
}

func TestCreateRejectsEmptyAndInvalidUTF8(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer closeStore(t, store)

	tests := []struct {
		name    string
		content []byte
		wantErr error
	}{
		{name: "empty", content: nil, wantErr: paste.ErrEmpty},
		{name: "invalid utf8", content: []byte{0xff}, wantErr: paste.ErrInvalidUTF8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := store.Create(ctx, paste.CreateRequest{
				Content: tt.content,
				TTL:     time.Hour,
				Now:     testNow(),
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Create() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func openTestStore(t *testing.T, opts ...option) *Store {
	t.Helper()

	store, err := open(context.Background(), filepath.Join(t.TempDir(), "pastebin.db"), opts...)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	return store
}

func closeStore(t *testing.T, store *Store) {
	t.Helper()

	if err := store.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func codeSequence(t *testing.T, codes ...string) func() (string, error) {
	t.Helper()

	next := 0
	return func() (string, error) {
		if next >= len(codes) {
			return "", fmt.Errorf("test code generator exhausted")
		}
		code := codes[next]
		next++
		return code, nil
	}
}

func testNow() time.Time {
	return time.Date(2026, 6, 26, 0, 0, 0, 123, time.UTC)
}
