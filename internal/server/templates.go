package server

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"pastebin/internal/paste"
)

//go:embed assets/* templates/*
var embeddedFiles embed.FS

var pageTemplates = template.Must(template.ParseFS(embeddedFiles, "templates/*.html"))

type pageAssets struct {
	Style  template.CSS
	Script template.JS
}

type expiresOption struct {
	Value   string
	Label   string
	Default bool
}

type homePageData struct {
	pageAssets
	Expires []expiresOption
}

type pastePageData struct {
	pageAssets
	PasteURL     string
	RawURL       string
	RawPath      string
	RenderedHTML template.HTML
	Code         string
	CreatedAt    string
	CreatedLabel string
	ExpiresAt    string
	ExpiresLabel string
	SizeLabel    string
}

func (s *Server) renderHome(w http.ResponseWriter) {
	s.renderTemplate(w, "home.html", homePageData{
		pageAssets: assets(),
		Expires:    expiresOptions(),
	})
}

func (s *Server) renderPaste(w http.ResponseWriter, r *http.Request, found paste.Paste) {
	baseURL := s.absoluteBaseURL(r)
	renderedHTML, err := renderMarkdown(found.Content)
	if err != nil {
		http.Error(w, "paste rendering failed", http.StatusInternalServerError)
		return
	}
	s.renderTemplate(w, "paste.html", pastePageData{
		pageAssets:   assets(),
		PasteURL:     pasteViewURL(baseURL, found.Code),
		RawURL:       rawPasteURL(baseURL, found.Code),
		RawPath:      "/raw/" + found.Code,
		RenderedHTML: renderedHTML,
		Code:         found.Code,
		CreatedAt:    htmlTime(found.CreatedAt),
		CreatedLabel: displayTime(found.CreatedAt),
		ExpiresAt:    htmlTime(found.ExpiresAt),
		ExpiresLabel: displayTime(found.ExpiresAt),
		SizeLabel:    sizeLabel(found.Size, len(found.Content)),
	})
}

func (s *Server) renderTemplate(w http.ResponseWriter, name string, data any) {
	var body bytes.Buffer
	if err := pageTemplates.ExecuteTemplate(&body, name, data); err != nil {
		http.Error(w, "template rendering failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = body.WriteTo(w)
}

func assets() pageAssets {
	return pageAssets{
		Style:  template.CSS(mustReadEmbedded("assets/style.css")),
		Script: template.JS(mustReadEmbedded("assets/app.js")),
	}
}

func mustReadEmbedded(path string) string {
	content, err := embeddedFiles.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return string(content)
}

func expiresOptions() []expiresOption {
	return []expiresOption{
		{Value: "1h", Label: "1 hour"},
		{Value: "1d", Label: "1 day"},
		{Value: "7d", Label: "7 days", Default: true},
		{Value: "30d", Label: "30 days"},
	}
}

func htmlTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}

func displayTime(value time.Time) string {
	return value.UTC().Format("2006-01-02 15:04:05 UTC")
}

func sizeLabel(storedSize int64, contentBytes int) string {
	size := storedSize
	if size == 0 {
		size = int64(contentBytes)
	}
	if size == 1 {
		return "1 byte"
	}
	return fmt.Sprintf("%d bytes", size)
}
