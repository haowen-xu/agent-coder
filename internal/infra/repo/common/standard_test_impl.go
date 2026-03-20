//go:build e2e

package common

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	db "github.com/haowen-xu/agent-coder/internal/dal"
)

// issueTrackerCommonRunner 封装标准 e2e 测试运行时上下文。
type issueTrackerCommonRunner struct {
	t       *testing.T       // t 是当前测试句柄。
	ctx     context.Context  // ctx 是标准测试使用的上下文。
	helper  *gitLabE2EHelper // helper 是 GitLab API 辅助调用器。
	project db.Project       // project 是待测仓库项目配置。

	issueIID      int64  // issueIID 是临时 issue 的 IID。
	defaultBranch string // defaultBranch 是项目默认分支。
	sourceBranch  string // sourceBranch 是测试临时源分支。
	mrIID         int64  // mrIID 是测试过程中生成的 MR IID。

	marker      string    // marker 是本轮测试唯一标记。
	noteNeedle  string    // noteNeedle 用于断言评论是否写入。
	testLabel   string    // testLabel 是测试标签名。
	createdTime time.Time // createdTime 是测试数据创建时间。
}

// RunStandardRepoTests 运行仓库平台标准 e2e 套件。
// 该函数由各平台实现复用，用于验证统一接口语义（Issue/MR/Merge）。
func RunStandardRepoTests(t *testing.T, it Client) {
	t.Helper()

	cfg := loadGitLabE2EConfig(t)
	if !cfg.Ready() {
		t.Skip("skip e2e repo tests: missing env GITLAB_TESTBED_URL/GITLAB_TESTBED_PRJ_ID/GITLAB_TESTBED_PRJ_TOKEN")
		return
	}

	ctx := context.Background()
	helper := &gitLabE2EHelper{
		http: &http.Client{Timeout: 30 * time.Second},
		cfg:  cfg,
	}

	defaultBranch, err := helper.getDefaultBranch(ctx)
	if err != nil {
		t.Fatalf("get default branch: %v", err)
	}
	if strings.TrimSpace(defaultBranch) == "" {
		defaultBranch = "main"
	}

	now := time.Now().UnixNano()
	marker := fmt.Sprintf("agent-coder-e2e-%d", now)
	title := fmt.Sprintf("[E2E] %s", marker)
	desc := fmt.Sprintf("temporary issue for repo e2e (%s)", marker)
	issueIID, err := helper.createIssue(ctx, title, desc)
	if err != nil {
		t.Fatalf("create temporary issue: %v", err)
	}

	testLabel := fmt.Sprintf("e2e-%d", now)
	noteNeedle := fmt.Sprintf("todo-marker-%d", now)
	sourceBranch := fmt.Sprintf("agent-coder/e2e-%d", now)

	projectID := cfg.ProjectID
	token := cfg.Token
	runner := &issueTrackerCommonRunner{
		t:      t,
		ctx:    ctx,
		helper: helper,
		project: db.Project{
			Provider:       db.ProviderGitLab,
			ProviderURL:    cfg.APIBase,
			ProjectSlug:    cfg.ProjectSlug,
			IssueProjectID: &projectID,
			ProjectToken:   &token,
			DefaultBranch:  defaultBranch,
		},
		issueIID:      issueIID,
		defaultBranch: defaultBranch,
		sourceBranch:  sourceBranch,
		marker:        marker,
		noteNeedle:    noteNeedle,
		testLabel:     testLabel,
		createdTime:   time.Now().UTC(),
	}

	t.Cleanup(func() {
		if runner.sourceBranch != "" {
			if err := helper.deleteBranch(ctx, runner.sourceBranch); err != nil && !isGitLabNotFound(err) {
				t.Errorf("cleanup delete branch %q: %v", runner.sourceBranch, err)
			}
		}
		if runner.issueIID > 0 {
			if err := helper.deleteIssue(ctx, runner.issueIID); err != nil {
				t.Errorf("cleanup delete issue iid=%d: %v", runner.issueIID, err)
			}
		}
	})

	runner.runListIssuesTest(it)
	runner.runSetIssueLabelsTest(it)
	runner.runCreateIssueNoteTest(it)
	runner.runEnsureMergeRequestTest(it)
	runner.runMergeMergeRequestTest(it)
	runner.runCloseIssueTest(it)
}

