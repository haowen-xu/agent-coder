package httpserver

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/common/ut"

	appcfg "github.com/haowen-xu/agent-coder/internal/config"
	db "github.com/haowen-xu/agent-coder/internal/dal"
	"github.com/haowen-xu/agent-coder/internal/infra/agent/promptstore"
	coresvc "github.com/haowen-xu/agent-coder/internal/service/core"
)

func newHTTPServerForTest(t *testing.T) (*Server, *db.Client) {
	t.Helper()
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &appcfg.Config{
		App: appcfg.AppConfig{Name: "agent-coder-test", Env: "test"},
		Server: appcfg.ServerConfig{
			Host:            "127.0.0.1",
			Port:            18082,
			ReadTimeout:     "5s",
			WriteTimeout:    "5s",
			ShutdownTimeout: "2s",
		},
		DB: appcfg.DBConfig{
			Enabled:         true,
			Driver:          "sqlite",
			SQLitePath:      filepath.Join(t.TempDir(), "httpserver.db"),
			MaxOpenConns:    4,
			MaxIdleConns:    2,
			ConnMaxLifetime: "1m",
			AutoMigrate:     true,
		},
		Auth: appcfg.AuthConfig{SessionTTL: "1h"},
	}
	dbClient, err := db.New(ctx, cfg.DB, log)
	if err != nil {
		t.Fatalf("init db failed: %v", err)
	}
	t.Cleanup(func() { _ = dbClient.Close() })
	if err := dbClient.EnsureBootstrapAdmin(ctx, "admin", "admin123"); err != nil {
		t.Fatalf("ensure bootstrap admin failed: %v", err)
	}
	ps := promptstore.NewService(dbClient)
	coreService := coresvc.New(cfg, dbClient, ps)
	return New(cfg, log, dbClient, coreService), dbClient
}

func httpReq(s *Server, method, path, body, token string, extraHeaders ...ut.Header) *ut.ResponseRecorder {
	headers := make([]ut.Header, 0, 2+len(extraHeaders))
	if token != "" {
		headers = append(headers, ut.Header{Key: "Authorization", Value: "Bearer " + token})
	}
	headers = append(headers, extraHeaders...)
	var reqBody *ut.Body
	if body != "" {
		reqBody = &ut.Body{
			Body: strings.NewReader(body),
			Len:  len(body),
		}
		headers = append(headers, ut.Header{Key: "Content-Type", Value: "application/json"})
	}
	return ut.PerformRequest(s.hz.Engine, method, path, reqBody, headers...)
}

func decodeBodyMap(t *testing.T, rec *ut.ResponseRecorder) map[string]any {
	t.Helper()
	out := map[string]any{}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode body failed: %v body=%s", err, rec.Body.String())
	}
	return out
}

func mustAdminToken(t *testing.T, s *Server) string {
	t.Helper()
	loginAdmin := httpReq(s, "POST", "/api/v1/auth/login", `{"username":"admin","password":"admin123"}`, "")
	if loginAdmin.Code != 200 {
		t.Fatalf("admin login should be 200, got %d body=%s", loginAdmin.Code, loginAdmin.Body.String())
	}
	loginAdminJSON := decodeBodyMap(t, loginAdmin)
	token, _ := loginAdminJSON["token"].(string)
	if token == "" {
		t.Fatalf("admin token should not be empty")
	}
	return token
}

