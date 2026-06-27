package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCreatePlainPost(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/" {
			t.Fatalf("path = %q, want /", r.URL.Path)
		}
		if got := r.URL.Query().Get("expires"); got != "1h" {
			t.Fatalf("expires = %q, want 1h", got)
		}
		if got := r.Header.Get("Accept"); got != "text/plain" {
			t.Fatalf("Accept = %q, want text/plain", got)
		}
		if got := r.Header.Get("Content-Type"); got != "text/plain; charset=utf-8" {
			t.Fatalf("Content-Type = %q, want text/plain; charset=utf-8", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != "hello\n" {
			t.Fatalf("body = %q, want hello newline", string(body))
		}
		fmt.Fprintln(w, server.URL+"/p/abc123")
	}))
	defer server.Close()

	api, err := New(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := api.Create(context.Background(), CreateOptions{
		Content: []byte("hello\n"),
		Expires: "1h",
	})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.URL != server.URL+"/p/abc123" {
		t.Fatalf("URL = %q, want server paste URL", receipt.URL)
	}
}

func TestCreateJSONRequestResponse(t *testing.T) {
	expiresAt := time.Date(2026, 6, 27, 0, 0, 0, 0, time.UTC)
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Fatalf("Accept = %q, want application/json", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q, want application/json", got)
		}
		var req struct {
			Content string `json:"content"`
			Expires string `json:"expires"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.Content != "line 1\r\nlast  " {
			t.Fatalf("content = %q, want exact text", req.Content)
		}
		if req.Expires != "7d" {
			t.Fatalf("expires = %q, want 7d", req.Expires)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Receipt{
			URL:       server.URL + "/p/json1",
			RawURL:    server.URL + "/raw/json1",
			Code:      "json1",
			ExpiresAt: &expiresAt,
			Size:      int64(len(req.Content)),
		})
	}))
	defer server.Close()

	api, err := New(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := api.Create(context.Background(), CreateOptions{
		Content:      []byte("line 1\r\nlast  "),
		Expires:      "7d",
		JSONRequest:  true,
		JSONResponse: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.URL != server.URL+"/p/json1" || receipt.RawURL != server.URL+"/raw/json1" || receipt.Code != "json1" {
		t.Fatalf("receipt = %#v, want populated URLs and code", receipt)
	}
	if receipt.ExpiresAt == nil || !receipt.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("ExpiresAt = %v, want %v", receipt.ExpiresAt, expiresAt)
	}
}

func TestGetRawResolvesPasteURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/raw/abc123" {
			t.Fatalf("path = %q, want /raw/abc123", r.URL.Path)
		}
		if got := r.Header.Get("Accept"); got != "text/plain" {
			t.Fatalf("Accept = %q, want text/plain", got)
		}
		_, _ = io.WriteString(w, "first\r\nlast  ")
	}))
	defer server.Close()

	api, err := New(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	got, err := api.Get(context.Background(), server.URL+"/p/ABC123", GetOptions{Raw: true})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "first\r\nlast  " {
		t.Fatalf("raw body = %q, want exact response", string(got))
	}
}

func TestGetJSONResolvesRawURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/p/abc123" {
			t.Fatalf("path = %q, want /p/abc123", r.URL.Path)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Fatalf("Accept = %q, want application/json", got)
		}
		_, _ = io.WriteString(w, `{"code":"abc123","content":"hello"}`)
	}))
	defer server.Close()

	api, err := New(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	got, err := api.Get(context.Background(), server.URL+"/raw/ABC123", GetOptions{JSON: true})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"code":"abc123","content":"hello"}` {
		t.Fatalf("JSON body = %q, want exact response", string(got))
	}
}
