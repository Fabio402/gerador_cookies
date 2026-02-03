package scraper

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestPickAkamaiScriptURLSbSd(t *testing.T) {
	html := `
	<html>
		<head>
			<script src="/static/app.js"></script>
			<script src="/path/one?v=111"></script>
			<script src="/lL_ES0/F5/yv/0MJy/REUaFZxlhjikc/bza?v=658b2841"></script>
		</head>
	</html>`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("failed to parse html: %v", err)
	}

	s := &Scraper{}
	got := s.pickAkamaiScriptURL(doc, true)
	want := "/lL_ES0/F5/yv/0MJy/REUaFZxlhjikc/bza?v=658b2841"
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestPickAkamaiScriptURLSensorMode(t *testing.T) {
	html := `
	<html>
		<head>
			<script src="/foo/bar/keep"></script>
			<script src="/foo/bar/ignore?v=123"></script>
			<script src="/foo/bar/last" ></script>
		</head>
	</html>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("failed to parse html: %v", err)
	}

	s := &Scraper{}
	got := s.pickAkamaiScriptURL(doc, false)
	if got != "/foo/bar/last" {
		t.Fatalf("expected /foo/bar/last, got %s", got)
	}
}

func TestResolveRedirectURL(t *testing.T) {
	next, err := resolveRedirectURL("https://example.com/start/path", "/target/page")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if next != "https://example.com/target/page" {
		t.Fatalf("unexpected next url: %s", next)
	}
}

func TestNormalizeStartURL(t *testing.T) {
	if got := normalizeStartURL("", "example.com"); got != "https://example.com" {
		t.Fatalf("expected https://example.com, got %s", got)
	}
	if got := normalizeStartURL("/foo", "shop.com"); got != "https://shop.com/foo" {
		t.Fatalf("expected https://shop.com/foo, got %s", got)
	}
	if got := normalizeStartURL("http://custom.com", "shop.com"); got != "http://custom.com" {
		t.Fatalf("expected http://custom.com, got %s", got)
	}
}
