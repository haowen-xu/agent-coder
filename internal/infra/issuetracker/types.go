package issuetracker

import "time"

type Issue struct {
	IID       int64
	UID       string
	Title     string
	State     string
	Labels    []string
	WebURL    string
	ClosedAt  *time.Time
	UpdatedAt time.Time
}

type MergeRequest struct {
	IID          int64
	WebURL       string
	SourceBranch string
	TargetBranch string
	State        string
}

type CreateOrUpdateMRRequest struct {
	SourceBranch string
	TargetBranch string
	Title        string
	Description  string
}
