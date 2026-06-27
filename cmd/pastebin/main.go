package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"pastebin/internal/client"
	"pastebin/internal/paste"
)

var version = "dev"

const rootUsage = `Usage:
  pastebin [--server URL] [--expires 1h|1d|7d|30d] [--json] [file]
  pastebin get [--server URL] [--raw] [--json] <url-or-code>
  pastebin version

Create a paste from a file or standard input. With no file argument, pastebin
reads from stdin until EOF.

Options:
  --server URL       Pastebin service URL for this command
  --expires TTL      Paste expiration: 1h, 1d, 7d, or 30d
  --json             Print a JSON receipt when creating a paste

Configuration:
  PASTEBIN_URL       Default service URL
  PASTEBIN_CONFIG    Config file path, default ~/.config/pastebin/config

Config file format:
  server=https://paste.example.ts.net
`

const getUsage = `Usage:
  pastebin get [--server URL] [--raw] [--json] <url-or-code>

Retrieve a paste by Paste URL, Raw Paste URL, or Paste Code. Raw paste content
is printed by default.

Options:
  --server URL       Pastebin service URL for code lookups
  --raw              Print raw paste content
  --json             Print structured paste JSON
`

const versionUsage = `Usage:
  pastebin version

Print the pastebin CLI version.
`

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) > 0 {
		switch args[0] {
		case "help", "-h", "--help":
			return runHelp(args[1:], stdout, stderr)
		case "get":
			return runGet(ctx, args[1:], stdout, stderr)
		case "version":
			if len(args) == 2 && isHelpArg(args[1]) {
				fmt.Fprint(stdout, versionUsage)
				return 0
			}
			if len(args) != 1 {
				return fail(stderr, "version does not accept arguments")
			}
			fmt.Fprintln(stdout, version)
			return 0
		}
	}
	return runCreate(ctx, args, stdin, stdout, stderr)
}

func runCreate(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags := newFlagSet("pastebin")
	server := flags.String("server", "", "pastebin service URL")
	expires := flags.String("expires", "", "paste expiration")
	jsonOut := flags.Bool("json", false, "print JSON receipt")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, rootUsage)
			return 0
		}
		return fail(stderr, "%v", err)
	}
	if flags.NArg() > 1 {
		return fail(stderr, "create accepts at most one file")
	}
	if *expires != "" {
		if _, err := paste.ParseAllowedTTL(*expires); err != nil {
			return fail(stderr, "%v", err)
		}
	}

	serverURL, err := configuredServer(*server)
	if err != nil {
		return fail(stderr, "%v", err)
	}
	api, err := client.New(serverURL)
	if err != nil {
		return fail(stderr, "%v", err)
	}

	content, err := readCreateInput(stdin, flags.Args())
	if err != nil {
		return fail(stderr, "%v", err)
	}
	receipt, err := api.Create(ctx, client.CreateOptions{
		Content:      content,
		Expires:      *expires,
		JSONResponse: *jsonOut,
	})
	if err != nil {
		return fail(stderr, "%v", err)
	}

	if *jsonOut {
		encoder := json.NewEncoder(stdout)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(receipt); err != nil {
			return fail(stderr, "%v", err)
		}
		return 0
	}
	fmt.Fprintln(stdout, receipt.URL)
	return 0
}

func runGet(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := newFlagSet("pastebin get")
	server := flags.String("server", "", "pastebin service URL")
	rawOut := flags.Bool("raw", false, "print raw paste content")
	jsonOut := flags.Bool("json", false, "print JSON paste")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprint(stdout, getUsage)
			return 0
		}
		return fail(stderr, "%v", err)
	}
	if flags.NArg() != 1 {
		return fail(stderr, "get requires exactly one paste URL or code")
	}
	if *rawOut && *jsonOut {
		return fail(stderr, "--raw and --json cannot be used together")
	}

	target := flags.Arg(0)
	serverURL, err := configuredServerForGet(*server, target)
	if err != nil {
		return fail(stderr, "%v", err)
	}
	api, err := client.New(serverURL)
	if err != nil {
		return fail(stderr, "%v", err)
	}

	if *jsonOut {
		data, err := api.Get(ctx, target, client.GetOptions{JSON: true})
		if err != nil {
			return fail(stderr, "%v", err)
		}
		if _, err := stdout.Write(data); err != nil {
			return fail(stderr, "%v", err)
		}
		if len(data) == 0 || data[len(data)-1] != '\n' {
			fmt.Fprintln(stdout)
		}
		return 0
	}

	data, err := api.Get(ctx, target, client.GetOptions{Raw: true})
	if err != nil {
		return fail(stderr, "%v", err)
	}

	if _, err := stdout.Write(data); err != nil {
		return fail(stderr, "%v", err)
	}
	return 0
}

func newFlagSet(name string) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	return flags
}

func runHelp(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stdout, rootUsage)
		return 0
	}
	if len(args) != 1 {
		return fail(stderr, "help accepts at most one topic")
	}
	switch args[0] {
	case "get":
		fmt.Fprint(stdout, getUsage)
	case "version":
		fmt.Fprint(stdout, versionUsage)
	default:
		return fail(stderr, "unknown help topic %q", args[0])
	}
	return 0
}

func isHelpArg(arg string) bool {
	return arg == "-h" || arg == "--help" || arg == "help"
}

func readCreateInput(stdin io.Reader, args []string) ([]byte, error) {
	if len(args) == 0 {
		return io.ReadAll(stdin)
	}
	return os.ReadFile(args[0])
}

func configuredServer(flagValue string) (string, error) {
	if server := strings.TrimSpace(flagValue); server != "" {
		return server, nil
	}
	if server := strings.TrimSpace(os.Getenv("PASTEBIN_URL")); server != "" {
		return server, nil
	}
	if server, err := configuredServerFromFile(); err != nil {
		return "", err
	} else if server != "" {
		return server, nil
	}
	return "", client.ErrMissingBaseURL
}

func configuredServerFromFile() (string, error) {
	path, err := configPath()
	if err != nil {
		return "", err
	}
	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return parseConfigServer(content), nil
}

func configPath() (string, error) {
	if path := strings.TrimSpace(os.Getenv("PASTEBIN_CONFIG")); path != "" {
		return path, nil
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "pastebin", "config"), nil
}

func parseConfigServer(content []byte) string {
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if key, value, ok := strings.Cut(line, "="); ok {
			if strings.TrimSpace(key) == "server" {
				return strings.TrimSpace(value)
			}
			continue
		}
		return line
	}
	return ""
}

func configuredServerForGet(flagValue, target string) (string, error) {
	if server, err := configuredServer(flagValue); err == nil {
		return server, nil
	} else if !errors.Is(err, client.ErrMissingBaseURL) {
		return "", err
	}

	parsed, err := url.Parse(strings.TrimSpace(target))
	if err == nil && parsed.IsAbs() && parsed.Scheme != "" && parsed.Host != "" {
		return parsed.Scheme + "://" + parsed.Host, nil
	}
	return "", client.ErrMissingBaseURL
}

func fail(stderr io.Writer, format string, args ...any) int {
	fmt.Fprintf(stderr, "pastebin: "+format+"\n", args...)
	return 1
}
