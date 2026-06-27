package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
	"unicode/utf8"

	"pastebin/internal/paste"
)

const maxErrorBody = 4096

var (
	ErrMissingBaseURL = errors.New("pastebin server URL is required")
	ErrInvalidTarget  = errors.New("invalid paste target")
)

type Client struct {
	base       *url.URL
	HTTPClient *http.Client
}

type CreateOptions struct {
	Content      []byte
	Expires      string
	JSONRequest  bool
	JSONResponse bool
}

type Receipt struct {
	URL       string     `json:"url,omitempty"`
	RawURL    string     `json:"raw_url,omitempty"`
	Code      string     `json:"code,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Size      int64      `json:"size,omitempty"`
}

type GetOptions struct {
	Raw  bool
	JSON bool
}

type StatusError struct {
	StatusCode int
	Status     string
	Body       string
}

func (e *StatusError) Error() string {
	if e.Body == "" {
		return e.Status
	}
	return fmt.Sprintf("%s: %s", e.Status, e.Body)
}

func New(baseURL string) (*Client, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil, ErrMissingBaseURL
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("%w: %q", ErrMissingBaseURL, baseURL)
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	return &Client{base: parsed}, nil
}

func (c *Client) Create(ctx context.Context, opts CreateOptions) (Receipt, error) {
	if opts.Expires != "" {
		if _, err := paste.ParseAllowedTTL(opts.Expires); err != nil {
			return Receipt{}, err
		}
	}

	createURL := c.createURL()
	body, contentType, err := createBody(opts)
	if err != nil {
		return Receipt{}, err
	}
	if !opts.JSONRequest && opts.Expires != "" {
		query := createURL.Query()
		query.Set("expires", opts.Expires)
		createURL.RawQuery = query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, createURL.String(), body)
	if err != nil {
		return Receipt{}, err
	}
	req.Header.Set("Content-Type", contentType)
	if opts.JSONResponse {
		req.Header.Set("Accept", "application/json")
	} else {
		req.Header.Set("Accept", "text/plain")
	}

	data, err := c.do(req)
	if err != nil {
		return Receipt{}, err
	}
	if !opts.JSONResponse {
		return Receipt{URL: strings.TrimSpace(string(data))}, nil
	}

	var receipt Receipt
	if err := json.Unmarshal(data, &receipt); err != nil {
		return Receipt{}, err
	}
	return receipt, nil
}

func (c *Client) Get(ctx context.Context, target string, opts GetOptions) ([]byte, error) {
	raw := opts.Raw || !opts.JSON
	getURL, err := c.resolveGetURL(target, raw)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, getURL.String(), nil)
	if err != nil {
		return nil, err
	}
	if opts.JSON {
		req.Header.Set("Accept", "application/json")
	} else {
		req.Header.Set("Accept", "text/plain")
	}
	return c.do(req)
}

func (c *Client) createURL() *url.URL {
	out := *c.base
	if out.Path == "" {
		out.Path = "/"
	}
	out.RawQuery = ""
	out.Fragment = ""
	return &out
}

func (c *Client) codeURL(kind, code string) *url.URL {
	out := *c.base
	out.Path = buildPath(splitPath(out.Path), kind, code)
	out.RawQuery = ""
	out.Fragment = ""
	return &out
}

func (c *Client) resolveGetURL(target string, raw bool) (*url.URL, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, ErrInvalidTarget
	}

	if parsed, err := url.Parse(target); err == nil && parsed.IsAbs() && parsed.Host != "" {
		prefix, code, err := splitTargetPath(parsed.Path)
		if err != nil {
			return nil, err
		}
		kind := "p"
		if raw {
			kind = "raw"
		}
		out := *parsed
		out.Path = buildPath(prefix, kind, paste.NormalizeCode(code))
		out.RawQuery = ""
		out.Fragment = ""
		return &out, nil
	}

	code := paste.NormalizeCode(target)
	if code == "" || strings.ContainsAny(code, "/?#") {
		return nil, fmt.Errorf("%w: %q", ErrInvalidTarget, target)
	}
	kind := "p"
	if raw {
		kind = "raw"
	}
	return c.codeURL(kind, code), nil
}

func createBody(opts CreateOptions) (io.Reader, string, error) {
	if !opts.JSONRequest {
		return bytes.NewReader(opts.Content), "text/plain; charset=utf-8", nil
	}
	if !utf8.Valid(opts.Content) {
		return nil, "", paste.ErrInvalidUTF8
	}
	payload := struct {
		Content string `json:"content"`
		Expires string `json:"expires,omitempty"`
	}{
		Content: string(opts.Content),
		Expires: opts.Expires,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, "", err
	}
	return bytes.NewReader(data), "application/json", nil
}

func (c *Client) do(req *http.Request) ([]byte, error) {
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, &StatusError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       strings.TrimSpace(limitString(string(data), maxErrorBody)),
		}
	}
	return data, nil
}

func splitTargetPath(targetPath string) ([]string, string, error) {
	segments := splitPath(targetPath)
	if len(segments) == 0 {
		return nil, "", ErrInvalidTarget
	}
	for i, segment := range segments {
		if segment != "p" && segment != "raw" {
			continue
		}
		if i+1 >= len(segments) {
			return nil, "", ErrInvalidTarget
		}
		code, err := url.PathUnescape(segments[i+1])
		if err != nil {
			return nil, "", err
		}
		return segments[:i], code, nil
	}

	code, err := url.PathUnescape(segments[len(segments)-1])
	if err != nil {
		return nil, "", err
	}
	return segments[:len(segments)-1], code, nil
}

func splitPath(value string) []string {
	value = strings.Trim(value, "/")
	if value == "" {
		return nil
	}
	return strings.Split(value, "/")
}

func buildPath(prefix []string, kind, code string) string {
	parts := append([]string{}, prefix...)
	parts = append(parts, kind, url.PathEscape(code))
	return "/" + path.Join(parts...)
}

func limitString(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}
