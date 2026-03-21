package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	db "github.com/haowen-xu/agent-coder/internal/dal"
	repocommon "github.com/haowen-xu/agent-coder/internal/infra/repo/common"
)

// TestGitLabIDUnmarshalJSON covers string/number/null/invalid payload branches.
func TestGitLabIDUnmarshalJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		payload string
		want    gitLabID
		wantErr bool
	}{
		{name: "string", payload: `"123"`, want: gitLabID("123")},
		{name: "number", payload: `456`, want: gitLabID("456")},
		{name: "null", payload: `null`, want: gitLabID("")},
		{name: "invalid", payload: `true`, wantErr: true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var got gitLabID
			err := json.Unmarshal([]byte(tc.payload), &got)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for payload %s", tc.payload)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected unmarshal error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("id mismatch: got=%q want=%q", got, tc.want)
			}
		})
	}
}

// TestClientSetIssueLabelsAndCloseIssue verifies request payloads.
func TestClientSetIssueLabelsAndCloseIssue(t *testing.T) {
	t.Parallel()

	projectID := "42"
	project := db.Project{IssueProjectID: &projectID}

	var setLabelsBody string
	var closeBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/projects/42/issues/9":
			raw, _ := io.ReadAll(r.Body)
			if strings.Contains(string(raw), `"state_event"`) {
				closeBody = string(raw)
			} else {
				setLabelsBody = string(raw)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()
	project.ProviderURL = server.URL

	client := NewClient(slog.New(slog.NewTextHandler(io.Discard, nil)), 5*time.Second, nil)
	if err := client.SetIssueLabels(context.Background(), project, 9, []string{"Agent Ready", "In Progress"}); err != nil {
		t.Fatalf("SetIssueLabels failed: %v", err)
	}
	if err := client.CloseIssue(context.Background(), project, 9); err != nil {
		t.Fatalf("CloseIssue failed: %v", err)
	}
	if !strings.Contains(setLabelsBody, `"labels":"Agent Ready,In Progress"`) {
		t.Fatalf("set labels payload mismatch: %s", setLabelsBody)
	}
	if !strings.Contains(closeBody, `"state_event":"close"`) {
		t.Fatalf("close issue payload mismatch: %s", closeBody)
	}
}

// TestClientEnsureMergeRequest verifies existing-MR and create-MR branches.
func TestClientEnsureMergeRequest(t *testing.T) {
	t.Parallel()

	projectID := "42"
	baseProject := db.Project{IssueProjectID: &projectID}

	t.Run("existing", func(t *testing.T) {
		var postCalled bool
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/projects/42/merge_requests":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`[{"iid":11,"web_url":"https://example.com/mr/11","source_branch":"agent/11","target_branch":"main","state":"opened"}]`))
			case r.Method == http.MethodPost && r.URL.Path == "/projects/42/merge_requests":
				postCalled = true
				w.WriteHeader(http.StatusInternalServerError)
			default:
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
			}
		}))
		defer server.Close()

		project := baseProject
		project.ProviderURL = server.URL
		client := NewClient(slog.New(slog.NewTextHandler(io.Discard, nil)), 5*time.Second, nil)
		mr, err := client.EnsureMergeRequest(context.Background(), project, repocommon.CreateOrUpdateMRRequest{
			SourceBranch: "agent/11",
			TargetBranch: "main",
			Title:        "title",
			Description:  "desc",
		})
		if err != nil {
			t.Fatalf("EnsureMergeRequest existing failed: %v", err)
		}
		if mr == nil || mr.IID != 11 || mr.WebURL != "https://example.com/mr/11" {
			t.Fatalf("unexpected existing MR: %#v", mr)
		}
		if postCalled {
			t.Fatalf("should not create MR when existing MR is found")
		}
	})

	t.Run("create", func(t *testing.T) {
		var postBody string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/projects/42/merge_requests":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`[]`))
			case r.Method == http.MethodPost && r.URL.Path == "/projects/42/merge_requests":
				raw, _ := io.ReadAll(r.Body)
				postBody = string(raw)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"iid":12,"web_url":"https://example.com/mr/12","source_branch":"agent/12","target_branch":"main","state":"opened"}`))
			default:
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
			}
		}))
		defer server.Close()

		project := baseProject
		project.ProviderURL = server.URL
		client := NewClient(slog.New(slog.NewTextHandler(io.Discard, nil)), 5*time.Second, nil)
		mr, err := client.EnsureMergeRequest(context.Background(), project, repocommon.CreateOrUpdateMRRequest{
			SourceBranch: "agent/12",
			TargetBranch: "main",
			Title:        "hello",
			Description:  "desc",
		})
		if err != nil {
			t.Fatalf("EnsureMergeRequest create failed: %v", err)
		}
		if mr == nil || mr.IID != 12 || mr.State != "opened" {
			t.Fatalf("unexpected created MR: %#v", mr)
		}
		if !strings.Contains(postBody, `"source_branch":"agent/12"`) || !strings.Contains(postBody, `"remove_source_branch":false`) {
			t.Fatalf("create MR payload mismatch: %s", postBody)
		}
	})
}

// TestClientMergeMergeRequest covers success, need-human and generic non-2xx branches.
func TestClientMergeMergeRequest(t *testing.T) {
	t.Parallel()

	projectID := "42"
	token := "project-token"
	project := db.Project{
		IssueProjectID: &projectID,
		ProjectToken:   &token,
	}

	t.Run("success", func(t *testing.T) {
		var gotToken string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut || r.URL.Path != "/projects/42/merge_requests/8/merge" {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
			}
			gotToken = r.Header.Get("PRIVATE-TOKEN")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		}))
		defer server.Close()
		p := project
		p.ProviderURL = server.URL

		client := NewClient(slog.New(slog.NewTextHandler(io.Discard, nil)), 5*time.Second, nil)
		if err := client.MergeMergeRequest(context.Background(), p, 8); err != nil {
			t.Fatalf("MergeMergeRequest success failed: %v", err)
		}
		if gotToken != token {
			t.Fatalf("expected PRIVATE-TOKEN header %q, got %q", token, gotToken)
		}
	})

	t.Run("need_human", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut || r.URL.Path != "/projects/42/merge_requests/8/merge" {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
			}
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(strings.Repeat("x", 700)))
		}))
		defer server.Close()
		p := project
		p.ProviderURL = server.URL

		client := NewClient(slog.New(slog.NewTextHandler(io.Discard, nil)), 5*time.Second, nil)
		err := client.MergeMergeRequest(context.Background(), p, 8)
		if err == nil {
			t.Fatalf("expected need-human error")
		}
		if !repocommon.IsNeedHumanMerge(err) {
			t.Fatalf("expected ErrNeedHumanMerge, got: %v", err)
		}
		var needHuman *repocommon.ErrNeedHumanMerge
		if !errors.As(err, &needHuman) {
			t.Fatalf("expected ErrNeedHumanMerge type, got: %T", err)
		}
		if len(needHuman.Reason) != 512 {
			t.Fatalf("expected truncated reason length 512, got %d", len(needHuman.Reason))
		}
	})

	t.Run("non_2xx", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut || r.URL.Path != "/projects/42/merge_requests/8/merge" {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
			}
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`internal error`))
		}))
		defer server.Close()
		p := project
		p.ProviderURL = server.URL

		client := NewClient(slog.New(slog.NewTextHandler(io.Discard, nil)), 5*time.Second, nil)
		err := client.MergeMergeRequest(context.Background(), p, 8)
		if err == nil {
			t.Fatalf("expected non-2xx error")
		}
		if repocommon.IsNeedHumanMerge(err) {
			t.Fatalf("500 should not be need-human error: %v", err)
		}
	})
}

// TestClientTokenProjectRefAndHelpers validates utility branches.
func TestClientTokenProjectRefAndHelpers(t *testing.T) {
	t.Parallel()

	projectID := "42"
	slug := "group/project"
	projectToken := " project-token "
	client := NewClient(slog.New(slog.NewTextHandler(io.Discard, nil)), 0, &fakeGitLabSecret{values: map[string]string{"cred": "secret-token"}})

	project := db.Project{
		IssueProjectID: &projectID,
		ProjectSlug:    slug,
		ProjectToken:   &projectToken,
		CredentialRef:  "cred",
	}
	if got := client.projectRef(project); got != "42" {
		t.Fatalf("projectRef should prefer IssueProjectID, got %q", got)
	}
	tok, err := client.token(context.Background(), project)
	if err != nil || tok != "project-token" {
		t.Fatalf("token should prefer ProjectToken, token=%q err=%v", tok, err)
	}

	project.IssueProjectID = nil
	project.ProjectToken = nil
	if got := client.projectRef(project); got != slug {
		t.Fatalf("projectRef should fallback to slug, got %q", got)
	}
	tok, err = client.token(context.Background(), project)
	if err != nil || tok != "secret-token" {
		t.Fatalf("token from secret mismatch, token=%q err=%v", tok, err)
	}

	_, err = NewClient(slog.New(slog.NewTextHandler(io.Discard, nil)), 5*time.Second, nil).token(context.Background(), db.Project{CredentialRef: "cred"})
	if err == nil {
		t.Fatalf("expected error when credential_ref configured without secret manager")
	}

	if got := truncate("abcdef", 3); got != "abc" {
		t.Fatalf("truncate mismatch: %q", got)
	}
	if got := truncate("abc", 10); got != "abc" {
		t.Fatalf("truncate should keep short strings: %q", got)
	}

	if !shouldNeedHumanMerge(http.StatusConflict) || !shouldNeedHumanMerge(http.StatusMethodNotAllowed) {
		t.Fatalf("expected conflict/method-not-allowed to require human merge")
	}
	if shouldNeedHumanMerge(http.StatusInternalServerError) {
		t.Fatalf("500 should not be treated as need-human merge")
	}
}

// TestClientDoJSON_ErrorPath verifies JSON decode failure branch.
func TestClientDoJSON_ErrorPath(t *testing.T) {
	t.Parallel()

	projectID := "42"
	project := db.Project{IssueProjectID: &projectID}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/projects/42/issues" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()
	project.ProviderURL = server.URL

	client := NewClient(slog.New(slog.NewTextHandler(io.Discard, nil)), 5*time.Second, nil)
	var out map[string]any
	err := client.doJSON(context.Background(), project, http.MethodGet, client.endpoint(project, "/projects/42/issues"), nil, &out)
	if err == nil {
		t.Fatalf("expected unmarshal error for invalid JSON")
	}
}

type fakeGitLabSecret struct {
	values map[string]string
}

func (f *fakeGitLabSecret) Get(_ context.Context, ref string) (string, error) {
	v, ok := f.values[ref]
	if !ok {
		return "", errors.New("missing secret")
	}
	return v, nil
}
