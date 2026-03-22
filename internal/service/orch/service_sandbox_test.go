package orch

import (
	"testing"

	db "github.com/haowen-xu/agent-coder/internal/dal"
)

// TestShouldUseSandboxForRole 用于单元测试。
func TestShouldUseSandboxForRole(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		project  db.Project
		role     string
		expected bool
	}{
		{
			name:     "dev always disabled",
			project:  db.Project{SandboxPlanReview: true},
			role:     db.AgentRoleDev,
			expected: false,
		},
		{
			name:     "merge always disabled",
			project:  db.Project{SandboxPlanReview: true},
			role:     db.AgentRoleMerge,
			expected: false,
		},
		{
			name:     "review follows project switch on",
			project:  db.Project{SandboxPlanReview: true},
			role:     db.AgentRoleReview,
			expected: true,
		},
		{
			name:     "review follows project switch off",
			project:  db.Project{SandboxPlanReview: false},
			role:     db.AgentRoleReview,
			expected: false,
		},
		{
			name:     "plan follows project switch on",
			project:  db.Project{SandboxPlanReview: true},
			role:     "plan",
			expected: true,
		},
		{
			name:     "unknown role defaults off",
			project:  db.Project{SandboxPlanReview: true},
			role:     "unknown",
			expected: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := shouldUseSandboxForRole(tc.project, tc.role)
			if got != tc.expected {
				t.Fatalf("unexpected sandbox decision: role=%s got=%v expected=%v", tc.role, got, tc.expected)
			}
		})
	}
}
