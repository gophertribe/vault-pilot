package sync

import (
	"fmt"
	"os"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

// GitManager handles git operations
type GitManager struct {
	RepoPath string
}

// NewGitManager creates a new GitManager
func NewGitManager(repoPath string) *GitManager {
	return &GitManager{RepoPath: repoPath}
}

// Sync commits all changes and pushes to remote
func (g *GitManager) Sync(message string) error {
	// Open Repo
	r, err := git.PlainOpen(g.RepoPath)
	if err != nil {
		return fmt.Errorf("failed to open repo: %w", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Add all changes
	_, err = w.Add(".")
	if err != nil {
		return fmt.Errorf("failed to add changes: %w", err)
	}

	// Commit
	if message == "" {
		message = fmt.Sprintf("Auto-sync: %s", time.Now().Format(time.RFC3339))
	}

	_, err = w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Vault Pilot",
			Email: "pilot@vault.local",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	// Push
	// Auth is tricky. Let's try to use SSH agent if available, or just default.
	// For now, let's assume SSH key at default location or agent.
	// go-git requires explicit auth for SSH usually.

	// Check if we have an SSH key path in env, else try agent.
	// For MVP, let's try pushing without auth args (works if using http/https with credential helper, or sometimes ssh agent)
	// Actually go-git doesn't support system credential helper out of the box easily.
	// Let's try to use the default SSH key.

	home, _ := os.UserHomeDir()
	sshKeyPath := fmt.Sprintf("%s/.ssh/id_rsa", home)

	publicKeys, err := ssh.NewPublicKeysFromFile("git", sshKeyPath, "")
	if err != nil {
		// Fallback: Try pushing without auth (might work if public repo or other mechanism)
		// Or just return error/warning.
		// For now, let's just log it and try.
		fmt.Printf("Warning: Could not load SSH key: %v. Trying push without explicit auth.\n", err)
		err = r.Push(&git.PushOptions{})
	} else {
		err = r.Push(&git.PushOptions{
			Auth: publicKeys,
		})
	}

	if err != nil {
		if err == git.NoErrAlreadyUpToDate {
			return nil
		}
		return fmt.Errorf("failed to push: %w", err)
	}

	return nil
}
