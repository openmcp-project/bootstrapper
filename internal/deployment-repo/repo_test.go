package deploymentrepo_test

import (
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/assert"

	deploymentrepo "github.com/openmcp-project/bootstrapper/internal/deployment-repo"
	gitconfig "github.com/openmcp-project/bootstrapper/internal/git-config"
	testutils "github.com/openmcp-project/bootstrapper/test/utils"
)

func Test_Repo(t *testing.T) {
	originDir := t.TempDir()
	targetDir := t.TempDir()

	origin, err := git.PlainInit(originDir, false)
	assert.NoError(t, err)

	originWorkTree, err := origin.Worktree()
	assert.NoError(t, err)
	assert.NotNil(t, originWorkTree)

	dummyFilePath := filepath.Join(originDir, "dummy.txt")
	testutils.WriteToFile(t, dummyFilePath, "This is a dummy file.")
	testutils.AddFileToWorkTree(t, originWorkTree, "dummy.txt")
	testutils.WorkTreeCommit(t, originWorkTree, "Initial commit")

	gitConfig := &gitconfig.Config{}

	repo, err := deploymentrepo.CloneRepo(originDir, targetDir, gitConfig)
	assert.NoError(t, err)
	assert.NotNil(t, repo)

	repoWorkTree, err := repo.Worktree()
	assert.NoError(t, err)
	assert.NotNil(t, repoWorkTree)

	err = deploymentrepo.CheckoutAndCreateBranchIfNotExists(repo, "test", gitConfig)
	assert.NoError(t, err)

	testFilePath := filepath.Join(targetDir, "test.txt")
	testutils.WriteToFile(t, testFilePath, "This is a test file.")
	testutils.AddFileToWorkTree(t, repoWorkTree, "test.txt")

	err = deploymentrepo.CommitChanges(repo, "Add test.txt", "Test User", "noreply@test")
	assert.NoError(t, err)

	err = deploymentrepo.PushRepo(repo, "test", gitConfig)
	assert.NoError(t, err)

	hasTestBranch := false
	branchIter, err := origin.Branches()
	assert.NoError(t, err)
	for {
		branch, err := branchIter.Next()
		if err != nil {
			break
		}
		t.Logf("Branch: %s", branch.Name().String())
		if branch.Name().Short() == "test" {
			hasTestBranch = true
			break
		}
	}

	assert.True(t, hasTestBranch, "Origin repository should have 'test' branch")
}