// runListIssuesTest 验证 ListIssues 能在增量窗口内拉到测试 issue。
func (r *issueTrackerCommonRunner) runListIssuesTest(it Client) {
	r.t.Helper()

	updatedAfter := r.createdTime.Add(-2 * time.Minute)
	err := waitFor(r.ctx, 20*time.Second, 2*time.Second, func() (bool, error) {
		issues, err := it.ListIssues(r.ctx, r.project, ListIssuesOptions{
			State:        "all",
			UpdatedAfter: &updatedAfter,
			PerPage:      50,
			MaxPages:     10,
		})
		if err != nil {
			return false, err
		}
		for _, row := range issues {
			if row.IID == r.issueIID {
				return true, nil
			}
		}
		return false, nil
	})
	if err != nil {
		r.t.Fatalf("runListIssuesTest failed: %v", err)
	}
}

// runSetIssueLabelsTest 验证标签写入与远端可见性。
func (r *issueTrackerCommonRunner) runSetIssueLabelsTest(it Client) {
	r.t.Helper()

	labels := []string{"Agent Ready", r.testLabel}
	if err := it.SetIssueLabels(r.ctx, r.project, r.issueIID, labels); err != nil {
		r.t.Fatalf("SetIssueLabels failed: %v", err)
	}

	err := waitFor(r.ctx, 20*time.Second, 2*time.Second, func() (bool, error) {
		row, err := r.helper.getIssue(r.ctx, r.issueIID)
		if err != nil {
			return false, err
		}
		if row == nil {
			return false, fmt.Errorf("issue %d not found", r.issueIID)
		}
		return containsLabel(row.Labels, labels[0]) && containsLabel(row.Labels, labels[1]), nil
	})
	if err != nil {
		r.t.Fatalf("runSetIssueLabelsTest failed: %v", err)
	}
}

// runCreateIssueNoteTest 验证评论创建能力。
func (r *issueTrackerCommonRunner) runCreateIssueNoteTest(it Client) {
	r.t.Helper()

	body := fmt.Sprintf("### AgentCoder E2E Todo %s\n- [ ] item-1\n- [x] item-2\n", r.noteNeedle)
	if err := it.CreateIssueNote(r.ctx, r.project, r.issueIID, body); err != nil {
		r.t.Fatalf("CreateIssueNote failed: %v", err)
	}

	err := waitFor(r.ctx, 20*time.Second, 2*time.Second, func() (bool, error) {
		notes, err := r.helper.listIssueNotes(r.ctx, r.issueIID)
		if err != nil {
			return false, err
		}
		for _, note := range notes {
			if strings.Contains(note.Body, r.noteNeedle) {
				return true, nil
			}
		}
		return false, nil
	})
	if err != nil {
		r.t.Fatalf("runCreateIssueNoteTest failed: %v", err)
	}
}

// runEnsureMergeRequestTest 验证 MR 创建与幂等复用行为。
func (r *issueTrackerCommonRunner) runEnsureMergeRequestTest(it Client) {
	r.t.Helper()

	if err := r.helper.createBranch(r.ctx, r.sourceBranch, r.defaultBranch); err != nil {
		r.t.Fatalf("create branch failed: %v", err)
	}
	filePath := fmt.Sprintf("agent-coder-e2e/%s.txt", r.marker)
	if err := r.helper.createFile(
		r.ctx,
		r.sourceBranch,
		filePath,
		fmt.Sprintf("e2e marker %s\n", r.marker),
		fmt.Sprintf("test: create e2e marker file (%s)", r.marker),
	); err != nil {
		r.t.Fatalf("create file failed: %v", err)
	}

	req := CreateOrUpdateMRRequest{
		SourceBranch: r.sourceBranch,
		TargetBranch: r.defaultBranch,
		Title:        fmt.Sprintf("[E2E] MR %s", r.marker),
		Description:  "temporary MR created by repo e2e",
	}
	mr, err := it.EnsureMergeRequest(r.ctx, r.project, req)
	if err != nil {
		r.t.Fatalf("EnsureMergeRequest create failed: %v", err)
	}
	if mr == nil || mr.IID <= 0 {
		r.t.Fatalf("EnsureMergeRequest create returned invalid MR: %#v", mr)
	}
	r.mrIID = mr.IID

	mr2, err := it.EnsureMergeRequest(r.ctx, r.project, req)
	if err != nil {
		r.t.Fatalf("EnsureMergeRequest idempotent failed: %v", err)
	}
	if mr2 == nil || mr2.IID != r.mrIID {
		r.t.Fatalf("EnsureMergeRequest idempotent mismatch: first=%d second=%v", r.mrIID, mr2)
	}
}