// TestServerEndpointsIntegration 用于单元测试。
func TestServerEndpointsIntegration(t *testing.T) {
	ctx := context.Background()
	s, dbClient := newHTTPServerForTest(t)

	health := httpReq(s, "GET", "/healthz", "", "")
	if health.Code != 200 {
		t.Fatalf("healthz status mismatch: %d body=%s", health.Code, health.Body.String())
	}
	healthJSON := decodeBodyMap(t, health)
	if healthJSON["status"] != "ok" {
		t.Fatalf("healthz payload mismatch: %v", healthJSON)
	}

	meta := httpReq(s, "GET", "/api/v1/meta", "", "")
	if meta.Code != 200 {
		t.Fatalf("meta status mismatch: %d body=%s", meta.Code, meta.Body.String())
	}

	loginBadJSON := httpReq(s, "POST", "/api/v1/auth/login", "{bad json", "")
	if loginBadJSON.Code != 400 {
		t.Fatalf("bad json login should be 400, got %d", loginBadJSON.Code)
	}
	loginBadCred := httpReq(s, "POST", "/api/v1/auth/login", `{"username":"admin","password":"wrong"}`, "")
	if loginBadCred.Code != 401 {
		t.Fatalf("bad credentials login should be 401, got %d", loginBadCred.Code)
	}
	loginAdmin := httpReq(s, "POST", "/api/v1/auth/login", `{"username":"admin","password":"admin123"}`, "")
	if loginAdmin.Code != 200 {
		t.Fatalf("admin login should be 200, got %d body=%s", loginAdmin.Code, loginAdmin.Body.String())
	}
	loginAdminJSON := decodeBodyMap(t, loginAdmin)
	adminToken, _ := loginAdminJSON["token"].(string)
	if adminToken == "" {
		t.Fatalf("admin token should not be empty")
	}

	meNoToken := httpReq(s, "GET", "/api/v1/auth/me", "", "")
	if meNoToken.Code != 401 {
		t.Fatalf("me without token should be 401, got %d", meNoToken.Code)
	}
	meAdmin := httpReq(s, "GET", "/api/v1/auth/me", "", adminToken)
	if meAdmin.Code != 200 {
		t.Fatalf("me with token should be 200, got %d", meAdmin.Code)
	}

	createUser := httpReq(s, "POST", "/api/v1/admin/users", `{"username":"dev1","password":"pass123","is_admin":false,"enabled":true}`, adminToken)
	if createUser.Code != 200 {
		t.Fatalf("admin create user should be 200, got %d body=%s", createUser.Code, createUser.Body.String())
	}
	loginDev := httpReq(s, "POST", "/api/v1/auth/login", `{"username":"dev1","password":"pass123"}`, "")
	if loginDev.Code != 200 {
		t.Fatalf("dev login should be 200, got %d body=%s", loginDev.Code, loginDev.Body.String())
	}
	loginDevJSON := decodeBodyMap(t, loginDev)
	devToken, _ := loginDevJSON["token"].(string)
	if devToken == "" {
		t.Fatalf("dev token should not be empty")
	}
	adminByDev := httpReq(s, "GET", "/api/v1/admin/users", "", devToken)
	if adminByDev.Code != 403 {
		t.Fatalf("non-admin access should be 403, got %d", adminByDev.Code)
	}

	listUsers := httpReq(s, "GET", "/api/v1/admin/users", "", adminToken)
	if listUsers.Code != 200 {
		t.Fatalf("admin list users should be 200, got %d", listUsers.Code)
	}
	listUsersJSON := decodeBodyMap(t, listUsers)
	users, _ := listUsersJSON["items"].([]any)
	if len(users) < 2 {
		t.Fatalf("expected at least 2 users, got %d", len(users))
	}

	createProjectBody := `{
		"project_key":"p-http",
		"project_slug":"group/p-http",
		"name":"Project HTTP",
		"provider":"gitlab",
		"provider_url":"https://gitlab.example.com/api/v4",
		"repo_url":"https://gitlab.example.com/group/p-http",
		"default_branch":"main",
		"project_token":"token-http",
		"sandbox_plan_review":false,
		"poll_interval_sec":30,
		"enabled":true
	}`
	createProject := httpReq(s, "POST", "/api/v1/admin/projects", createProjectBody, adminToken)
	if createProject.Code != 200 {
		t.Fatalf("admin create project should be 200, got %d body=%s", createProject.Code, createProject.Body.String())
	}
	listProjects := httpReq(s, "GET", "/api/v1/admin/projects", "", adminToken)
	if listProjects.Code != 200 {
		t.Fatalf("admin list projects should be 200, got %d", listProjects.Code)
	}

	updateProjectBody := `{
		"project_key":"p-http",
		"project_slug":"group/p-http",
		"name":"Project HTTP Updated",
		"provider":"gitlab",
		"provider_url":"https://gitlab.example.com/api/v4",
		"repo_url":"https://gitlab.example.com/group/p-http",
		"default_branch":"main",
		"project_token":"token-http-new",
		"enabled":true
	}`
	updateProject := httpReq(s, "PUT", "/api/v1/admin/projects/p-http", updateProjectBody, adminToken)
	if updateProject.Code != 200 {
		t.Fatalf("admin update project should be 200, got %d body=%s", updateProject.Code, updateProject.Body.String())
	}

	boardProjects := httpReq(s, "GET", "/api/v1/board/projects", "", adminToken)
	if boardProjects.Code != 200 {
		t.Fatalf("board projects should be 200, got %d", boardProjects.Code)
	}

	projectRow, err := dbClient.GetProjectByKey(ctx, "p-http")
	if err != nil || projectRow == nil {
		t.Fatalf("project p-http should exist: row=%#v err=%v", projectRow, err)
	}
	issue := &db.Issue{
		ProjectID:       projectRow.ID,
		IssueIID:        701,
		Title:           "HTTP issue",
		State:           "opened",
		LabelsJSON:      "[]",
		RegisteredAt:    time.Now(),
		LifecycleStatus: db.IssueLifecycleRegistered,
		IssueDir:        "/tmp/http-issue",
		LastSyncedAt:    time.Now(),
	}
	if err := dbClient.CreateIssue(ctx, issue); err != nil {
		t.Fatalf("create issue for http tests failed: %v", err)
	}

	boardIssues := httpReq(s, "GET", "/api/v1/board/projects/p-http/issues", "", adminToken)
	if boardIssues.Code != 200 {
		t.Fatalf("board project issues should be 200, got %d body=%s", boardIssues.Code, boardIssues.Body.String())
	}
	adminProjectIssues := httpReq(s, "GET", "/api/v1/admin/projects/p-http/issues", "", adminToken)
	if adminProjectIssues.Code != 200 {
		t.Fatalf("admin project issues should be 200, got %d", adminProjectIssues.Code)
	}

	defaultPrompts := httpReq(s, "GET", "/api/v1/admin/prompts/defaults", "", adminToken)
	if defaultPrompts.Code != 200 {
		t.Fatalf("list default prompts should be 200, got %d", defaultPrompts.Code)
	}
	projectPrompts := httpReq(s, "GET", "/api/v1/admin/projects/p-http/prompts", "", adminToken)
	if projectPrompts.Code != 200 {
		t.Fatalf("list project prompts should be 200, got %d", projectPrompts.Code)
	}
	putPrompt := httpReq(s, "PUT", "/api/v1/admin/projects/p-http/prompts/dev/dev", `{"content":"override from http test"}`, adminToken)
	if putPrompt.Code != 200 {
		t.Fatalf("put project prompt should be 200, got %d body=%s", putPrompt.Code, putPrompt.Body.String())
	}
	deletePrompt := httpReq(s, "DELETE", "/api/v1/admin/projects/p-http/prompts/dev/dev", "", adminToken)
	if deletePrompt.Code != 200 {
		t.Fatalf("delete project prompt should be 200, got %d body=%s", deletePrompt.Code, deletePrompt.Body.String())
	}

	run := &db.IssueRun{
		IssueID:          issue.ID,
		RunNo:            1,
		RunKind:          db.RunKindDev,
		TriggerType:      db.TriggerScheduler,
		Status:           db.RunStatusQueued,
		AgentRole:        db.AgentRoleDev,
		LoopStep:         1,
		MaxLoopStep:      5,
		QueuedAt:         time.Now(),
		BranchName:       "agent/http-701",
		GitTreePath:      "/tmp/git-tree-http",
		AgentRunDir:      "/tmp/agent-http",
		MaxConflictRetry: 3,
	}
	if err := dbClient.CreateRun(ctx, run); err != nil {
		t.Fatalf("create run failed: %v", err)
	}
	if err := dbClient.AppendRunLog(ctx, &db.RunLog{
		RunID:     run.ID,
		Seq:       1,
		Level:     "INFO",
		Stage:     "agent",
		EventType: "dev",
		Message:   "log from http test",
	}); err != nil {
		t.Fatalf("append run log failed: %v", err)
	}

	adminIssueRuns := httpReq(s, "GET", "/api/v1/admin/issues/"+itoa(issue.ID)+"/runs", "", adminToken)
	if adminIssueRuns.Code != 200 {
		t.Fatalf("admin issue runs should be 200, got %d", adminIssueRuns.Code)
	}
	adminRunLogs := httpReq(s, "GET", "/api/v1/admin/runs/"+itoa(run.ID)+"/logs", "", adminToken)
	if adminRunLogs.Code != 200 {
		t.Fatalf("admin run logs should be 200, got %d", adminRunLogs.Code)
	}

	cancelRun := httpReq(s, "POST", "/api/v1/admin/runs/"+itoa(run.ID)+"/cancel", `{"reason":"stop now"}`, adminToken)
	if cancelRun.Code != 200 {
		t.Fatalf("admin cancel run should be 200, got %d body=%s", cancelRun.Code, cancelRun.Body.String())
	}

	issue.LifecycleStatus = db.IssueLifecycleClosed
	if err := dbClient.SaveIssue(ctx, issue); err != nil {
		t.Fatalf("save issue closed failed: %v", err)
	}
	retryClosed := httpReq(s, "POST", "/api/v1/admin/issues/"+itoa(issue.ID)+"/retry", "", adminToken)
	if retryClosed.Code != 400 {
		t.Fatalf("retry closed issue should be 400, got %d", retryClosed.Code)
	}
	issue.LifecycleStatus = db.IssueLifecycleRegistered
	issue.CurrentRunID = nil
	issue.CloseReason = nil
	if err := dbClient.SaveIssue(ctx, issue); err != nil {
		t.Fatalf("save issue for retry failed: %v", err)
	}
	retryIssue := httpReq(s, "POST", "/api/v1/admin/issues/"+itoa(issue.ID)+"/retry", "", adminToken)
	if retryIssue.Code != 200 {
		t.Fatalf("retry issue should be 200, got %d body=%s", retryIssue.Code, retryIssue.Body.String())
	}

	projectRow.LastIssueSyncAt = ptrTime(time.Now())
	if err := dbClient.SaveProject(ctx, projectRow); err != nil {
		t.Fatalf("save project last_issue_sync_at failed: %v", err)
	}
	resetCursor := httpReq(s, "POST", "/api/v1/admin/projects/p-http/reset-sync-cursor", "", adminToken)
	if resetCursor.Code != 200 {
		t.Fatalf("reset sync cursor should be 200, got %d body=%s", resetCursor.Code, resetCursor.Body.String())
	}

	metrics := httpReq(s, "GET", "/api/v1/admin/metrics", "", adminToken)
	if metrics.Code != 200 {
		t.Fatalf("admin metrics should be 200, got %d", metrics.Code)
	}

	missingAPIRoute := httpReq(s, "GET", "/api/v1/not-found", "", "")
	if missingAPIRoute.Code != 404 {
		t.Fatalf("missing api route should be 404, got %d", missingAPIRoute.Code)
	}
	gzipIndex := httpReq(s, "GET", "/index.html", "", "", ut.Header{Key: "Accept-Encoding", Value: "gzip"})
	if gzipIndex.Code != 200 {
		t.Fatalf("index route should be 200, got %d", gzipIndex.Code)
	}
	spaRoute := httpReq(s, "GET", "/some/spa/path", "", "")
	if spaRoute.Code != 200 {
		t.Fatalf("spa route should be 200, got %d", spaRoute.Code)
	}
}

