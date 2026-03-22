package orch

import (
	"log/slog"
	"time"

	appcfg "github.com/haowen-xu/agent-coder/internal/config"
	db "github.com/haowen-xu/agent-coder/internal/dal"
	"github.com/haowen-xu/agent-coder/internal/infra/agent/base"
	"github.com/haowen-xu/agent-coder/internal/infra/agent/codex"
	"github.com/haowen-xu/agent-coder/internal/infra/agent/promptstore"
	infraGit "github.com/haowen-xu/agent-coder/internal/infra/git"
	infraorch "github.com/haowen-xu/agent-coder/internal/infra/orch"
	"github.com/haowen-xu/agent-coder/internal/infra/secret"
)

// Service 表示数据结构定义。
type Service struct {
	cfg        *appcfg.Config       // cfg 字段说明。
	log        *slog.Logger         // log 字段说明。
	db         *db.Client           // db 字段说明。
	ps         *promptstore.Service // ps 字段说明。
	agent      base.Client          // agent 字段说明。
	git        *infraGit.Client     // git 字段说明。
	secret     secret.Manager       // secret 字段说明。
	lastPolled map[uint]time.Time   // lastPolled 字段说明。
	orchWork   *infraorch.WorkDir   // orchWork 字段说明。
	orchQueue  *infraorch.OrchWorkerQueue
}

const (
	issueNoteMarkerRunStatus   = "<!-- agent-coder:run-status -->"
	issueNoteMarkerMergeStatus = "<!-- agent-coder:merge-status -->"
	issueNoteMarkerMRReady     = "<!-- agent-coder:mr-ready -->"
	runClaimBatchSize          = 20
)

// New 执行相关逻辑。
func New(
	cfg *appcfg.Config,
	log *slog.Logger,
	dbClient *db.Client,
	ps *promptstore.Service,
	secretManager secret.Manager,
) *Service {
	agentClient := codex.NewClient(cfg.Agent.Codex.Binary, cfg.Agent.Codex.Sandbox)
	return &Service{
		cfg:        cfg,
		log:        log,
		db:         dbClient,
		ps:         ps,
		agent:      agentClient,
		git:        infraGit.NewClient(),
		secret:     secretManager,
		lastPolled: make(map[uint]time.Time),
		orchWork:   infraorch.NewWorkDir(cfg.Work.WorkDir),
		orchQueue:  infraorch.NewOrchWorkerQueue(runClaimBatchSize, 1),
	}
}
