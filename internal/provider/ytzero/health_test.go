package ytzero

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestResolvePaths(t *testing.T) {
	t.Parallel()
	paths, err := ResolvePaths(".")
	if err != nil {
		t.Fatalf("ResolvePaths: %v", err)
	}
	if paths.Root == "" {
		t.Fatal("expected non-empty root")
	}
	if paths.ComposeFile == "" || paths.ProviderDir == "" {
		t.Fatalf("incomplete paths: %+v", paths)
	}
}

func TestHealthClientOK(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != HealthPath {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"version":"test"}`))
	}))
	defer server.Close()

	client := NewHealthClient(server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	result, err := client.CheckGET(ctx)
	if err != nil {
		t.Fatalf("CheckGET: %v", err)
	}
	if !result.OK || result.StatusCode != http.StatusOK {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.Body["version"] != "test" {
		t.Fatalf("body = %#v", result.Body)
	}
}

func TestFormatStatus(t *testing.T) {
	t.Parallel()
	out, err := FormatStatus(StatusReport{
		Root:         "/repo",
		ComposeFile:  "/repo/deploy/ytzero/compose.yaml",
		DataDir:      "/repo/data/ytzero",
		ProviderDir:  "/repo/providers/ytzero",
		ProviderHEAD: "abc",
		ProviderDesc: "2026.09.8",
		BaseURL:      DefaultBaseURL,
		ComposePS:    "NAME  STATUS",
	})
	if err != nil {
		t.Fatalf("FormatStatus: %v", err)
	}
	for _, part := range []string{"Provider: YT Zero", "2026.09.8", DefaultBaseURL} {
		if !strings.Contains(out, part) {
			t.Fatalf("missing %q in:\n%s", part, out)
		}
	}
}
