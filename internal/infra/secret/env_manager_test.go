package secret

import (
	"context"
	"testing"
)

// TestSanitizeEnvKey 用于单元测试。
func TestSanitizeEnvKey(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want string
	}{
		{in: "repo_token", want: "REPO_TOKEN"},
		{in: " Group/Repo.Token ", want: "GROUP_REPO_TOKEN"},
		{in: "a--b__c", want: "A_B_C"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			if got := sanitizeEnvKey(tc.in); got != tc.want {
				t.Fatalf("sanitizeEnvKey(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestEnvManagerGet 用于单元测试。
func TestEnvManagerGet(t *testing.T) {
	m := NewEnvManager("TEST_SECRET_")

	if _, err := m.Get(context.Background(), ""); err == nil {
		t.Fatalf("expected error for empty ref")
	}

	t.Setenv("TEST_SECRET_GITLAB_TOKEN", "  secret-value  ")
	got, err := m.Get(context.Background(), "gitlab.token")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got != "secret-value" {
		t.Fatalf("unexpected secret value: %q", got)
	}

	if _, err := m.Get(context.Background(), "missing"); err == nil {
		t.Fatalf("expected error for missing secret")
	}
}