// runMergeMergeRequestTest 验证自动合并路径或 need_human_merge 语义。
func (r *issueTrackerCommonRunner) runMergeMergeRequestTest(it Client) {
	r.t.Helper()

	if r.mrIID <= 0 {
		r.t.Fatalf("runMergeMergeRequestTest invalid mrIID=%d", r.mrIID)
	}
	if err := it.MergeMergeRequest(r.ctx, r.project, r.mrIID); err != nil {
		if IsNeedHumanMerge(err) {
			r.t.Logf("MergeMergeRequest returned need_human_merge, treat as covered: %v", err)
			return
		}
		r.t.Fatalf("MergeMergeRequest failed: %v", err)
	}

	err := waitFor(r.ctx, 30*time.Second, 2*time.Second, func() (bool, error) {
		row, err := r.helper.getMergeRequest(r.ctx, r.mrIID)
		if err != nil {
			return false, err
		}
		if row == nil {
			return false, fmt.Errorf("mr %d not found", r.mrIID)
		}
		return strings.EqualFold(strings.TrimSpace(row.State), "merged"), nil
	})
	if err != nil {
		r.t.Fatalf("runMergeMergeRequestTest failed: %v", err)
	}
}

// runCloseIssueTest 验证 issue 关闭能力。
func (r *issueTrackerCommonRunner) runCloseIssueTest(it Client) {
	r.t.Helper()

	if err := it.CloseIssue(r.ctx, r.project, r.issueIID); err != nil {
		r.t.Fatalf("CloseIssue failed: %v", err)
	}

	err := waitFor(r.ctx, 20*time.Second, 2*time.Second, func() (bool, error) {
		row, err := r.helper.getIssue(r.ctx, r.issueIID)
		if err != nil {
			return false, err
		}
		if row == nil {
			return false, fmt.Errorf("issue %d not found", r.issueIID)
		}
		return strings.EqualFold(strings.TrimSpace(row.State), "closed"), nil
	})
	if err != nil {
		r.t.Fatalf("runCloseIssueTest failed: %v", err)
	}
}

// gitLabE2EConfig 描述标准 e2e 用例所需的 GitLab 连接参数。
type gitLabE2EConfig struct {
	ProjectURL  string // ProjectURL 是测试仓库页面地址。
	ProjectID   string // ProjectID 是 GitLab 项目 ID。
	Token       string // Token 是项目访问令牌。
	ProjectSlug string // ProjectSlug 是 group/repo 形式的仓库标识。
	APIBase     string // APIBase 是 GitLab API 根地址。
}

// Ready 判断 e2e 所需配置是否完整。
func (c gitLabE2EConfig) Ready() bool {
	return strings.TrimSpace(c.ProjectURL) != "" &&
		strings.TrimSpace(c.ProjectID) != "" &&
		strings.TrimSpace(c.Token) != "" &&
		strings.TrimSpace(c.ProjectSlug) != "" &&
		strings.TrimSpace(c.APIBase) != ""
}

// loadGitLabE2EConfig 从环境变量和项目 URL 解析 e2e 配置。
func loadGitLabE2EConfig(t *testing.T) gitLabE2EConfig {
	t.Helper()
	loadDotEnvIfExists(t)

	cfg := gitLabE2EConfig{
		ProjectURL: strings.TrimSpace(os.Getenv("GITLAB_TESTBED_URL")),
		ProjectID:  strings.TrimSpace(os.Getenv("GITLAB_TESTBED_PRJ_ID")),
		Token:      strings.TrimSpace(os.Getenv("GITLAB_TESTBED_PRJ_TOKEN")),
	}
	if strings.TrimSpace(cfg.ProjectURL) == "" {
		return cfg
	}

	u, err := url.Parse(cfg.ProjectURL)
	if err != nil {
		t.Fatalf("invalid GITLAB_TESTBED_URL: %v", err)
	}
	cfg.ProjectSlug = strings.Trim(strings.TrimPrefix(u.Path, "/"), "/")
	if cfg.ProjectSlug == "" {
		t.Fatalf("invalid GITLAB_TESTBED_URL path, missing project slug: %q", cfg.ProjectURL)
	}
	cfg.APIBase = strings.TrimRight((&url.URL{
		Scheme: u.Scheme,
		Host:   u.Host,
		Path:   "/api/v4",
	}).String(), "/")
	return cfg
}

