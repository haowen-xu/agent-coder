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

// SystemInfo 表示数据结构定义。
type SystemInfo struct {
	ID        uint      `gorm:"primaryKey"`                    // ID 字段说明。
	Key       string    `gorm:"size:128;uniqueIndex;not null"` // Key 字段说明。
	Value     string    `gorm:"size:1024;not null"`            // Value 字段说明。
	CreatedAt time.Time // CreatedAt 字段说明。
	UpdatedAt time.Time // UpdatedAt 字段说明。
}

// User 表示数据结构定义。
type User struct {
	ID           uint           `gorm:"primaryKey"`                   // ID 字段说明。
	Username     string         `gorm:"size:64;uniqueIndex;not null"` // Username 字段说明。
	PasswordHash string         `gorm:"size:255;not null"`            // PasswordHash 字段说明。
	IsAdmin      bool           `gorm:"not null;default:false"`       // IsAdmin 字段说明。
	Enabled      bool           `gorm:"not null;default:true;index"`  // Enabled 字段说明。
	LastLoginAt  *time.Time     `gorm:"type:timestamp"`               // LastLoginAt 字段说明。
	CreatedAt    time.Time      `gorm:"not null"`                     // CreatedAt 字段说明。
	UpdatedAt    time.Time      `gorm:"not null"`                     // UpdatedAt 字段说明。
	DeletedAt    gorm.DeletedAt `gorm:"index"`                        // DeletedAt 字段说明。
}

// UserSession 表示数据结构定义。
type UserSession struct {
	ID        uint           `gorm:"primaryKey"`                    // ID 字段说明。
	UserID    uint           `gorm:"not null;index"`                // UserID 字段说明。
	Token     string         `gorm:"size:128;uniqueIndex;not null"` // Token 字段说明。
	ExpiredAt time.Time      `gorm:"not null;index"`                // ExpiredAt 字段说明。
	CreatedAt time.Time      `gorm:"not null"`                      // CreatedAt 字段说明。
	UpdatedAt time.Time      `gorm:"not null"`                      // UpdatedAt 字段说明。
	DeletedAt gorm.DeletedAt `gorm:"index"`                         // DeletedAt 字段说明。
}

// Project 表示数据结构定义。
type Project struct {
	ID                uint           `gorm:"primaryKey"`                                   // ID 字段说明。
	ProjectKey        string         `gorm:"size:64;not null;uniqueIndex"`                 // ProjectKey 字段说明。
	ProjectSlug       string         `gorm:"size:128;not null;uniqueIndex"`                // ProjectSlug 字段说明。
	Name              string         `gorm:"size:128;not null"`                            // Name 字段说明。
	Provider          string         `gorm:"size:16;not null;default:gitlab"`              // Provider 字段说明。
	ProviderURL       string         `gorm:"size:255;not null"`                            // ProviderURL 字段说明。
	RepoURL           string         `gorm:"type:text;not null"`                           // RepoURL 字段说明。
	DefaultBranch     string         `gorm:"size:64;not null;default:main"`                // DefaultBranch 字段说明。
	IssueProjectID    *string        `gorm:"size:64"`                                      // IssueProjectID 字段说明。
	CredentialRef     string         `gorm:"size:128;not null"`                            // CredentialRef 字段说明。
	ProjectToken      *string        `gorm:"type:text"`                                    // ProjectToken 字段说明。
	SandboxPlanReview bool           `gorm:"not null;default:false"`                       // SandboxPlanReview 字段说明。
	PollIntervalSec   int            `gorm:"not null;default:60;index:idx_project_poll"`   // PollIntervalSec 字段说明。
	Enabled           bool           `gorm:"not null;default:true;index:idx_project_poll"` // Enabled 字段说明。
	LastIssueSyncAt   *time.Time     `gorm:"type:timestamp"`                               // LastIssueSyncAt 字段说明。
	LabelAgentReady   string         `gorm:"size:64;not null;default:Agent Ready"`         // LabelAgentReady 字段说明。
	LabelInProgress   string         `gorm:"size:64;not null;default:In Progress"`         // LabelInProgress 字段说明。
	LabelHumanReview  string         `gorm:"size:64;not null;default:Human Review"`        // LabelHumanReview 字段说明。
	LabelRework       string         `gorm:"size:64;not null;default:Rework"`              // LabelRework 字段说明。
	LabelVerified     string         `gorm:"size:64;not null;default:Verified"`            // LabelVerified 字段说明。
	LabelMerged       string         `gorm:"size:64;not null;default:Merged"`              // LabelMerged 字段说明。
	CreatedBy         uint           `gorm:"not null"`                                     // CreatedBy 字段说明。
	CreatedAt         time.Time      `gorm:"not null"`                                     // CreatedAt 字段说明。
	UpdatedAt         time.Time      `gorm:"not null"`                                     // UpdatedAt 字段说明。
	DeletedAt         gorm.DeletedAt `gorm:"index"`                                        // DeletedAt 字段说明。
}

