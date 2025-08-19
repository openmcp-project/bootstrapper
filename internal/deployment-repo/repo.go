package deploymentrepo

import (
	"errors"
	"fmt"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"

	gitconfig "github.com/openmcp-project/bootstrapper/internal/git-config"
	"github.com/openmcp-project/bootstrapper/internal/log"
)

// gitProgressWriter is a writer that logs Git progress messages.
type gitProgressWriter struct{}

func (w gitProgressWriter) Write(p []byte) (n int, err error) {
	logger := log.GetLogger()
	logger.Tracef("[Git] %s", string(p))
	return len(p), nil
}

// CloneRepo clones a Git repository from the specified URL to the given path.
// It uses the provided gitConfig to configure the clone options with authentication.
func CloneRepo(repoURL, path string, gitConfig *gitconfig.Config) (*git.Repository, error) {
	logger := log.GetLogger()

	logger.Debugf("Cloning repository from %s to %s", repoURL, path)

	cloneOptions := &git.CloneOptions{
		URL:          repoURL,
		SingleBranch: false,
		Progress:     gitProgressWriter{},
	}

	if err := gitConfig.ConfigureCloneOptions(cloneOptions); err != nil {
		return nil, err
	}

	repo, err := git.PlainClone(path, false, cloneOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	return repo, nil
}

// PushRepo pushes the changes in the given repository to the remote.
// It uses the provided gitConfig to configure the push options with authentication.
func PushRepo(repo *git.Repository, branch string, gitConfig *gitconfig.Config) error {
	logger := log.GetLogger()

	logger.Debug("Pushing changes to remote repository")

	pushOptions := &git.PushOptions{
		RefSpecs: []config.RefSpec{
			config.RefSpec(plumbing.HEAD + ":" + plumbing.NewBranchReferenceName(branch)),
		},
		Progress: gitProgressWriter{},
	}

	if err := gitConfig.ConfigurePushOptions(pushOptions); err != nil {
		return fmt.Errorf("failed to configure push options: %w", err)
	}

	if err := repo.Push(pushOptions); err != nil {
		if errors.Is(err, git.NoErrAlreadyUpToDate) {
			logger.Info("No changes to push")
		} else {
			return fmt.Errorf("failed to push changes: %w", err)
		}
	}

	return nil
}

// CommitChanges commits all changes in the repository with the specified message, author name, and email.
func CommitChanges(repo *git.Repository, message, name, email string) error {
	logger := log.GetLogger()

	logger.Debugf("Committing changes with message: %s", message)

	workTree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	hash, err := workTree.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  name,
			Email: email,
			When:  time.Now(),
		},
	})
	if err != nil {
		if errors.Is(err, git.ErrEmptyCommit) {
			logger.Info("No changes to commit")
		} else {
			return fmt.Errorf("failed to commit changes: %w", err)
		}
	}

	if !hash.IsZero() {
		logger.Infof("Created commit: %s", hash.String())
	}

	return nil
}

// CheckoutAndCreateBranchIfNotExists checks out a branch with the given name.
// If the branch does not exist, it creates a new branch with that name and pushes it
// to the remote repository. If the branch already exists, it checks out the existing branch.
func CheckoutAndCreateBranchIfNotExists(repo *git.Repository, branchName string, gitConfig *gitconfig.Config) error {
	logger := log.GetLogger()

	branchExists := false
	references, err := repo.References()
	if err != nil {
		return fmt.Errorf("failed to list references: %w", err)
	}

	localRef := plumbing.NewBranchReferenceName(branchName)
	remoteRef := plumbing.NewRemoteReferenceName("origin", branchName)

	err = references.ForEach(func(ref *plumbing.Reference) error {
		logger.Trace("Found reference: ", ref.Name())
		if ref.Name() == localRef || ref.Name() == remoteRef {
			branchExists = true
			logger.Tracef("Branch %s exists as %s", branchName, ref.Name())
			return storer.ErrStop
		}
		return nil
	})
	if err != nil {
		return err
	}

	workTree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	if !branchExists {
		// Create and checkout new branch
		logger.Debugf("Branch %s does not exist. Creating...\n", branchName)
		err = workTree.Checkout(&git.CheckoutOptions{
			Branch: localRef,
			Create: true,
		})
		if err != nil {
			return fmt.Errorf("failed to create branch: %w", err)
		}

		pushOptions := &git.PushOptions{
			RefSpecs: []config.RefSpec{
				config.RefSpec(localRef + ":" + localRef),
			},
			Progress: gitProgressWriter{},
		}

		if err := gitConfig.ConfigurePushOptions(pushOptions); err != nil {
			return fmt.Errorf("failed to configure push options: %w", err)
		}

		// Push new branch to remote
		err = repo.Push(pushOptions)
		if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
			return fmt.Errorf("failed to push new branch: %w", err)
		}
	}

	// Checkout existing branch
	err = workTree.Checkout(&git.CheckoutOptions{
		Branch: remoteRef,
	})
	if err != nil {
		return fmt.Errorf("failed to checkout branch: %w", err)
	}

	return nil
}
