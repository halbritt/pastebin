package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"pastebin/internal/paste"
)

var testNow = time.Date(2026, 6, 26, 23, 0, 0, 0, time.UTC)

type recordingStore struct {
	createFunc func(paste.CreateRequest) (paste.Paste, error)
	getFunc    func(string, time.Time) (paste.Paste, error)
}

func (s *recordingStore) Create(_ context.Context, req paste.CreateRequest) (paste.Paste, error) {
	return s.createFunc(req)
}

func (s *recordingStore) Get(_ context.Context, code string, now time.Time) (paste.Paste, error) {
	return s.getFunc(code, now)
}

func (s *recordingStore) CleanupExpired(context.Context, time.Time) (int64, error) {
	return 0, nil
}

func TestPostRootPlainTextReceiptIncludesPasteURL(t *testing.T) {
	store := &recordingStore{
		createFunc: func(req paste.CreateRequest) (paste.Paste, error) {
			if string(req.Content) != "hello" {
				t.Fatalf("content = %q, want hello", req.Content)
			}
			if req.TTL != time.Hour {
				t.Fatalf("ttl = %v, want 1h", req.TTL)
			}
			if !req.Now.Equal(testNow) {
				t.Fatalf("now = %v, want %v", req.Now, testNow)
			}
			return paste.Paste{
				Code:      "abc123",
				Content:   req.Content,
				CreatedAt: req.Now,
				ExpiresAt: req.Now.Add(req.TTL),
				Size:      int64(len(req.Content)),
			}, nil
		},
	}
	handler := testServer(t, store, 1024)
	request := httptest.NewRequest(http.MethodPost, "/?expires=1h", strings.NewReader("hello"))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}
	if response.Body.String() != "https://paste.example.ts.net/p/abc123\n" {
		t.Fatalf("body = %q", response.Body.String())
	}
}

func TestPostRootJSONReceiptIncludesRawURL(t *testing.T) {
	store := &recordingStore{
		createFunc: func(req paste.CreateRequest) (paste.Paste, error) {
			return paste.Paste{
				Code:      "abc123",
				Content:   req.Content,
				CreatedAt: req.Now,
				ExpiresAt: req.Now.Add(req.TTL),
				Size:      int64(len(req.Content)),
			}, nil
		},
	}
	handler := testServer(t, store, 1024)
	form := url.Values{"content": {"hello"}, "expires": {"7d"}}
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}
	var receipt Receipt
	if err := json.NewDecoder(response.Body).Decode(&receipt); err != nil {
		t.Fatalf("decode receipt: %v", err)
	}
	if receipt.URL != "https://paste.example.ts.net/p/abc123" {
		t.Fatalf("url = %q", receipt.URL)
	}
	if receipt.RawURL != "https://paste.example.ts.net/raw/abc123" {
		t.Fatalf("raw_url = %q", receipt.RawURL)
	}
	if receipt.Code != "abc123" || receipt.Size != 5 {
		t.Fatalf("receipt = %+v", receipt)
	}
}

func TestPostRootRejectsInvalidPasteRequests(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		body       []byte
		maxBytes   int64
		wantStatus int
	}{
		{name: "empty", target: "/", body: nil, maxBytes: 10, wantStatus: http.StatusBadRequest},
		{name: "invalid utf8", target: "/", body: []byte{0xff}, maxBytes: 10, wantStatus: http.StatusBadRequest},
		{name: "too large", target: "/", body: []byte("hello"), maxBytes: 4, wantStatus: http.StatusRequestEntityTooLarge},
		{name: "invalid expires", target: "/?expires=2d", body: []byte("hello"), maxBytes: 10, wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &recordingStore{
				createFunc: func(paste.CreateRequest) (paste.Paste, error) {
					t.Fatal("store should not be called")
					return paste.Paste{}, nil
				},
			}
			handler := testServer(t, store, tt.maxBytes)
			request := httptest.NewRequest(http.MethodPost, tt.target, bytes.NewReader(tt.body))
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body %q", response.Code, tt.wantStatus, response.Body.String())
			}
		})
	}
}