// Issue 表示数据结构定义。
type Issue struct {
	ID              uint           `gorm:"primaryKey"`                                                                                                     // ID 字段说明。
	ProjectID       uint           `gorm:"not null;index:idx_issue_project_lifecycle;index:idx_issue_project_registered;uniqueIndex:uk_issue_project_iid"` // ProjectID 字段说明。
	IssueIID        int64          `gorm:"not null;uniqueIndex:uk_issue_project_iid"`                                                                      // IssueIID 字段说明。
	IssueUID        *string        `gorm:"size:64"`                                                                                                        // IssueUID 字段说明。
	Title           string         `gorm:"type:text;not null"`                                                                                             // Title 字段说明。
	State           string         `gorm:"size:16;not null"`                                                                                               // State 字段说明。
	LabelsJSON      string         `gorm:"type:text;not null"`                                                                                             // LabelsJSON 字段说明。
	RegisteredAt    time.Time      `gorm:"not null;index:idx_issue_project_registered"`                                                                    // RegisteredAt 字段说明。
	LifecycleStatus string         `gorm:"size:24;not null;default:registered;index:idx_issue_project_lifecycle"`                                          // LifecycleStatus 字段说明。
	IssueDir        string         `gorm:"type:text;not null"`                                                                                             // IssueDir 字段说明。
	BranchName      *string        `gorm:"size:128"`                                                                                                       // BranchName 字段说明。
	MRIID           *int64         // MRIID 字段说明。
	MRURL           *string        `gorm:"type:text"` // MRURL 字段说明。
	CurrentRunID    *uint          // CurrentRunID 字段说明。
	LastSyncedAt    time.Time      `gorm:"not null"`       // LastSyncedAt 字段说明。
	ClosedAt        *time.Time     `gorm:"type:timestamp"` // ClosedAt 字段说明。
	CloseReason     *string        `gorm:"size:32"`        // CloseReason 字段说明。
	CreatedAt       time.Time      `gorm:"not null"`       // CreatedAt 字段说明。
	UpdatedAt       time.Time      `gorm:"not null"`       // UpdatedAt 字段说明。
	DeletedAt       gorm.DeletedAt `gorm:"index"`          // DeletedAt 字段说明。
}

