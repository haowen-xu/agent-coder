package httpserver

import (
	"strings"
	"testing"
	"testing/fstest"
)

// TestNormalizeStaticPath 用于单元测试。
func TestNormalizeStaticPath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: "/index.html"},
		{name: "root", in: "/", want: "/index.html"},
		{name: "trim", in: "  /board  ", want: "/board"},
		{name: "clean", in: "/assets/../index.html", want: "/index.html"},
		{name: "nested", in: "admin/overview", want: "/admin/overview"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := normalizeStaticPath(tc.in); got != tc.want {
				t.Fatalf("normalizeStaticPath(%q)=%q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestAcceptsGzipEncodingHeader 用于单元测试。
func TestAcceptsGzipEncodingHeader(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		header string
		want   bool
	}{
		{name: "empty", header: "", want: false},
		{name: "gzip", header: "gzip", want: true},
		{name: "multiple", header: "br, gzip", want: true},
		{name: "q-zero", header: "gzip;q=0", want: false},
		{name: "q-positive", header: "gzip;q=0.2", want: true},
		{name: "wildcard", header: "br;q=1,*;q=0.5", want: true},
		{name: "identity-only", header: "identity", want: false},
		{name: "x-gzip", header: "x-gzip", want: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := acceptsGzipEncodingHeader(tc.header); got != tc.want {
				t.Fatalf("acceptsGzipEncodingHeader(%q)=%v, want %v", tc.header, got, tc.want)
			}
		})
	}
}

// TestReadPrecompressedStaticAsset 用于单元测试。
func TestReadPrecompressedStaticAsset(t *testing.T) {
	t.Parallel()

	mockFS := fstest.MapFS{
		"static/index.html.gz": {
			Data: []byte("gz-index"),
		},
		"static/assets/app.js.gz": {
			Data: []byte("gz-js"),
		},
	}

	data, contentType, ok := readPrecompressedStaticAsset(mockFS, "/index.html")
	if !ok {
		t.Fatal("expected index.html.gz to exist")
	}
	if string(data) != "gz-index" {
		t.Fatalf("unexpected index gzip data: %q", string(data))
	}
	if !strings.Contains(contentType, "text/html") {
		t.Fatalf("unexpected html content type: %q", contentType)
	}

	data, contentType, ok = readPrecompressedStaticAsset(mockFS, "/assets/app.js")
	if !ok {
		t.Fatal("expected app.js.gz to exist")
	}
	if string(data) != "gz-js" {
		t.Fatalf("unexpected js gzip data: %q", string(data))
	}
	if contentType == "" {
		t.Fatal("expected non-empty js content type")
	}

	_, _, ok = readPrecompressedStaticAsset(mockFS, "/missing.css")
	if ok {
		t.Fatal("expected missing.css.gz to be absent")
	}
}