func TestRawPastePreservesStoredBytesAndNormalizesCode(t *testing.T) {
	raw := []byte("line 1\r\nline 2\t  ")
	store := &recordingStore{
		getFunc: func(code string, now time.Time) (paste.Paste, error) {
			if code != "abc123" {
				t.Fatalf("code = %q, want abc123", code)
			}
			if !now.Equal(testNow) {
				t.Fatalf("now = %v, want %v", now, testNow)
			}
			return paste.Paste{
				Code:      code,
				Content:   raw,
				CreatedAt: testNow,
				ExpiresAt: testNow.Add(paste.DefaultTTL),
				Size:      int64(len(raw)),
			}, nil
		},
	}
	handler := testServer(t, store, 1024)
	request := httptest.NewRequest(http.MethodGet, "/raw/ABC123", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !bytes.Equal(response.Body.Bytes(), raw) {
		t.Fatalf("raw body = %q, want %q", response.Body.Bytes(), raw)
	}
}

func TestRetrievalMapsExpiredAndUnknownPastes(t *testing.T) {
	store := &recordingStore{
		getFunc: func(code string, _ time.Time) (paste.Paste, error) {
			switch code {
			case "old":
				return paste.Paste{}, paste.ErrExpired
			case "missing":
				return paste.Paste{}, paste.ErrNotFound
			default:
				return paste.Paste{}, errors.New("unexpected code")
			}
		},
	}
	handler := testServer(t, store, 1024)

	for _, tt := range []struct {
		path       string
		wantStatus int
	}{
		{path: "/p/old", wantStatus: http.StatusGone},
		{path: "/raw/missing", wantStatus: http.StatusNotFound},
	} {
		t.Run(tt.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, tt.path, nil)

			handler.ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, tt.wantStatus)
			}
		})
	}
}

func TestPasteViewRendersMarkdownSafelyAndShowsMetadata(t *testing.T) {
	content := []byte("# Heading\n\n- item\n\n<script>alert(\"x\")</script>\n\n[bad](javascript:alert(1))")
	store := &recordingStore{
		getFunc: func(code string, _ time.Time) (paste.Paste, error) {
			return paste.Paste{
				Code:      code,
				Content:   content,
				CreatedAt: testNow,
				ExpiresAt: testNow.Add(time.Hour),
				Size:      int64(len(content)),
			}, nil
		},
	}
	handler := testServer(t, store, 1024)
	request := httptest.NewRequest(http.MethodGet, "/p/abc123", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	body := response.Body.String()
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if strings.Contains(body, `<script>alert("x")</script>`) {
		t.Fatalf("body contains unescaped script: %s", body)
	}
	if strings.Contains(body, "javascript:alert") {
		t.Fatalf("body contains unsafe link: %s", body)
	}
	for _, want := range []string{"<h1>Heading</h1>", "<li>item</li>", "Created", "Expires", "74 bytes", `href="https://paste.example.ts.net/raw/abc123"`, "Copy raw text", "Copy raw URL"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q", want)
		}
	}
}

func TestPasteViewRendersMarkdownTables(t *testing.T) {
	content := []byte("| Header 1 | Header 2 |\n| :--- | ---: |\n| Cell 1 | Cell 2 |")
	store := &recordingStore{
		getFunc: func(code string, _ time.Time) (paste.Paste, error) {
			return paste.Paste{
				Code:      code,
				Content:   content,
				CreatedAt: testNow,
				ExpiresAt: testNow.Add(time.Hour),
				Size:      int64(len(content)),
			}, nil
		},
	}
	handler := testServer(t, store, 1024)
	request := httptest.NewRequest(http.MethodGet, "/p/table123", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	body := response.Body.String()
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	for _, want := range []string{"<table>", "<thead>", "<tbody>", "<th>Header 1</th>", `<td data-label="Header 1">Cell 1</td>`, `<td data-label="Header 2">Cell 2</td>`} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing expected table tag %q in output: %s", want, body)
		}
	}
}