// loadDotEnvIfExists 在本地存在 .env 时加载为环境变量（CI 无 .env 时自动跳过）。
func loadDotEnvIfExists(t *testing.T) {
	t.Helper()

	path, ok, err := findFileUpwards(".env")
	if err != nil || !ok {
		return
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read .env: %v", err)
	}
	lines := strings.Split(string(raw), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		if key == "" {
			continue
		}
		val = strings.Trim(val, `"'`)
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
}

// findFileUpwards 从当前目录向上查找指定文件。
func findFileUpwards(name string) (string, bool, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", false, err
	}
	cur := wd
	for i := 0; i < 8; i++ {
		path := filepath.Join(cur, name)
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			return path, true, nil
		}
		next := filepath.Dir(cur)
		if next == cur {
			break
		}
		cur = next
	}
	return "", false, nil
}

// gitLabE2EHelper 封装标准测试中对 GitLab API 的直接访问。
type gitLabE2EHelper struct {
	http *http.Client    // http 是 e2e 请求客户端。
	cfg  gitLabE2EConfig // cfg 是 e2e 测试配置快照。
}

// gitLabIssueDTO 是 GitLab issue API 的最小响应结构。
type gitLabIssueDTO struct {
	IID    int64    `json:"iid"`    // IID 是 issue 编号。
	State  string   `json:"state"`  // State 是 issue 状态。
	Labels []string `json:"labels"` // Labels 是 issue 标签列表。
}

// gitLabNoteDTO 是 GitLab issue note API 的最小响应结构。
type gitLabNoteDTO struct {
	ID   int64  `json:"id"`   // ID 是 note 编号。
	Body string `json:"body"` // Body 是评论正文。
}

// gitLabMRDTO 是 GitLab MR API 的最小响应结构。
type gitLabMRDTO struct {
	IID   int64  `json:"iid"`   // IID 是 MR 编号。
	State string `json:"state"` // State 是 MR 状态。
}

// gitLabProjectDTO 是 GitLab project API 的最小响应结构。
type gitLabProjectDTO struct {
	DefaultBranch string `json:"default_branch"` // DefaultBranch 是项目默认分支。
}

// getDefaultBranch 查询项目默认分支。
func (h *gitLabE2EHelper) getDefaultBranch(ctx context.Context) (string, error) {
	var row gitLabProjectDTO
	if err := h.doJSON(ctx, http.MethodGet, fmt.Sprintf("/projects/%s", url.PathEscape(h.cfg.ProjectID)), nil, &row); err != nil {
		return "", err
	}
	return strings.TrimSpace(row.DefaultBranch), nil
}

// createIssue 在测试项目中创建临时 issue。
func (h *gitLabE2EHelper) createIssue(ctx context.Context, title string, desc string) (int64, error) {
	payload := map[string]any{
		"title":       title,
		"description": desc,
	}
	var row gitLabIssueDTO
	if err := h.doJSON(ctx, http.MethodPost, fmt.Sprintf("/projects/%s/issues", url.PathEscape(h.cfg.ProjectID)), payload, &row); err != nil {
		return 0, err
	}
	return row.IID, nil
}

// deleteIssue 删除临时 issue，供清理阶段调用。
func (h *gitLabE2EHelper) deleteIssue(ctx context.Context, issueIID int64) error {
	return h.doJSON(ctx, http.MethodDelete, fmt.Sprintf("/projects/%s/issues/%d", url.PathEscape(h.cfg.ProjectID), issueIID), nil, nil)
}

