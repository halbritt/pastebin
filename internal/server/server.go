package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"pastebin/internal/paste"
)

type Config struct {
	Store      paste.Store
	BaseURL    string
	MaxBytes   int64
	DefaultTTL time.Duration
	MaxTTL     time.Duration
	Now        func() time.Time
}

type Server struct {
	store      paste.Store
	mux        *http.ServeMux
	baseURL    string
	maxBytes   int64
	defaultTTL time.Duration
	maxTTL     time.Duration
	now        func() time.Time
}

type Receipt struct {
	URL       string    `json:"url"`
	RawURL    string    `json:"raw_url"`
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expires_at"`
	Size      int64     `json:"size"`
}

func New(cfg Config) (*Server, error) {
	if cfg.Store == nil {
		return nil, errors.New("server requires a paste store")
	}
	srv := &Server{
		store:      cfg.Store,
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		maxBytes:   defaultMaxBytes(cfg.MaxBytes),
		defaultTTL: defaultTTL(cfg.DefaultTTL),
		maxTTL:     defaultMaxTTL(cfg.MaxTTL),
		now:        defaultClock(cfg.Now),
	}
	srv.mountRoutes()
	return srv, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) mountRoutes() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.home)
	mux.HandleFunc("POST /{$}", s.createPaste)
	mux.HandleFunc("GET /p/{code}", s.pasteView)
	mux.HandleFunc("GET /raw/{code}", s.rawPaste)
	mux.HandleFunc("GET /healthz", s.health)
	s.mux = mux
}

func (s *Server) home(w http.ResponseWriter, _ *http.Request) {
	s.renderHome(w)
}

