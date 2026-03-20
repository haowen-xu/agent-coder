package db

import (
	"time"

	"gorm.io/gorm"
)

const (
	ProviderGitLab = "gitlab"
)

const (
	IssueLifecycleRegistered  = "registered"
	IssueLifecycleRunning     = "running"
	IssueLifecycleHumanReview = "human_review"
	IssueLifecycleRework      = "rework"
	IssueLifecycleVerified    = "verified"
	IssueLifecycleFailed      = "failed"
	IssueLifecycleClosed      = "closed"
)

const (
	IssueCloseReasonMerged         = "merged"
	IssueCloseReasonManual         = "manual"
	IssueCloseReasonNeedHumanMerge = "need_human_merge"
)

const (
	RunKindDev   = "dev"
	RunKindMerge = "merge"
)

const (
	RunStatusQueued    = "queued"
	RunStatusRunning   = "running"
	RunStatusSucceeded = "succeeded"
	RunStatusFailed    = "failed"
	RunStatusCanceled  = "canceled"
)

const (
	AgentRoleDev    = "dev"
	AgentRoleReview = "review"
	AgentRoleMerge  = "merge"
)

const (
	TriggerScheduler = "scheduler"
	TriggerManual    = "manual"
	TriggerRework    = "rework"
	TriggerRetry     = "retry"
)

