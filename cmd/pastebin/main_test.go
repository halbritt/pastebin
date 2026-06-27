package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestRunCreateFromFilePrintsOnlyPasteURL(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			t.Fatalf("path = %q, want /", r.URL.Path)
		}
		if got := r.URL.Query().Get("expires"); got != "1d" {
			t.Fatalf("expires = %q, want 1d", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != "from file\n" {
			t.Fatalf("body = %q, want file content", string(body))
		}
		_, _ = io.WriteString(w, server.URL+"/p/file1\n")
	}))
	defer server.Close()

	dir := t.TempDir()
	file := filepath.Join(dir, "paste.txt")
	if err := os.WriteFile(file, []byte("from file\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"--server", server.URL, "--expires", "1d", file}, nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
	}
	if got := stdout.String(); got != server.URL+"/p/file1\n" {
		t.Fatalf("stdout = %q, want only paste URL", got)
	}
}

func TestRunCreateFromFileUsesConfigServer(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			t.Fatalf("path = %q, want /", r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != "from file via config\n" {
			t.Fatalf("body = %q, want file content", string(body))
		}
		_, _ = io.WriteString(w, server.URL+"/p/config1\n")
	}))
	defer server.Close()

	dir := t.TempDir()
	file := filepath.Join(dir, "paste.txt")
	configFile := filepath.Join(dir, "config")
	if err := os.WriteFile(file, []byte("from file via config\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configFile, []byte("server="+server.URL+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PASTEBIN_URL", "")
	t.Setenv("PASTEBIN_CONFIG", configFile)

	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{file}, nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
	}
	if got := stdout.String(); got != server.URL+"/p/config1\n" {
		t.Fatalf("stdout = %q, want only paste URL", got)
	}
}

func TestRunCreateFromStdinJSON(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Fatalf("Accept = %q, want application/json", got)
		}
		if got := r.Header.Get("Content-Type"); got != "text/plain; charset=utf-8" {
			t.Fatalf("Content-Type = %q, want text/plain; charset=utf-8", got)
		}
		if got := r.URL.Query().Get("expires"); got != "" {
			t.Fatalf("expires = %q, want empty server default", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != "stdin\r\ntext" {
			t.Fatalf("body = %q, want stdin content", string(body))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"url":"`+server.URL+`/p/stdin1","raw_url":"`+server.URL+`/raw/stdin1","code":"stdin1","size":11}`)
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"--server", server.URL, "--json"}, bytes.NewBufferString("stdin\r\ntext"), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
	}
	var receipt struct {
		URL    string `json:"url"`
		RawURL string `json:"raw_url"`
		Code   string `json:"code"`
		Size   int64  `json:"size"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &receipt); err != nil {
		t.Fatalf("stdout is not JSON: %q: %v", stdout.String(), err)
	}
	if receipt.URL != server.URL+"/p/stdin1" || receipt.RawURL != server.URL+"/raw/stdin1" || receipt.Code != "stdin1" || receipt.Size != 11 {
		t.Fatalf("receipt = %#v, want server receipt", receipt)
	}
}

func TestRunGetCodePrintsRawContentExactly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/raw/abc123" {
			t.Fatalf("path = %q, want /raw/abc123", r.URL.Path)
		}
		_, _ = io.WriteString(w, "raw\r\nwithout final newline")
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"get", "--server", server.URL, "ABC123"}, nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
	}
	if got := stdout.String(); got != "raw\r\nwithout final newline" {
		t.Fatalf("stdout = %q, want exact raw content", got)
	}
}

func TestRunGetJSONPrintsServerPasteJSON(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/p/abc123" {
			t.Fatalf("path = %q, want /p/abc123", r.URL.Path)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Fatalf("Accept = %q, want application/json", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"url":"`+server.URL+`/p/abc123","raw_url":"`+server.URL+`/raw/abc123","code":"abc123","content":"raw text","size":8}`)
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"get", "--server", server.URL, "--json", "ABC123"}, nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
	}
	var body struct {
		URL     string `json:"url"`
		RawURL  string `json:"raw_url"`
		Code    string `json:"code"`
		Content string `json:"content"`
		Size    int64  `json:"size"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &body); err != nil {
		t.Fatalf("stdout is not JSON: %q: %v", stdout.String(), err)
	}
	if body.URL != server.URL+"/p/abc123" || body.RawURL != server.URL+"/raw/abc123" || body.Code != "abc123" || body.Content != "raw text" || body.Size != 8 {
		t.Fatalf("body = %+v", body)
	}
}

func TestRunGetFullPasteURLDoesNotRequireConfiguredServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/raw/abc123" {
			t.Fatalf("path = %q, want /raw/abc123", r.URL.Path)
		}
		_, _ = io.WriteString(w, "content")
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"get", server.URL + "/p/ABC123"}, nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
	}
	if got := stdout.String(); got != "content" {
		t.Fatalf("stdout = %q, want raw content", got)
	}
}

func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"version"}, nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
	}
	if got := stdout.String(); got != version+"\n" {
		t.Fatalf("stdout = %q, want version", got)
	}
}