// IssueRun 表示数据结构定义。
type IssueRun struct {
	ID                 uint       `gorm:"primaryKey"`                                                                                                               // ID 字段说明。
	IssueID            uint       `gorm:"not null;index:idx_run_issue_status_created;index:idx_run_issue_kind_status_created;uniqueIndex:uk_run_issue_no"`          // IssueID 字段说明。
	RunNo              int        `gorm:"not null;uniqueIndex:uk_run_issue_no"`                                                                                     // RunNo 字段说明。
	RunKind            string     `gorm:"size:16;not null;index:idx_run_issue_kind_status_created"`                                                                 // RunKind 字段说明。
	TriggerType        string     `gorm:"size:16;not null"`                                                                                                         // TriggerType 字段说明。
	Status             string     `gorm:"size:16;not null;index:idx_run_issue_status_created;index:idx_run_issue_kind_status_created;index:idx_run_status_created"` // Status 字段说明。
	AgentRole          string     `gorm:"size:16;not null"`                                                                                                         // AgentRole 字段说明。
	LoopStep           int        `gorm:"not null;default:1"`                                                                                                       // LoopStep 字段说明。
	MaxLoopStep        int        `gorm:"not null;default:5"`                                                                                                       // MaxLoopStep 字段说明。
	QueuedAt           time.Time  `gorm:"not null"`                                                                                                                 // QueuedAt 字段说明。
	StartedAt          *time.Time `gorm:"type:timestamp"`                                                                                                           // StartedAt 字段说明。
	FinishedAt         *time.Time `gorm:"type:timestamp"`                                                                                                           // FinishedAt 字段说明。
	BranchName         string     `gorm:"size:128;not null"`                                                                                                        // BranchName 字段说明。
	BaseSHA            *string    `gorm:"size:64"`                                                                                                                  // BaseSHA 字段说明。
	HeadSHA            *string    `gorm:"size:64"`                                                                                                                  // HeadSHA 字段说明。
	MRIID              *int64     // MRIID 字段说明。
	MRURL              *string    `gorm:"type:text"`          // MRURL 字段说明。
	GitTreePath        string     `gorm:"type:text;not null"` // GitTreePath 字段说明。
	AgentRunDir        string     `gorm:"type:text;not null"` // AgentRunDir 字段说明。
	ConflictRetryCount int        `gorm:"not null;default:0"` // ConflictRetryCount 字段说明。
	MaxConflictRetry   int        `gorm:"not null;default:5"` // MaxConflictRetry 字段说明。
	ExitCode           *int       // ExitCode 字段说明。
	ErrorSummary       *string    `gorm:"type:text"` // ErrorSummary 字段说明。
	ExecutorSessionID  *string    `gorm:"size:128"`  // ExecutorSessionID 字段说明。
	CreatedByUserID    *uint      // CreatedByUserID 字段说明。
	CreatedAt          time.Time  `gorm:"not null;index:idx_run_status_created"` // CreatedAt 字段说明。
	UpdatedAt          time.Time  `gorm:"not null"`                              // UpdatedAt 字段说明。
}

// RunLog 表示数据结构定义。
type RunLog struct {
	ID          uint      `gorm:"primaryKey"`                                                     // ID 字段说明。
	RunID       uint      `gorm:"not null;index:idx_runlog_run_at;uniqueIndex:uk_runlog_run_seq"` // RunID 字段说明。
	Seq         int       `gorm:"not null;uniqueIndex:uk_runlog_run_seq"`                         // Seq 字段说明。
	At          time.Time `gorm:"not null;index:idx_runlog_run_at;index:idx_runlog_level_at"`     // At 字段说明。
	Level       string    `gorm:"size:8;not null;index:idx_runlog_level_at"`                      // Level 字段说明。
	Stage       string    `gorm:"size:32;not null"`                                               // Stage 字段说明。
	EventType   string    `gorm:"size:32;not null"`                                               // EventType 字段说明。
	Message     string    `gorm:"type:text;not null"`                                             // Message 字段说明。
	PayloadJSON *string   `gorm:"type:text"`                                                      // PayloadJSON 字段说明。
	CreatedAt   time.Time `gorm:"not null"`                                                       // CreatedAt 字段说明。
}

// PromptTemplate 表示数据结构定义。
type PromptTemplate struct {
	ID         uint      `gorm:"primaryKey"`                                                                               // ID 字段说明。
	ProjectKey string    `gorm:"size:64;not null;index:idx_prompt_project_kind_role,unique;index:idx_prompt_project_kind"` // ProjectKey 字段说明。
	RunKind    string    `gorm:"size:16;not null;index:idx_prompt_project_kind_role,unique;index:idx_prompt_project_kind"` // RunKind 字段说明。
	AgentRole  string    `gorm:"size:16;not null;index:idx_prompt_project_kind_role,unique"`                               // AgentRole 字段说明。
	Content    string    `gorm:"type:text;not null"`                                                                       // Content 字段说明。
	CreatedAt  time.Time // CreatedAt 字段说明。
	UpdatedAt  time.Time // UpdatedAt 字段说明。
}