func TestPasteViewMarksWideMarkdownTables(t *testing.T) {
	content := []byte("| One | Two | Three | Four | Five |\n| --- | --- | --- | --- | --- |\n| A | B | C | D | E |")
	store := &recordingStore{
		getFunc: func(code string, _ time.Time) (paste.Paste, error) {
			return paste.Paste{
				Code:      code,
				Content:   content,
				CreatedAt: testNow,
				ExpiresAt: testNow.Add(time.Hour),
				Size:      int64(len(content)),
			}, nil
		},
	}
	handler := testServer(t, store, 1024)
	request := httptest.NewRequest(http.MethodGet, "/p/wide123", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	body := response.Body.String()
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	for _, want := range []string{`<table class="table-wide">`, `<td data-label="One">A</td>`, `<td data-label="Five">E</td>`} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing expected wide table output %q in output: %s", want, body)
		}
	}
}

func TestPostRootJSONCreateBody(t *testing.T) {
	store := &recordingStore{
		createFunc: func(req paste.CreateRequest) (paste.Paste, error) {
			if string(req.Content) != "json\r\nbody  " {
				t.Fatalf("content = %q, want JSON content", req.Content)
			}
			if req.TTL != time.Hour {
				t.Fatalf("ttl = %v, want 1h", req.TTL)
			}
			return paste.Paste{
				Code:      "json123",
				Content:   req.Content,
				CreatedAt: req.Now,
				ExpiresAt: req.Now.Add(req.TTL),
				Size:      int64(len(req.Content)),
			}, nil
		},
	}
	handler := testServer(t, store, 1024)
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"content":"json\r\nbody  ","expires":"1h"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body %q", response.Code, http.StatusCreated, response.Body.String())
	}
	var receipt Receipt
	if err := json.NewDecoder(response.Body).Decode(&receipt); err != nil {
		t.Fatalf("decode receipt: %v", err)
	}
	if receipt.URL != "https://paste.example.ts.net/p/json123" || receipt.RawURL != "https://paste.example.ts.net/raw/json123" {
		t.Fatalf("receipt = %+v", receipt)
	}
}

