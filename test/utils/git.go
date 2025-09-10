package utils

import (
	"os"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func WriteToFile(t *testing.T, filePath, content string) {
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write to file %s: %v", filePath, err)
	}
}

func ReadFromFile(t *testing.T, filePath string) string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read from file %s: %v", filePath, err)
	}
	return string(data)
}

func AddFileToWorkTree(t *testing.T, workTree *git.Worktree, filePath string) {
	_, err := workTree.Add(filePath)
	if err != nil {
		t.Fatalf("failed to add file %s to worktree: %v", filePath, err)
	}
}

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
