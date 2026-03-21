package gitlab

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// gitLabID 定义相关类型。
type gitLabID string

// UnmarshalJSON 是 *gitLabID 的方法实现。
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

// gitLabIssue 表示数据结构定义。
type gitLabIssue struct {
	ID        gitLabID   `json:"id"`         // ID 字段说明。
	IID       int64      `json:"iid"`        // IID 字段说明。
	Title     string     `json:"title"`      // Title 字段说明。
	State     string     `json:"state"`      // State 字段说明。
	Labels    []string   `json:"labels"`     // Labels 字段说明。
	WebURL    string     `json:"web_url"`    // WebURL 字段说明。
	ClosedAt  *time.Time `json:"closed_at"`  // ClosedAt 字段说明。
	UpdatedAt time.Time  `json:"updated_at"` // UpdatedAt 字段说明。
}

// gitLabMR 表示数据结构定义。
type gitLabMR struct {
	IID          int64  `json:"iid"`           // IID 字段说明。
	WebURL       string `json:"web_url"`       // WebURL 字段说明。
	SourceBranch string `json:"source_branch"` // SourceBranch 字段说明。
	TargetBranch string `json:"target_branch"` // TargetBranch 字段说明。
	State        string `json:"state"`         // State 字段说明。
}

// gitLabIssueNote 表示数据结构定义。
type gitLabIssueNote struct {
	ID   int64  `json:"id"`   // ID 字段说明。
	Body string `json:"body"` // Body 字段说明。
}
