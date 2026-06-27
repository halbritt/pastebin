package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type acceptanceBinaries struct {
	cli    string
	server string
}

type pasteReceipt struct {
	URL       string `json:"url"`
	RawURL    string `json:"raw_url"`
	Code      string `json:"code"`
	ExpiresAt string `json:"expires_at"`
	Size      int64  `json:"size"`
}

type pastebinServer struct {
	baseURL string
}

type serverConfig struct {
	defaultTTL string
	maxTTL     string
}

func TestCLIStdinCreatePreservesRawBytesAndGetAcceptsURLAndCode(t *testing.T) {
	bins := buildAcceptanceBinariesOrSkip(t)
	server := startPastebinServer(t, bins.server, serverConfig{})

	content := []byte("alpha\tbeta  \r\nsecond line\nthird line without final newline")
	createOut := runCreateCLI(t, bins.cli, server.baseURL, nil, content)
	pasteURL := parsePlainPasteURL(t, createOut)
	code := codeFromPasteURL(t, pasteURL)

	gotByURL := runGetCLI(t, bins.cli, server.baseURL, nil, pasteURL.String())
	assertBytesEqual(t, gotByURL, content, "pastebin get <url>")

	gotByCode := runGetCLI(t, bins.cli, server.baseURL, nil, code)
	assertBytesEqual(t, gotByCode, content, "pastebin get <code>")
}

func TestCLIFileCreateAndGetAcceptsRawURL(t *testing.T) {
	bins := buildAcceptanceBinariesOrSkip(t)
	server := startPastebinServer(t, bins.server, serverConfig{})

	content := []byte("from a file\r\nwith trailing spaces  \n")
	sourceFile := filepath.Join(t.TempDir(), "paste.txt")
	if err := os.WriteFile(sourceFile, content, 0o600); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	createOut := runCreateCLI(t, bins.cli, server.baseURL, []string{sourceFile}, nil)
	code := codeFromPasteURL(t, parsePlainPasteURL(t, createOut))
	rawURL := server.baseURL + "/raw/" + code

	got := runGetCLI(t, bins.cli, server.baseURL, nil, rawURL)
	assertBytesEqual(t, got, content, "pastebin get <raw-url>")
}

func TestCLIJSONCreateReturnsReceipt(t *testing.T) {
	bins := buildAcceptanceBinariesOrSkip(t)
	server := startPastebinServer(t, bins.server, serverConfig{})

	content := []byte("json create body")
	out := runCreateCLI(t, bins.cli, server.baseURL, []string{"--json"}, content)

	var receipt pasteReceipt
	if err := json.Unmarshal(out, &receipt); err != nil {
		t.Fatalf("json create output is not a receipt: %v\noutput:\n%s", err, out)
	}
	if receipt.URL == "" || receipt.RawURL == "" || receipt.Code == "" || receipt.ExpiresAt == "" {
		t.Fatalf("json receipt missing required fields: %+v", receipt)
	}
	if receipt.Size != int64(len(content)) {
		t.Fatalf("json receipt size = %d, want %d", receipt.Size, len(content))
	}
	if got := codeFromPasteURL(t, mustParseAbsoluteURL(t, receipt.URL)); got != receipt.Code {
		t.Fatalf("json receipt code = %q, URL contains %q", receipt.Code, got)
	}

	resp, err := http.Get(receipt.RawURL)
	if err != nil {
		t.Fatalf("GET raw_url: %v", err)
	}
	defer resp.Body.Close()
	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read raw_url response: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET raw_url status = %d, want 200; body:\n%s", resp.StatusCode, got)
	}
	assertBytesEqual(t, got, content, "GET raw_url")
}

func TestCLIRejectsEmptyAndInvalidUTF8(t *testing.T) {
	bins := buildAcceptanceBinariesOrSkip(t)
	server := startPastebinServer(t, bins.server, serverConfig{})

	runCreateCLIFailure(t, bins.cli, server.baseURL, nil, nil)
	runCreateCLIFailure(t, bins.cli, server.baseURL, nil, []byte{0xff, 'x'})
}

