package server

import (
	"maps"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestPasteViewRendersGFMTextExtensions(t *testing.T) {
	rendered, err := renderMarkdown([]byte("~~removed~~\n\nVisit https://example.com or email team@example.com."))
	if err != nil {
		t.Fatalf("render markdown: %v", err)
	}

	body := string(rendered)
	for _, want := range []string{
		"<del>removed</del>",
		`href="https://example.com"`,
		`href="mailto:team@example.com"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("rendered markdown missing %q: %s", want, body)
		}
	}
}

func TestPasteViewRendersDisabledTaskCheckboxes(t *testing.T) {
	rendered, err := renderMarkdown([]byte("- [ ] pending\n- [x] done"))
	if err != nil {
		t.Fatalf("render markdown: %v", err)
	}

	doc, err := html.Parse(strings.NewReader(string(rendered)))
	if err != nil {
		t.Fatalf("parse rendered markdown: %v", err)
	}
	inputs := descendantElements(doc, "input")
	if len(inputs) != 2 {
		t.Fatalf("rendered input count = %d, want 2: %s", len(inputs), rendered)
	}

	checked := 0
	for _, input := range inputs {
		attrs := make(map[string]string, len(input.Attr))
		for _, attr := range input.Attr {
			attrs[attr.Key] = attr.Val
		}
		if attrs["type"] != "checkbox" {
			t.Errorf("input type = %q, want checkbox", attrs["type"])
		}
		if _, ok := attrs["disabled"]; !ok {
			t.Error("task checkbox is enabled")
		}
		if _, ok := attrs["checked"]; ok {
			checked++
		}
		for name := range attrs {
			if name != "type" && name != "disabled" && name != "checked" {
				t.Errorf("task checkbox has unexpected attribute %q", name)
			}
		}
	}
	if checked != 1 {
		t.Errorf("checked task checkbox count = %d, want 1", checked)
	}
}

func TestPasteViewOmitsRawInputsAndEventAttributes(t *testing.T) {
	rendered, err := renderMarkdown([]byte(`<input type="text" onfocus="alert(1)">`))
	if err != nil {
		t.Fatalf("render markdown: %v", err)
	}

	body := strings.ToLower(string(rendered))
	for _, unwanted := range []string{"<input", "onfocus"} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("rendered markdown contains %q: %s", unwanted, rendered)
		}
	}
}

func TestMarkdownHTMLPostProcessingRemovesUnsafeInputs(t *testing.T) {
	sanitized := markdownSanitizer.SanitizeBytes([]byte(
		`<input disabled="" type="text">` +
			`<input type="checkbox">` +
			`<input disabled="" id="unexpected" type="checkbox">` +
			`<input checked="" disabled="" type="checkbox" onfocus="alert(1)">`,
	))
	if !strings.Contains(string(sanitized), `id="unexpected"`) {
		t.Fatalf("test setup did not retain globally allowed attribute: %s", sanitized)
	}
	rendered, err := postProcessMarkdownHTML(sanitized)
	if err != nil {
		t.Fatalf("post-process markdown HTML: %v", err)
	}

	doc, err := html.Parse(strings.NewReader(string(rendered)))
	if err != nil {
		t.Fatalf("parse post-processed markdown: %v", err)
	}
	inputs := descendantElements(doc, "input")
	if len(inputs) != 1 {
		t.Fatalf("post-processed input count = %d, want 1: %s", len(inputs), rendered)
	}
	attrs := make(map[string]string, len(inputs[0].Attr))
	for _, attr := range inputs[0].Attr {
		attrs[attr.Key] = attr.Val
	}
	wantAttrs := map[string]string{"type": "checkbox", "disabled": "", "checked": ""}
	if !maps.Equal(attrs, wantAttrs) {
		t.Fatalf("safe task checkbox attributes = %v, want exact disabled checked checkbox attributes", attrs)
	}
}

func TestPasteViewDoesNotPermitFTPLinks(t *testing.T) {
	rendered, err := renderMarkdown([]byte("ftp://example.com/file"))
	if err != nil {
		t.Fatalf("render markdown: %v", err)
	}
	if strings.Contains(strings.ToLower(string(rendered)), `href="ftp:`) {
		t.Fatalf("rendered markdown permits FTP link: %s", rendered)
	}
}