// getIssue 获取指定 issue 的当前状态快照。
func (h *gitLabE2EHelper) getIssue(ctx context.Context, issueIID int64) (*gitLabIssueDTO, error) {
	var row gitLabIssueDTO
	if err := h.doJSON(ctx, http.MethodGet, fmt.Sprintf("/projects/%s/issues/%d", url.PathEscape(h.cfg.ProjectID), issueIID), nil, &row); err != nil {
		return nil, err
	}
	return &row, nil
}

// listIssueNotes 拉取 issue 评论列表。
func (h *gitLabE2EHelper) listIssueNotes(ctx context.Context, issueIID int64) ([]gitLabNoteDTO, error) {
	var rows []gitLabNoteDTO
	path := fmt.Sprintf(
		"/projects/%s/issues/%d/notes?per_page=100&page=1&order_by=created_at&sort=desc",
		url.PathEscape(h.cfg.ProjectID),
		issueIID,
	)
	if err := h.doJSON(ctx, http.MethodGet, path, nil, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

// createBranch 基于给定 ref 创建测试分支。
func (h *gitLabE2EHelper) createBranch(ctx context.Context, branch string, ref string) error {
	payload := map[string]any{
		"branch": branch,
		"ref":    ref,
	}
	return h.doJSON(ctx, http.MethodPost, fmt.Sprintf("/projects/%s/repository/branches", url.PathEscape(h.cfg.ProjectID)), payload, nil)
}

// createFile 在测试分支写入文件并提交一次 commit。
func (h *gitLabE2EHelper) createFile(ctx context.Context, branch string, filePath string, content string, commitMessage string) error {
	payload := map[string]any{
		"branch":         branch,
		"content":        content,
		"commit_message": commitMessage,
	}
	path := fmt.Sprintf(
		"/projects/%s/repository/files/%s",
		url.PathEscape(h.cfg.ProjectID),
		url.PathEscape(filePath),
	)
	return h.doJSON(ctx, http.MethodPost, path, payload, nil)
}

// deleteBranch 删除测试分支。
func (h *gitLabE2EHelper) deleteBranch(ctx context.Context, branch string) error {
	path := fmt.Sprintf("/projects/%s/repository/branches/%s", url.PathEscape(h.cfg.ProjectID), url.PathEscape(branch))
	return h.doJSON(ctx, http.MethodDelete, path, nil, nil)
}

// getMergeRequest 获取 MR 状态，用于合并结果断言。
func (h *gitLabE2EHelper) getMergeRequest(ctx context.Context, mrIID int64) (*gitLabMRDTO, error) {
	path := fmt.Sprintf("/projects/%s/merge_requests/%d", url.PathEscape(h.cfg.ProjectID), mrIID)
	var row gitLabMRDTO
	if err := h.doJSON(ctx, http.MethodGet, path, nil, &row); err != nil {
		return nil, err
	}
	return &row, nil
}

// doJSON 执行 GitLab JSON API 调用并解析响应。
func (h *gitLabE2EHelper) doJSON(ctx context.Context, method string, path string, payload any, out any) error {
	urlRaw := strings.TrimRight(h.cfg.APIBase, "/") + path
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal payload: %w", err)
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, urlRaw, body)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("PRIVATE-TOKEN", h.cfg.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.http.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf(
			"gitlab api non-2xx method=%s path=%s status=%d body=%s",
			method,
			path,
			resp.StatusCode,
			truncateString(string(raw), 256),
		)
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}
	return nil
}

// waitFor 以固定间隔轮询检查函数，直到成功或超时。
func waitFor(ctx context.Context, timeout time.Duration, interval time.Duration, check func() (bool, error)) error {
	deadline := time.Now().Add(timeout)
	for {
		ok, err := check()
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout after %s", timeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
}

// containsLabel 判断标签列表中是否包含目标标签（忽略大小写）。
func containsLabel(labels []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, label := range labels {
		if strings.EqualFold(strings.TrimSpace(label), target) {
			return true
		}
	}
	return false
}

// isGitLabNotFound 判断错误是否为 GitLab 404 场景。
func isGitLabNotFound(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, " status=404") || strings.Contains(text, " not found")
}

// truncateString 按最大长度截断字符串，避免日志过长。
func truncateString(in string, max int) string {
	if max <= 0 || len(in) <= max {
		return in
	}
	return in[:max]
}
