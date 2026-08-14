package tests

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// PASTEBIN-3 regression: the first public deployment reused the private database.
func TestPASTEBIN3DeploymentKeepsPublicDocumentsInSeparateDatabase(t *testing.T) {
	private := readEnvironmentFile(t, filepath.Join("..", "docs", "deployment", "pastebin.env.example"))
	public := readEnvironmentFile(t, filepath.Join("..", "docs", "deployment", "pastebin-public.env.example"))

	if private["PASTEBIN_DB"] == "" || public["PASTEBIN_DB"] == "" {
		t.Fatal("both deployment environments must define PASTEBIN_DB")
	}
	if private["PASTEBIN_DB"] == public["PASTEBIN_DB"] {
		t.Fatalf("private and public deployments share PASTEBIN_DB %q", private["PASTEBIN_DB"])
	}
	if private["PASTEBIN_LISTEN"] == public["PASTEBIN_LISTEN"] {
		t.Fatalf("private and public deployments share PASTEBIN_LISTEN %q", private["PASTEBIN_LISTEN"])
	}
	if private["PASTEBIN_PUBLIC_HOST"] != "" {
		t.Fatalf("private deployment exposes PASTEBIN_PUBLIC_HOST %q", private["PASTEBIN_PUBLIC_HOST"])
	}
	if public["PASTEBIN_PUBLIC_HOST"] == "" {
		t.Fatal("public deployment does not define PASTEBIN_PUBLIC_HOST")
	}
	if private["PASTEBIN_PUBLISH_TOKEN_FILE"] != "" {
		t.Fatal("private deployment must not define PASTEBIN_PUBLISH_TOKEN_FILE")
	}
	if public["PASTEBIN_PUBLISH_TOKEN_FILE"] == "" {
		t.Fatal("public deployment does not define PASTEBIN_PUBLISH_TOKEN_FILE")
	}
}

func readEnvironmentFile(t *testing.T, path string) map[string]string {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer file.Close()

	values := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			t.Fatalf("%s contains invalid environment line %q", path, line)
		}
		values[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return values
}
