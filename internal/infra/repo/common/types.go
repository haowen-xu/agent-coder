package common

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type ErrNeedHumanMerge struct {
	Provider   string
	StatusCode int
	Reason     string
}

func (e *ErrNeedHumanMerge) Error() string {
	if e == nil {
		return "need human merge"
	}
	reason := strings.TrimSpace(e.Reason)
	if reason == "" {
		reason = "need human merge"
	}
	if strings.TrimSpace(e.Provider) == "" {
		if e.StatusCode > 0 {
			return fmt.Sprintf("%s (status=%d)", reason, e.StatusCode)
		}
		return reason
	}
	if e.StatusCode > 0 {
		return fmt.Sprintf("%s provider=%s status=%d", reason, e.Provider, e.StatusCode)
	}
	return fmt.Sprintf("%s provider=%s", reason, e.Provider)
}

func IsNeedHumanMerge(err error) bool {
	var target *ErrNeedHumanMerge
	return errors.As(err, &target)
}

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

type ListIssuesOptions struct {
	State        string
	UpdatedAfter *time.Time
	PerPage      int
	MaxPages     int
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