func TestHTTPUnknownAndExpiredPastesUseDistinctStatuses(t *testing.T) {
	bins := buildAcceptanceBinariesOrSkip(t)

	server := startPastebinServer(t, bins.server, serverConfig{})
	assertHTTPStatus(t, server.baseURL+"/raw/unknownpastecode", http.StatusNotFound)

	expiringServer := startPastebinServer(t, bins.server, serverConfig{
		defaultTTL: "1ms",
		maxTTL:     "1h",
	})
	createOut := runCreateCLI(t, bins.cli, expiringServer.baseURL, nil, []byte("short lived"))
	code := codeFromPasteURL(t, parsePlainPasteURL(t, createOut))
	rawURL := expiringServer.baseURL + "/raw/" + code

	waitForHTTPStatus(t, rawURL, http.StatusGone, 5*time.Second)
}

func buildAcceptanceBinariesOrSkip(t *testing.T) acceptanceBinaries {
	t.Helper()

	requireBuildableMainPackageOrSkip(t, filepath.Join("..", "cmd", "pastebin"))
	requireBuildableMainPackageOrSkip(t, filepath.Join("..", "cmd", "pastebind"))

	binDir := t.TempDir()
	cli := filepath.Join(binDir, "pastebin")
	server := filepath.Join(binDir, "pastebind")
	buildGoBinary(t, cli, "../cmd/pastebin")
	buildGoBinary(t, server, "../cmd/pastebind")

	return acceptanceBinaries{cli: cli, server: server}
}

func requireBuildableMainPackageOrSkip(t *testing.T, dir string) {
	t.Helper()

	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skipf("%s is not present yet; skipping compiled CLI/server acceptance tests", dir)
		}
		t.Fatalf("stat %s: %v", dir, err)
	}
	if !info.IsDir() {
		t.Fatalf("%s is not a directory", dir)
	}
	goFiles, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("glob %s: %v", dir, err)
	}
	if len(goFiles) == 0 {
		t.Skipf("%s has no Go files yet; skipping compiled CLI/server acceptance tests", dir)
	}
}

func buildGoBinary(t *testing.T, output, pkg string) {
	t.Helper()

	cmd := exec.Command("go", "build", "-o", output, pkg)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build %s: %v\n%s", pkg, err, out)
	}
}

func startPastebinServer(t *testing.T, serverBin string, cfg serverConfig) pastebinServer {
	t.Helper()

	addr := reserveLocalAddress(t)
	baseURL := "http://" + addr
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, serverBin)

	dbPath := filepath.Join(t.TempDir(), "pastebin.db")
	cmd.Env = append(os.Environ(),
		"PASTEBIN_LISTEN="+addr,
		"PASTEBIN_BASE_URL="+baseURL,
		"PASTEBIN_DB="+dbPath,
		"PASTEBIN_MAX_BYTES=1048576",
	)
	if cfg.defaultTTL != "" {
		cmd.Env = append(cmd.Env, "PASTEBIN_DEFAULT_TTL="+cfg.defaultTTL)
	}
	if cfg.maxTTL != "" {
		cmd.Env = append(cmd.Env, "PASTEBIN_MAX_TTL="+cfg.maxTTL)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		cancel()
		t.Fatalf("start pastebind: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			_ = cmd.Process.Kill()
			<-done
		}
	})

	waitForHealth(t, baseURL, done, &stdout, &stderr)
	return pastebinServer{baseURL: baseURL}
}

func reserveLocalAddress(t *testing.T) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve local address: %v", err)
	}
	defer ln.Close()
	return ln.Addr().String()
}

func waitForHealth(t *testing.T, baseURL string, done <-chan error, stdout, stderr *bytes.Buffer) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-done:
			t.Fatalf("pastebind exited before becoming healthy: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
		default:
		}

		resp, err := http.Get(baseURL + "/healthz")
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("pastebind did not become healthy at %s\nstdout:\n%s\nstderr:\n%s", baseURL, stdout.String(), stderr.String())
}

