package gitlab

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/haowen-xu/agent-coder/internal/dal"
	"github.com/haowen-xu/agent-coder/internal/infra/issuetracker"
	"github.com/haowen-xu/agent-coder/internal/infra/secret"
	"github.com/haowen-xu/agent-coder/internal/xerr"
)

type Client struct {
	log    *slog.Logger
	http   *http.Client
	secret secret.Manager
}

func NewClient(log *slog.Logger, timeout time.Duration, secretManager secret.Manager) *Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Client{
		log:    log,
		http:   &http.Client{Timeout: timeout},
		secret: secretManager,
	}
}

func (c *Client) ListIssues(ctx context.Context, project db.Project, opt issuetracker.ListIssuesOptions) ([]issuetracker.Issue, error) {
	state := strings.TrimSpace(opt.State)
	if state == "" {
		state = "all"
	}
	perPage := opt.PerPage
	if perPage <= 0 || perPage > 100 {
		perPage = 100
	}
	maxPages := opt.MaxPages
	if maxPages <= 0 {
		maxPages = 20
	}

	all := make([]issuetracker.Issue, 0, perPage)
	for page := 1; page <= maxPages; page++ {
		values := url.Values{}
		values.Set("state", state)
		values.Set("per_page", strconv.Itoa(perPage))
		values.Set("page", strconv.Itoa(page))
		values.Set("order_by", "updated_at")
		values.Set("sort", "asc")
		if opt.UpdatedAfter != nil {
			values.Set("updated_after", opt.UpdatedAfter.UTC().Format(time.RFC3339))
		}

		var rows []gitLabIssue
		endpoint := c.endpoint(project, fmt.Sprintf(
			"/projects/%s/issues?%s",
			url.PathEscape(c.projectRef(project)),
			values.Encode(),
		))
		if err := c.doJSON(ctx, project, http.MethodGet, endpoint, nil, &rows); err != nil {
			return nil, err
		}
		for _, row := range rows {
			all = append(all, issuetracker.Issue{
				IID:       row.IID,
				UID:       string(row.ID),
				Title:     row.Title,
				State:     row.State,
				Labels:    row.Labels,
				WebURL:    row.WebURL,
				ClosedAt:  row.ClosedAt,
				UpdatedAt: row.UpdatedAt,
			})
		}
		if len(rows) < perPage {
			break
		}
	}
	return all, nil
}

func (c *Client) SetIssueLabels(ctx context.Context, project db.Project, issueIID int64, labels []string) error {
	endpoint := c.endpoint(project, fmt.Sprintf("/projects/%s/issues/%d", url.PathEscape(c.projectRef(project)), issueIID))
	body := map[string]any{
		"labels": strings.Join(labels, ","),
	}
	return c.doJSON(ctx, project, http.MethodPut, endpoint, body, nil)
}

func (c *Client) CreateIssueNote(ctx context.Context, project db.Project, issueIID int64, body string) error {
	endpoint := c.endpoint(project, fmt.Sprintf("/projects/%s/issues/%d/notes", url.PathEscape(c.projectRef(project)), issueIID))
	payload := map[string]any{
		"body": body,
	}
	return c.doJSON(ctx, project, http.MethodPost, endpoint, payload, nil)
}

func (c *Client) CloseIssue(ctx context.Context, project db.Project, issueIID int64) error {
	endpoint := c.endpoint(project, fmt.Sprintf("/projects/%s/issues/%d", url.PathEscape(c.projectRef(project)), issueIID))
	body := map[string]any{
		"state_event": "close",
	}
	return c.doJSON(ctx, project, http.MethodPut, endpoint, body, nil)
}

func (c *Client) EnsureMergeRequest(
	ctx context.Context,
	project db.Project,
	req issuetracker.CreateOrUpdateMRRequest,
) (*issuetracker.MergeRequest, error) {
	ref := url.PathEscape(c.projectRef(project))
	listURL := c.endpoint(project, fmt.Sprintf(
		"/projects/%s/merge_requests?state=opened&source_branch=%s&per_page=1",
		ref,
		url.QueryEscape(req.SourceBranch),
	))
	var exists []gitLabMR
	if err := c.doJSON(ctx, project, http.MethodGet, listURL, nil, &exists); err != nil {
		return nil, err
	}
	if len(exists) > 0 {
		row := exists[0]
		return &issuetracker.MergeRequest{
			IID:          row.IID,
			WebURL:       row.WebURL,
			SourceBranch: row.SourceBranch,
			TargetBranch: row.TargetBranch,
			State:        row.State,
		}, nil
	}

	createURL := c.endpoint(project, fmt.Sprintf("/projects/%s/merge_requests", ref))
	payload := map[string]any{
		"source_branch":        req.SourceBranch,
		"target_branch":        req.TargetBranch,
		"title":                req.Title,
		"description":          req.Description,
		"remove_source_branch": false,
	}
	var created gitLabMR
	if err := c.doJSON(ctx, project, http.MethodPost, createURL, payload, &created); err != nil {
		return nil, err
	}
	return &issuetracker.MergeRequest{
		IID:          created.IID,
		WebURL:       created.WebURL,
		SourceBranch: created.SourceBranch,
		TargetBranch: created.TargetBranch,
		State:        created.State,
	}, nil
}