type SystemInfo struct {
	ID        uint   `gorm:"primaryKey"`
	Key       string `gorm:"size:128;uniqueIndex;not null"`
	Value     string `gorm:"size:1024;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type User struct {
	ID           uint           `gorm:"primaryKey"`
	Username     string         `gorm:"size:64;uniqueIndex;not null"`
	PasswordHash string         `gorm:"size:255;not null"`
	IsAdmin      bool           `gorm:"not null;default:false"`
	Enabled      bool           `gorm:"not null;default:true;index"`
	LastLoginAt  *time.Time     `gorm:"type:timestamp"`
	CreatedAt    time.Time      `gorm:"not null"`
	UpdatedAt    time.Time      `gorm:"not null"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

type UserSession struct {
	ID        uint           `gorm:"primaryKey"`
	UserID    uint           `gorm:"not null;index"`
	Token     string         `gorm:"size:128;uniqueIndex;not null"`
	ExpiredAt time.Time      `gorm:"not null;index"`
	CreatedAt time.Time      `gorm:"not null"`
	UpdatedAt time.Time      `gorm:"not null"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type Project struct {
	ID               uint           `gorm:"primaryKey"`
	ProjectKey       string         `gorm:"size:64;not null;uniqueIndex"`
	ProjectSlug      string         `gorm:"size:128;not null;uniqueIndex"`
	Name             string         `gorm:"size:128;not null"`
	Provider         string         `gorm:"size:16;not null;default:gitlab"`
	ProviderURL      string         `gorm:"size:255;not null"`
	RepoURL          string         `gorm:"type:text;not null"`
	DefaultBranch    string         `gorm:"size:64;not null;default:main"`
	IssueProjectID   *string        `gorm:"size:64"`
	CredentialRef    string         `gorm:"size:128;not null"`
	ProjectToken     *string        `gorm:"type:text"`
	PollIntervalSec  int            `gorm:"not null;default:60;index:idx_project_poll"`
	Enabled          bool           `gorm:"not null;default:true;index:idx_project_poll"`
	LastIssueSyncAt  *time.Time     `gorm:"type:timestamp"`
	LabelAgentReady  string         `gorm:"size:64;not null;default:Agent Ready"`
	LabelInProgress  string         `gorm:"size:64;not null;default:In Progress"`
	LabelHumanReview string         `gorm:"size:64;not null;default:Human Review"`
	LabelRework      string         `gorm:"size:64;not null;default:Rework"`
	LabelVerified    string         `gorm:"size:64;not null;default:Verified"`
	LabelMerged      string         `gorm:"size:64;not null;default:Merged"`
	CreatedBy        uint           `gorm:"not null"`
	CreatedAt        time.Time      `gorm:"not null"`
	UpdatedAt        time.Time      `gorm:"not null"`
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

type Issue struct {
	ID              uint      `gorm:"primaryKey"`
	ProjectID       uint      `gorm:"not null;index:idx_issue_project_lifecycle;index:idx_issue_project_registered;uniqueIndex:uk_issue_project_iid"`
	IssueIID        int64     `gorm:"not null;uniqueIndex:uk_issue_project_iid"`
	IssueUID        *string   `gorm:"size:64"`
	Title           string    `gorm:"type:text;not null"`
	State           string    `gorm:"size:16;not null"`
	LabelsJSON      string    `gorm:"type:text;not null"`
	RegisteredAt    time.Time `gorm:"not null;index:idx_issue_project_registered"`
	LifecycleStatus string    `gorm:"size:24;not null;default:registered;index:idx_issue_project_lifecycle"`
	IssueDir        string    `gorm:"type:text;not null"`
	BranchName      *string   `gorm:"size:128"`
	MRIID           *int64
	MRURL           *string `gorm:"type:text"`
	CurrentRunID    *uint
	LastSyncedAt    time.Time      `gorm:"not null"`
	ClosedAt        *time.Time     `gorm:"type:timestamp"`
	CloseReason     *string        `gorm:"size:32"`
	CreatedAt       time.Time      `gorm:"not null"`
	UpdatedAt       time.Time      `gorm:"not null"`
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

type IssueRun struct {
	ID                 uint       `gorm:"primaryKey"`
	IssueID            uint       `gorm:"not null;index:idx_run_issue_status_created;index:idx_run_issue_kind_status_created;uniqueIndex:uk_run_issue_no"`
	RunNo              int        `gorm:"not null;uniqueIndex:uk_run_issue_no"`
	RunKind            string     `gorm:"size:16;not null;index:idx_run_issue_kind_status_created"`
	TriggerType        string     `gorm:"size:16;not null"`
	Status             string     `gorm:"size:16;not null;index:idx_run_issue_status_created;index:idx_run_issue_kind_status_created;index:idx_run_status_created"`
	AgentRole          string     `gorm:"size:16;not null"`
	LoopStep           int        `gorm:"not null;default:1"`
	MaxLoopStep        int        `gorm:"not null;default:5"`
	QueuedAt           time.Time  `gorm:"not null"`
	StartedAt          *time.Time `gorm:"type:timestamp"`
	FinishedAt         *time.Time `gorm:"type:timestamp"`
	BranchName         string     `gorm:"size:128;not null"`
	BaseSHA            *string    `gorm:"size:64"`
	HeadSHA            *string    `gorm:"size:64"`
	MRIID              *int64
	MRURL              *string `gorm:"type:text"`
	GitTreePath        string  `gorm:"type:text;not null"`
	AgentRunDir        string  `gorm:"type:text;not null"`
	ConflictRetryCount int     `gorm:"not null;default:0"`
	MaxConflictRetry   int     `gorm:"not null;default:5"`
	ExitCode           *int
	ErrorSummary       *string `gorm:"type:text"`
	ExecutorSessionID  *string `gorm:"size:128"`
	CreatedByUserID    *uint
	CreatedAt          time.Time `gorm:"not null;index:idx_run_status_created"`
	UpdatedAt          time.Time `gorm:"not null"`
}

type RunLog struct {
	ID          uint      `gorm:"primaryKey"`
	RunID       uint      `gorm:"not null;index:idx_runlog_run_at;uniqueIndex:uk_runlog_run_seq"`
	Seq         int       `gorm:"not null;uniqueIndex:uk_runlog_run_seq"`
	At          time.Time `gorm:"not null;index:idx_runlog_run_at;index:idx_runlog_level_at"`
	Level       string    `gorm:"size:8;not null;index:idx_runlog_level_at"`
	Stage       string    `gorm:"size:32;not null"`
	EventType   string    `gorm:"size:32;not null"`
	Message     string    `gorm:"type:text;not null"`
	PayloadJSON *string   `gorm:"type:text"`
	CreatedAt   time.Time `gorm:"not null"`
}

type PromptTemplate struct {
	ID         uint   `gorm:"primaryKey"`
	ProjectKey string `gorm:"size:64;not null;index:idx_prompt_project_kind_role,unique;index:idx_prompt_project_kind"`
	RunKind    string `gorm:"size:16;not null;index:idx_prompt_project_kind_role,unique;index:idx_prompt_project_kind"`
	AgentRole  string `gorm:"size:16;not null;index:idx_prompt_project_kind_role,unique"`
	Content    string `gorm:"type:text;not null"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
