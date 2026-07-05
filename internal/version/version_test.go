package version

import "testing"

func TestCurrentUsesInjectedVersion(t *testing.T) {
	oldVersion := Version
	t.Cleanup(func() { Version = oldVersion })

	Version = "v9.9.9"
	if got := Current(); got != "v9.9.9" {
		t.Fatalf("Current() = %q, want v9.9.9", got)
	}
}

func TestCurrentCommitUsesInjectedCommit(t *testing.T) {
	oldCommit := Commit
	t.Cleanup(func() { Commit = oldCommit })

	Commit = "abc123"
	if got := CurrentCommit(); got != "abc123" {
		t.Fatalf("CurrentCommit() = %q, want abc123", got)
	}
}
