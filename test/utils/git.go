package utils

import (
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// AddFileToWorkTree stages a file in the given worktree
func AddFileToWorkTree(t *testing.T, workTree *git.Worktree, filePath string) {
	_, err := workTree.Add(filePath)
	if err != nil {
		t.Fatalf("failed to add file %s to worktree: %v", filePath, err)
	}
}

// WorkTreeCommit commits staged changes in the given worktree with the specified message
func WorkTreeCommit(t *testing.T, workTree *git.Worktree, message string) {
	_, err := workTree.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "noreply@test",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("failed to commit changes: %v", err)
	}
}
