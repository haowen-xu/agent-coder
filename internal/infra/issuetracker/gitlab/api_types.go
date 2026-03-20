package gitlab

import "time"

type gitLabIssue struct {
	ID        string     `json:"id"`
	IID       int64      `json:"iid"`
	Title     string     `json:"title"`
	State     string     `json:"state"`
	Labels    []string   `json:"labels"`
	WebURL    string     `json:"web_url"`
	ClosedAt  *time.Time `json:"closed_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type gitLabMR struct {
	IID          int64  `json:"iid"`
	WebURL       string `json:"web_url"`
	SourceBranch string `json:"source_branch"`
	TargetBranch string `json:"target_branch"`
	State        string `json:"state"`
}