func TestPublicHostRejectsPasteCreation(t *testing.T) {
	createCalled := false
	store := &recordingStore{
		createFunc: func(req paste.CreateRequest) (paste.Paste, error) {
			createCalled = true
			return paste.Paste{}, nil
		},
	}
	server, err := New(Config{
		Store:      store,
		PublicHost: "pastebin.harm.org",
	})
	if err != nil {
		t.Fatalf("create server: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "https://pastebin.harm.org/", strings.NewReader("public write"))
	request.Host = "PASTEBIN.HARM.ORG:443"
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
	if createCalled {
		t.Fatal("public request reached Paste creation")
	}
}

func TestPublishingHostCreatesDocumentWhenPublicHostIsConfigured(t *testing.T) {
	content := []byte("explicit public document")
	createCalled := false
	server, err := New(Config{
		Store: &recordingStore{
			createFunc: func(req paste.CreateRequest) (paste.Paste, error) {
				createCalled = true
				return paste.Paste{
					Code:      "public123",
					Content:   req.Content,
					CreatedAt: testNow,
					ExpiresAt: testNow.Add(time.Hour),
					Size:      int64(len(req.Content)),
				}, nil
			},
		},
		BaseURL:    "https://pastebin.harm.org",
		PublicHost: "pastebin.harm.org",
	})
	if err != nil {
		t.Fatalf("create server: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "https://proximal.tail0ecc2e.ts.net:18081/", bytes.NewReader(content))
	request.Header.Set("Accept", "application/json")
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusCreated, response.Body.String())
	}
	if !createCalled {
		t.Fatal("publishing request did not reach document creation")
	}
	if !strings.Contains(response.Body.String(), `"url":"https://pastebin.harm.org/p/public123"`) {
		t.Fatalf("publication receipt does not use public Document URL: %s", response.Body.String())
	}
}

func TestPublicHostShowsReadOnlyLandingPage(t *testing.T) {
	server, err := New(Config{
		Store:      &recordingStore{},
		PublicHost: "pastebin.harm.org",
	})
	if err != nil {
		t.Fatalf("create server: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "https://pastebin.harm.org/", nil)
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	body := response.Body.String()
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if strings.Contains(body, `id="create-form"`) {
		t.Fatalf("public landing page exposes Paste creation form: %s", body)
	}
	if !strings.Contains(body, "Document publication is available only inside the private tailnet.") {
		t.Fatalf("public landing page does not explain read-only access: %s", body)
	}
}

func TestPublicHostReadsPasteWithoutLeakingBearerURL(t *testing.T) {
	content := []byte("public read")
	server, err := New(Config{
		Store: &recordingStore{
			getFunc: func(code string, _ time.Time) (paste.Paste, error) {
				return paste.Paste{
					Code:      code,
					Content:   content,
					CreatedAt: testNow,
					ExpiresAt: testNow.Add(time.Hour),
					Size:      int64(len(content)),
				}, nil
			},
		},
		PublicHost: "pastebin.harm.org",
	})
	if err != nil {
		t.Fatalf("create server: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "https://pastebin.harm.org/p/public123", nil)
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), "public read") {
		t.Fatalf("public Paste View missing content: %s", response.Body.String())
	}
	wantHeaders := map[string]string{
		"Cache-Control":   "private, no-store",
		"Referrer-Policy": "no-referrer",
		"X-Robots-Tag":    "noindex, nofollow, noarchive",
	}
	for name, want := range wantHeaders {
		if got := response.Header().Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}

func TestPasteViewJSON(t *testing.T) {
	content := []byte("json view")
	store := &recordingStore{
		getFunc: func(code string, _ time.Time) (paste.Paste, error) {
			return paste.Paste{
				Code:      code,
				Content:   content,
				CreatedAt: testNow,
				ExpiresAt: testNow.Add(time.Hour),
				Size:      int64(len(content)),
			}, nil
		},
	}
	handler := testServer(t, store, 1024)
	request := httptest.NewRequest(http.MethodGet, "/p/ABC123", nil)
	request.Header.Set("Accept", "application/json")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var body struct {
		URL     string `json:"url"`
		RawURL  string `json:"raw_url"`
		Code    string `json:"code"`
		Content string `json:"content"`
		Size    int64  `json:"size"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode paste JSON: %v", err)
	}
	if body.URL != "https://paste.example.ts.net/p/abc123" || body.RawURL != "https://paste.example.ts.net/raw/abc123" || body.Code != "abc123" || body.Content != string(content) || body.Size != int64(len(content)) {
		t.Fatalf("paste JSON = %+v", body)
	}
}

func TestHomeAndHealthRoutes(t *testing.T) {
	handler := testServer(t, &recordingStore{}, 1024)

	home := httptest.NewRecorder()
	handler.ServeHTTP(home, httptest.NewRequest(http.MethodGet, "/", nil))
	if home.Code != http.StatusOK {
		t.Fatalf("home status = %d, want %d", home.Code, http.StatusOK)
	}
	for _, want := range []string{"<textarea", "<select", `value="1h"`, `value="30d"`} {
		if !strings.Contains(home.Body.String(), want) {
			t.Fatalf("home missing %q", want)
		}
	}

	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if health.Code != http.StatusOK || health.Body.String() != "ok\n" {
		t.Fatalf("health = (%d, %q), want (200, ok newline)", health.Code, health.Body.String())
	}
}

func testServer(t *testing.T, store paste.Store, maxBytes int64) *Server {
	t.Helper()
	handler, err := New(Config{
		Store:    store,
		BaseURL:  "https://paste.example.ts.net",
		MaxBytes: maxBytes,
		Now:      func() time.Time { return testNow },
	})
	if err != nil {
		t.Fatalf("New server: %v", err)
	}
	return handler
}
