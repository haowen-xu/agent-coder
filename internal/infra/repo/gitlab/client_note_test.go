package gitlab

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	db "github.com/haowen-xu/agent-coder/internal/dal"
)

// TestClient_UpsertIssueNote_UpdateExistingByMarker 用于单元测试。
func TestClient_UpsertIssueNote_UpdateExistingByMarker(t *testing.T) {
	t.Parallel()

	var gotPUT bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/projects/42/issues/9/notes"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":77,"body":"<!-- marker --> old"}]`))
			return
		case r.Method == http.MethodPut && r.URL.Path == "/projects/42/issues/9/notes/77":
			gotPUT = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
			return
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	projectID := "42"
	project := db.Project{
		ProviderURL:    server.URL,
		IssueProjectID: &projectID,
	}
	client := NewClient(slog.New(slog.NewTextHandler(io.Discard, nil)), 10*time.Second, nil)

	if err := client.UpsertIssueNote(context.Background(), project, 9, "<!-- marker -->", "<!-- marker -->\nnew body"); err != nil {
		t.Fatalf("UpsertIssueNote failed: %v", err)
	}
	if !gotPUT {
		t.Fatalf("expected note update via PUT")
	}
}

// TestClient_UpsertIssueNote_CreateWhenMarkerNotFound 用于单元测试。
func TestClient_UpsertIssueNote_CreateWhenMarkerNotFound(t *testing.T) {
	t.Parallel()

	var gotPOST bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/projects/42/issues/9/notes"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
			return
		case r.Method == http.MethodPost && r.URL.Path == "/projects/42/issues/9/notes":
			gotPOST = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
			return
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	projectID := "42"
	project := db.Project{
		ProviderURL:    server.URL,
		IssueProjectID: &projectID,
	}
	client := NewClient(slog.New(slog.NewTextHandler(io.Discard, nil)), 10*time.Second, nil)

	if err := client.UpsertIssueNote(context.Background(), project, 9, "<!-- marker -->", "<!-- marker -->\nnew body"); err != nil {
		t.Fatalf("UpsertIssueNote failed: %v", err)
	}
	if !gotPOST {
		t.Fatalf("expected note create via POST")
	}
}
