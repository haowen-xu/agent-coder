package gitlab

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type gitLabID string

func (v *gitLabID) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*v = ""
		return nil
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		*v = gitLabID(s)
		return nil
	}
	var n int64
	if err := json.Unmarshal(data, &n); err == nil {
		*v = gitLabID(strconv.FormatInt(n, 10))
		return nil
	}
	return fmt.Errorf("unsupported gitlab id payload: %s", string(data))
}

type gitLabIssue struct {
	ID        gitLabID   `json:"id"`
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