func (c *Client) MergeMergeRequest(ctx context.Context, project db.Project, mrIID int64) error {
	endpoint := c.endpoint(project, fmt.Sprintf("/projects/%s/merge_requests/%d/merge", url.PathEscape(c.projectRef(project)), mrIID))
	raw, err := json.Marshal(map[string]any{})
	if err != nil {
		return xerr.Infra.Wrap(err, "marshal gitlab merge payload")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(raw))
	if err != nil {
		return xerr.Infra.Wrap(err, "build gitlab merge request")
	}
	req.Header.Set("Content-Type", "application/json")

	token, err := c.token(ctx, project)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("PRIVATE-TOKEN", token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return xerr.Infra.Wrap(err, "send gitlab merge request")
	}
	defer func() { _ = resp.Body.Close() }()

	respRaw, err := io.ReadAll(resp.Body)
	if err != nil {
		return xerr.Infra.Wrap(err, "read gitlab merge response")
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	if shouldNeedHumanMerge(resp.StatusCode) {
		return &issuetracker.ErrNeedHumanMerge{
			Provider:   db.ProviderGitLab,
			StatusCode: resp.StatusCode,
			Reason:     truncate(strings.TrimSpace(string(respRaw)), 512),
		}
	}

	c.log.Warn("gitlab merge api non-2xx",
		slog.String("url", endpoint),
		slog.Int("status", resp.StatusCode),
		slog.String("body", truncate(string(respRaw), 512)),
	)
	return xerr.Infra.New("gitlab api PUT %s failed with status=%d", endpoint, resp.StatusCode)
}

func (c *Client) endpoint(project db.Project, p string) string {
	base := strings.TrimRight(strings.TrimSpace(project.ProviderURL), "/")
	return base + p
}

func (c *Client) projectRef(project db.Project) string {
	if project.IssueProjectID != nil && strings.TrimSpace(*project.IssueProjectID) != "" {
		return strings.TrimSpace(*project.IssueProjectID)
	}
	return strings.TrimSpace(project.ProjectSlug)
}

func (c *Client) token(ctx context.Context, project db.Project) (string, error) {
	if project.ProjectToken != nil {
		if token := strings.TrimSpace(*project.ProjectToken); token != "" {
			return token, nil
		}
	}

	ref := strings.TrimSpace(project.CredentialRef)
	if ref == "" {
		return "", nil
	}
	if c.secret == nil {
		return "", xerr.Infra.New("secret manager is not configured")
	}
	token, err := c.secret.Get(ctx, ref)
	if err != nil {
		return "", xerr.Infra.Wrap(err, "get issue tracker token")
	}
	return strings.TrimSpace(token), nil
}

func (c *Client) doJSON(
	ctx context.Context,
	project db.Project,
	method string,
	url string,
	payload any,
	out any,
) error {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return xerr.Infra.Wrap(err, "marshal gitlab payload")
		}
		body = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return xerr.Infra.Wrap(err, "build gitlab request")
	}
	req.Header.Set("Content-Type", "application/json")

	token, err := c.token(ctx, project)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("PRIVATE-TOKEN", token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return xerr.Infra.Wrap(err, "send gitlab request")
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return xerr.Infra.Wrap(err, "read gitlab response")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.log.Warn("gitlab api non-2xx",
			slog.String("method", method),
			slog.String("url", url),
			slog.Int("status", resp.StatusCode),
			slog.String("body", truncate(string(raw), 512)),
		)
		return xerr.Infra.New("gitlab api %s %s failed with status=%d", method, url, resp.StatusCode)
	}

	if out != nil {
		if len(raw) == 0 {
			return nil
		}
		if err := json.Unmarshal(raw, out); err != nil {
			return xerr.Infra.Wrap(err, "unmarshal gitlab response")
		}
	}
	return nil
}

func truncate(in string, max int) string {
	if len(in) <= max {
		return in
	}
	return in[:max]
}

func shouldNeedHumanMerge(status int) bool {
	switch status {
	case http.StatusMethodNotAllowed, http.StatusNotAcceptable, http.StatusConflict, http.StatusUnprocessableEntity:
		return true
	default:
		return false
	}
}
