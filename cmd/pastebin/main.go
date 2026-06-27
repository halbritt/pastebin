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
	"strings"

	"pastebin/internal/client"
	"pastebin/internal/paste"
)

var version = "dev"

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) > 0 {
		switch args[0] {
		case "get":
			return runGet(ctx, args[1:], stdout, stderr)
		case "version":
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
	return "", client.ErrMissingBaseURL
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