func runCreateCLI(t *testing.T, cliBin, baseURL string, args []string, input []byte) []byte {
	t.Helper()

	fullArgs := append([]string{"--server", baseURL}, args...)
	out, err := runCLICommand(cliBin, baseURL, fullArgs, input)
	if err != nil {
		t.Fatalf("pastebin %s failed: %v\noutput:\n%s", strings.Join(fullArgs, " "), err, out)
	}
	return out
}

func runGetCLI(t *testing.T, cliBin, baseURL string, args []string, target string) []byte {
	t.Helper()

	fullArgs := append([]string{"get", "--server", baseURL}, args...)
	fullArgs = append(fullArgs, target)
	out, err := runCLICommand(cliBin, baseURL, fullArgs, nil)
	if err != nil {
		t.Fatalf("pastebin %s failed: %v\noutput:\n%s", strings.Join(fullArgs, " "), err, out)
	}
	return out
}

func runCreateCLIFailure(t *testing.T, cliBin, baseURL string, args []string, input []byte) []byte {
	t.Helper()

	fullArgs := append([]string{"--server", baseURL}, args...)
	out, err := runCLICommand(cliBin, baseURL, fullArgs, input)
	if err == nil {
		t.Fatalf("pastebin %s succeeded unexpectedly; output:\n%s", strings.Join(fullArgs, " "), out)
	}
	return out
}

func runCLICommand(cliBin, baseURL string, args []string, input []byte) ([]byte, error) {
	cmd := exec.Command(cliBin, args...)
	cmd.Env = append(os.Environ(), "PASTEBIN_URL="+baseURL)
	cmd.Stdin = bytes.NewReader(input)
	return cmd.CombinedOutput()
}

func parsePlainPasteURL(t *testing.T, out []byte) *url.URL {
	t.Helper()

	text := string(out)
	line := strings.TrimSpace(text)
	if line == "" || strings.Contains(line, "\n") {
		t.Fatalf("create output = %q, want exactly one paste URL line", text)
	}
	return mustParseAbsoluteURL(t, line)
}

func mustParseAbsoluteURL(t *testing.T, raw string) *url.URL {
	t.Helper()

	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse URL %q: %v", raw, err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		t.Fatalf("URL %q is not absolute", raw)
	}
	return parsed
}

func codeFromPasteURL(t *testing.T, pasteURL *url.URL) string {
	t.Helper()

	const prefix = "/p/"
	if !strings.HasPrefix(pasteURL.Path, prefix) {
		t.Fatalf("paste URL path = %q, want /p/{code}", pasteURL.Path)
	}
	code := strings.TrimPrefix(pasteURL.Path, prefix)
	if code == "" || strings.Contains(code, "/") {
		t.Fatalf("paste URL path = %q, want one paste code", pasteURL.Path)
	}
	return code
}

func assertBytesEqual(t *testing.T, got, want []byte, label string) {
	t.Helper()

	if !bytes.Equal(got, want) {
		t.Fatalf("%s returned bytes %s, want %s", label, describeBytes(got), describeBytes(want))
	}
}

func describeBytes(value []byte) string {
	return fmt.Sprintf("len=%d quoted=%q", len(value), string(value))
}

func assertHTTPStatus(t *testing.T, target string, want int) {
	t.Helper()

	resp, err := http.Get(target)
	if err != nil {
		t.Fatalf("GET %s: %v", target, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != want {
		t.Fatalf("GET %s status = %d, want %d; body:\n%s", target, resp.StatusCode, want, body)
	}
}

func waitForHTTPStatus(t *testing.T, target string, want int, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	var lastStatus int
	var lastBody []byte
	for time.Now().Before(deadline) {
		resp, err := http.Get(target)
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			lastStatus = resp.StatusCode
			lastBody = body
			if resp.StatusCode == want {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("GET %s did not return %d before timeout; last status=%d body:\n%s", target, want, lastStatus, lastBody)
}
