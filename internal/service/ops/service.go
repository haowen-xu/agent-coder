package ops

import (
	"context"
	"time"

	db "github.com/haowen-xu/agent-coder/internal/dal"
	"github.com/haowen-xu/agent-coder/internal/utils"
)

// Metrics 表示数据结构定义。
type Metrics struct {
	Timestamp time.Time      `json:"timestamp"` // Timestamp 字段说明。
	Projects  ProjectsMetric `json:"projects"`  // Projects 字段说明。
	Issues    IssueMetric    `json:"issues"`    // Issues 字段说明。
	Runs      RunMetric      `json:"runs"`      // Runs 字段说明。
}

// ProjectsMetric 表示数据结构定义。
type ProjectsMetric struct {
	Total   int64 `json:"total"`   // Total 字段说明。
	Enabled int64 `json:"enabled"` // Enabled 字段说明。
}

// IssueMetric 表示数据结构定义。
type IssueMetric struct {
	Total       int64            `json:"total"`        // Total 字段说明。
	ByLifecycle map[string]int64 `json:"by_lifecycle"` // ByLifecycle 字段说明。
}

// RunMetric 表示数据结构定义。
type RunMetric struct {
	Total    int64            `json:"total"`     // Total 字段说明。
	ByStatus map[string]int64 `json:"by_status"` // ByStatus 字段说明。
	ByKind   map[string]int64 `json:"by_kind"`   // ByKind 字段说明。
}

// Service 表示数据结构定义。
type Service struct {
	db *db.Client // db 字段说明。
}

// New 执行相关逻辑。
func New(dbClient *db.Client) *Service {
	return &Service{db: dbClient}
}

// GetMetrics 是 *Service 的方法实现。
func (s *Service) GetMetrics(ctx context.Context) (*Metrics, error) {
	projectTotal, projectEnabled, err := s.db.CountProjects(ctx)
	if err != nil {
		return nil, err
	}
	issueTotal, err := s.db.CountIssues(ctx)
	if err != nil {
		return nil, err
	}
	issueByLifecycle, err := s.db.CountIssuesByLifecycle(ctx)
	if err != nil {
		return nil, err
	}
	runTotal, err := s.db.CountRuns(ctx)
	if err != nil {
		return nil, err
	}
	runByStatus, err := s.db.CountRunsByStatus(ctx)
	if err != nil {
		return nil, err
	}
	runByKind, err := s.db.CountRunsByKind(ctx)
	if err != nil {
		return nil, err
	}
	return &Metrics{
		Timestamp: utils.NowUTC(),
		Projects: ProjectsMetric{
			Total:   projectTotal,
			Enabled: projectEnabled,
		},
		Issues: IssueMetric{
			Total:       issueTotal,
			ByLifecycle: issueByLifecycle,
		},
		Runs: RunMetric{
			Total:    runTotal,
			ByStatus: runByStatus,
			ByKind:   runByKind,
		},
	}, nil
}