func (s *Server) createPaste(w http.ResponseWriter, r *http.Request) {
	content, expiresValue, err := s.creationContent(w, r)
	if err != nil {
		s.writeError(w, r, statusForError(err), err.Error())
		return
	}
	if err := paste.ValidateContent(content, s.maxBytes); err != nil {
		s.writeError(w, r, statusForError(err), err.Error())
		return
	}
	ttl, err := s.expiration(expiresValue)
	if err != nil {
		s.writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	created, err := s.store.Create(r.Context(), paste.CreateRequest{
		Content: content,
		TTL:     ttl,
		Now:     s.now(),
	})
	if err != nil {
		s.writeError(w, r, statusForError(err), err.Error())
		return
	}
	s.writeReceipt(w, r, s.receiptFor(r, created))
}

func (s *Server) pasteView(w http.ResponseWriter, r *http.Request) {
	found, err := s.findPaste(r.Context(), r.PathValue("code"))
	if err != nil {
		s.writeError(w, r, statusForError(err), err.Error())
		return
	}
	if wantsJSON(r) {
		s.writePasteJSON(w, r, found)
		return
	}
	s.renderPaste(w, r, found)
}

func (s *Server) rawPaste(w http.ResponseWriter, r *http.Request) {
	found, err := s.findPaste(r.Context(), r.PathValue("code"))
	if err != nil {
		s.writeError(w, r, statusForError(err), err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(found.Content)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = io.WriteString(w, "ok\n")
}

func (s *Server) findPaste(ctx context.Context, code string) (paste.Paste, error) {
	code = paste.NormalizeCode(code)
	if code == "" {
		return paste.Paste{}, paste.ErrNotFound
	}
	return s.store.Get(ctx, code, s.now())
}

func (s *Server) creationContent(w http.ResponseWriter, r *http.Request) ([]byte, string, error) {
	switch requestMediaType(r.Header.Get("Content-Type")) {
	case "application/x-www-form-urlencoded":
		r.Body = http.MaxBytesReader(w, r.Body, formBodyLimit(s.maxBytes))
		if err := r.ParseForm(); err != nil {
			return nil, "", formReadError(err)
		}
		return []byte(r.PostForm.Get("content")), r.PostForm.Get("expires"), nil
	case "application/json":
		r.Body = http.MaxBytesReader(w, r.Body, formBodyLimit(s.maxBytes))
		var payload struct {
			Content string `json:"content"`
			Expires string `json:"expires"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			if strings.Contains(err.Error(), "request body too large") {
				return nil, "", paste.ErrTooLarge
			}
			return nil, "", err
		}
		return []byte(payload.Content), payload.Expires, nil
	}

	r.Body = http.MaxBytesReader(w, r.Body, s.maxBytes+1)
	content, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, "", paste.ErrTooLarge
	}
	if int64(len(content)) > s.maxBytes {
		return nil, "", paste.ErrTooLarge
	}
	return content, r.URL.Query().Get("expires"), nil
}

func (s *Server) expiration(value string) (time.Duration, error) {
	if strings.TrimSpace(value) == "" {
		return paste.ValidateTTL(0, s.defaultTTL, s.maxTTL)
	}
	ttl, err := paste.ParseAllowedTTL(value)
	if err != nil {
		return 0, err
	}
	return paste.ValidateTTL(ttl, s.defaultTTL, s.maxTTL)
}

func (s *Server) writeReceipt(w http.ResponseWriter, r *http.Request, receipt Receipt) {
	if wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(receipt)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_, _ = fmt.Fprintf(w, "%s\n", receipt.URL)
}

func (s *Server) writePasteJSON(w http.ResponseWriter, r *http.Request, found paste.Paste) {
	receipt := s.receiptFor(r, found)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(struct {
		URL       string    `json:"url"`
		RawURL    string    `json:"raw_url"`
		Code      string    `json:"code"`
		Content   string    `json:"content"`
		CreatedAt time.Time `json:"created_at"`
		ExpiresAt time.Time `json:"expires_at"`
		Size      int64     `json:"size"`
	}{
		URL:       receipt.URL,
		RawURL:    receipt.RawURL,
		Code:      receipt.Code,
		Content:   string(found.Content),
		CreatedAt: found.CreatedAt.UTC(),
		ExpiresAt: receipt.ExpiresAt,
		Size:      receipt.Size,
	})
}

func (s *Server) writeError(w http.ResponseWriter, r *http.Request, status int, message string) {
	if wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
		return
	}
	http.Error(w, message, status)
}

func (s *Server) receiptFor(r *http.Request, found paste.Paste) Receipt {
	size := found.Size
	if size == 0 {
		size = int64(len(found.Content))
	}
	baseURL := s.absoluteBaseURL(r)
	return Receipt{
		URL:       pasteViewURL(baseURL, found.Code),
		RawURL:    rawPasteURL(baseURL, found.Code),
		Code:      found.Code,
		ExpiresAt: found.ExpiresAt.UTC(),
		Size:      size,
	}
}

func (s *Server) absoluteBaseURL(r *http.Request) string {
	if s.baseURL != "" {
		return s.baseURL
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := r.Header.Get("X-Forwarded-Proto"); forwarded == "http" || forwarded == "https" {
		scheme = forwarded
	}
	return scheme + "://" + r.Host
}

func pasteViewURL(baseURL, code string) string {
	return strings.TrimRight(baseURL, "/") + "/p/" + url.PathEscape(code)
}

func rawPasteURL(baseURL, code string) string {
	return strings.TrimRight(baseURL, "/") + "/raw/" + url.PathEscape(code)
}

func requestMediaType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if cut := strings.Index(value, ";"); cut >= 0 {
		value = value[:cut]
	}
	return strings.TrimSpace(value)
}

func formBodyLimit(maxBytes int64) int64 {
	const overhead int64 = 4096
	if maxBytes > (1<<62)/4 {
		return maxBytes
	}
	return maxBytes*4 + overhead
}

func formReadError(err error) error {
	if strings.Contains(err.Error(), "request body too large") {
		return paste.ErrTooLarge
	}
	return err
}

func statusForError(err error) int {
	switch {
	case errors.Is(err, paste.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, paste.ErrExpired):
		return http.StatusGone
	case errors.Is(err, paste.ErrTooLarge):
		return http.StatusRequestEntityTooLarge
	case errors.Is(err, paste.ErrEmpty), errors.Is(err, paste.ErrInvalidUTF8), errors.Is(err, paste.ErrInvalidTTL):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func wantsJSON(r *http.Request) bool {
	for _, value := range strings.Split(r.Header.Get("Accept"), ",") {
		mediaType := requestMediaType(value)
		if mediaType == "application/json" || strings.HasSuffix(mediaType, "+json") {
			return true
		}
	}
	return false
}

func defaultMaxBytes(value int64) int64 {
	if value <= 0 {
		return paste.DefaultMaxBytes
	}
	return value
}

func defaultTTL(value time.Duration) time.Duration {
	if value <= 0 {
		return paste.DefaultTTL
	}
	return value
}

func defaultMaxTTL(value time.Duration) time.Duration {
	if value <= 0 {
		return paste.MaxTTL
	}
	return value
}

func defaultClock(clock func() time.Time) func() time.Time {
	if clock != nil {
		return clock
	}
	return func() time.Time { return time.Now().UTC() }
}
