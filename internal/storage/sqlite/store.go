package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"pastebin/internal/paste"

	_ "modernc.org/sqlite"
)

const (
	defaultMaxCreateAttempts = 16
	driverName               = "sqlite"
)

type codeGenerator func() (string, error)

type Store struct {
	db                *sql.DB
	newCode           codeGenerator
	maxCreateAttempts int
}

var _ paste.Store = (*Store)(nil)

type option func(*Store)

type pasteDraft struct {
	content   []byte
	createdAt time.Time
	expiresAt time.Time
	size      int64
}

func withCodeGenerator(newCode func() (string, error)) option {
	return func(s *Store) {
		if newCode != nil {
			s.newCode = newCode
		}
	}
}

func Open(ctx context.Context, path string) (*Store, error) {
	return open(ctx, path)
}

func open(ctx context.Context, path string, opts ...option) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("sqlite store path is empty")
	}

	db, err := sql.Open(driverName, path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite store: %w", err)
	}
	db.SetMaxOpenConns(1)

	store, err := newStore(ctx, db, opts...)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func New(ctx context.Context, db *sql.DB) (*Store, error) {
	return newStore(ctx, db)
}

func newStore(ctx context.Context, db *sql.DB, opts ...option) (*Store, error) {
	if db == nil {
		return nil, errors.New("sqlite store db is nil")
	}

	store := &Store{
		db:                db,
		newCode:           paste.NewCode,
		maxCreateAttempts: defaultMaxCreateAttempts,
	}
	for _, opt := range opts {
		opt(store)
	}

	if err := store.init(ctx); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) Create(ctx context.Context, req paste.CreateRequest) (paste.Paste, error) {
	draft, err := newPasteDraft(req)
	if err != nil {
		return paste.Paste{}, err
	}

	for attempt := 0; attempt < s.maxCreateAttempts; attempt++ {
		code, err := s.nextCode()
		if err != nil {
			return paste.Paste{}, err
		}
		if code == "" {
			continue
		}

		inserted, err := s.insertPaste(ctx, code, draft)
		if err != nil {
			return paste.Paste{}, err
		}
		if inserted {
			return draft.paste(code), nil
		}
	}

	return paste.Paste{}, fmt.Errorf("create paste: exhausted %d code attempts", s.maxCreateAttempts)
}

func (s *Store) Get(ctx context.Context, code string, now time.Time) (paste.Paste, error) {
	code = paste.NormalizeCode(code)
	if code == "" {
		return paste.Paste{}, paste.ErrNotFound
	}
	now = normalizeTime(now)
	if now.IsZero() {
		now = normalizeTime(time.Now())
	}

	p, err := s.findPaste(ctx, code)
	if err != nil {
		return paste.Paste{}, err
	}
	if !now.Before(p.ExpiresAt) {
		return paste.Paste{}, paste.ErrExpired
	}

	return p, nil
}

func (s *Store) CleanupExpired(ctx context.Context, now time.Time) (int64, error) {
	now = normalizeTime(now)
	if now.IsZero() {
		now = normalizeTime(time.Now())
	}

	res, err := s.db.ExecContext(ctx, `DELETE FROM pastes WHERE expires_at <= ?`, now.UnixNano())
	if err != nil {
		return 0, fmt.Errorf("cleanup expired pastes: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("cleanup expired paste rows affected: %w", err)
	}
	return rows, nil
}

func (s *Store) findPaste(ctx context.Context, code string) (paste.Paste, error) {
	var p paste.Paste
	var createdAtNS, expiresAtNS int64
	err := s.db.QueryRowContext(ctx, `
SELECT code, content, created_at, expires_at, size
FROM pastes
WHERE code = ? COLLATE NOCASE`,
		code).Scan(&p.Code, &p.Content, &createdAtNS, &expiresAtNS, &p.Size)
	if errors.Is(err, sql.ErrNoRows) {
		return paste.Paste{}, paste.ErrNotFound
	}
	if err != nil {
		return paste.Paste{}, fmt.Errorf("get paste: %w", err)
	}

	p.Content = append([]byte(nil), p.Content...)
	p.CreatedAt = time.Unix(0, createdAtNS).UTC()
	p.ExpiresAt = time.Unix(0, expiresAtNS).UTC()

	return p, nil
}

func newPasteDraft(req paste.CreateRequest) (pasteDraft, error) {
	if err := validateContent(req.Content); err != nil {
		return pasteDraft{}, err
	}

	ttl, err := paste.ValidateTTL(req.TTL, paste.DefaultTTL, paste.MaxTTL)
	if err != nil {
		return pasteDraft{}, err
	}

	createdAt := normalizeTime(req.Now)
	if createdAt.IsZero() {
		createdAt = normalizeTime(time.Now())
	}
	content := append([]byte(nil), req.Content...)
	return pasteDraft{
		content:   content,
		createdAt: createdAt,
		expiresAt: createdAt.Add(ttl),
		size:      int64(len(content)),
	}, nil
}

func (s *Store) nextCode() (string, error) {
	code, err := s.newCode()
	if err != nil {
		return "", fmt.Errorf("generate paste code: %w", err)
	}
	return paste.NormalizeCode(code), nil
}

func (s *Store) insertPaste(ctx context.Context, code string, draft pasteDraft) (bool, error) {
	res, err := s.db.ExecContext(ctx, `
INSERT OR IGNORE INTO pastes (code, content, created_at, expires_at, size)
VALUES (?, ?, ?, ?, ?)`,
		code, draft.content, draft.createdAt.UnixNano(), draft.expiresAt.UnixNano(), draft.size)
	if err != nil {
		return false, fmt.Errorf("insert paste: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("insert paste rows affected: %w", err)
	}
	return rows > 0, nil
}

func (d pasteDraft) paste(code string) paste.Paste {
	return paste.Paste{
		Code:      code,
		Content:   append([]byte(nil), d.content...),
		CreatedAt: d.createdAt,
		ExpiresAt: d.expiresAt,
		Size:      d.size,
	}
}

func (s *Store) init(ctx context.Context) error {
	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping sqlite store: %w", err)
	}

	statements := []string{
		`PRAGMA busy_timeout = 5000`,
		`PRAGMA journal_mode = WAL`,
		`CREATE TABLE IF NOT EXISTS pastes (
	code TEXT NOT NULL PRIMARY KEY COLLATE NOCASE,
	content BLOB NOT NULL,
	created_at INTEGER NOT NULL,
	expires_at INTEGER NOT NULL,
	size INTEGER NOT NULL,
	CHECK (length(code) > 0),
	CHECK (code = lower(code)),
	CHECK (size = length(content)),
	CHECK (expires_at > created_at)
)`,
		`CREATE INDEX IF NOT EXISTS pastes_expires_at_idx ON pastes (expires_at)`,
	}

	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("initialize sqlite store: %w", err)
		}
	}
	return nil
}

func validateContent(content []byte) error {
	if len(content) == 0 {
		return paste.ErrEmpty
	}
	return paste.ValidateContent(content, int64(len(content)))
}

func normalizeTime(t time.Time) time.Time {
	if t.IsZero() {
		return time.Time{}
	}
	return time.Unix(0, t.UnixNano()).UTC()
}