// TestServerAdminEndpointsErrorBranches 覆盖 admin handler 的参数/JSON/不存在资源错误分支。
func TestServerAdminEndpointsErrorBranches(t *testing.T) {
	s, _ := newHTTPServerForTest(t)
	adminToken := mustAdminToken(t, s)

	cases := []struct {
		name   string
		method string
		path   string
		body   string
		code   int
	}{
		{name: "create_user_invalid_json", method: "POST", path: "/api/v1/admin/users", body: "{bad json", code: 400},
		{name: "create_user_validation_error", method: "POST", path: "/api/v1/admin/users", body: `{"username":"","password":"","is_admin":false}`, code: 400},
		{name: "update_user_invalid_id", method: "PUT", path: "/api/v1/admin/users/not-number", body: `{}`, code: 400},
		{name: "update_user_invalid_json", method: "PUT", path: "/api/v1/admin/users/1", body: "{bad json", code: 400},
		{name: "create_project_invalid_json", method: "POST", path: "/api/v1/admin/projects", body: "{bad json", code: 400},
		{name: "create_project_validation_error", method: "POST", path: "/api/v1/admin/projects", body: `{"project_key":"","project_slug":"","name":"","provider_url":"","repo_url":""}`, code: 400},
		{name: "update_project_missing_key", method: "PUT", path: "/api/v1/admin/projects/%20", body: `{}`, code: 400},
		{name: "update_project_not_found", method: "PUT", path: "/api/v1/admin/projects/not-found", body: `{"project_key":"not-found","project_slug":"g/not-found","name":"n","provider":"gitlab","provider_url":"https://gitlab.example.com/api/v4","repo_url":"https://gitlab.example.com/g/not-found","project_token":"token"}`, code: 400},
		{name: "list_project_prompts_missing_key", method: "GET", path: "/api/v1/admin/projects/%20/prompts", code: 400},
		{name: "put_project_prompt_missing_key", method: "PUT", path: "/api/v1/admin/projects/%20/prompts/dev/dev", body: `{"content":"x"}`, code: 400},
		{name: "put_project_prompt_missing_run_kind", method: "PUT", path: "/api/v1/admin/projects/p/prompts/%20/dev", body: `{"content":"x"}`, code: 400},
		{name: "put_project_prompt_invalid_json", method: "PUT", path: "/api/v1/admin/projects/p/prompts/dev/dev", body: "{bad json", code: 400},
		{name: "delete_project_prompt_missing_key", method: "DELETE", path: "/api/v1/admin/projects/%20/prompts/dev/dev", code: 400},
		{name: "delete_project_prompt_missing_run_kind", method: "DELETE", path: "/api/v1/admin/projects/p/prompts/%20/dev", code: 400},
		{name: "project_issues_missing_key", method: "GET", path: "/api/v1/admin/projects/%20/issues", code: 400},
		{name: "project_issues_not_found", method: "GET", path: "/api/v1/admin/projects/not-found/issues", code: 400},
		{name: "issue_runs_invalid_id", method: "GET", path: "/api/v1/admin/issues/not-number/runs", code: 400},
		{name: "issue_runs_not_found", method: "GET", path: "/api/v1/admin/issues/999999/runs", code: 400},
		{name: "run_logs_invalid_id", method: "GET", path: "/api/v1/admin/runs/not-number/logs", code: 400},
		{name: "run_logs_not_found", method: "GET", path: "/api/v1/admin/runs/999999/logs", code: 400},
		{name: "retry_issue_invalid_id", method: "POST", path: "/api/v1/admin/issues/not-number/retry", code: 400},
		{name: "retry_issue_not_found", method: "POST", path: "/api/v1/admin/issues/999999/retry", code: 400},
		{name: "cancel_run_invalid_id", method: "POST", path: "/api/v1/admin/runs/not-number/cancel", code: 400},
		{name: "cancel_run_invalid_json", method: "POST", path: "/api/v1/admin/runs/1/cancel", body: "{bad json", code: 400},
		{name: "cancel_run_not_found", method: "POST", path: "/api/v1/admin/runs/999999/cancel", body: `{}`, code: 400},
		{name: "reset_sync_cursor_missing_key", method: "POST", path: "/api/v1/admin/projects/%20/reset-sync-cursor", code: 400},
		{name: "reset_sync_cursor_not_found", method: "POST", path: "/api/v1/admin/projects/not-found/reset-sync-cursor", code: 400},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			rec := httpReq(s, tc.method, tc.path, tc.body, adminToken)
			if rec.Code != tc.code {
				t.Fatalf("status mismatch: got=%d want=%d body=%s", rec.Code, tc.code, rec.Body.String())
			}
		})
	}
}

func itoa(v uint) string {
	return strconv.FormatUint(uint64(v), 10)
}

func ptrTime(v time.Time) *time.Time {
	return &v
}
