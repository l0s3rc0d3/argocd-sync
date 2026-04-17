package repository

import (
	"fmt"
	"os"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
)

type Options struct {
	URL         string
	PAT         string
	AuthorName  string
	AuthorEmail string
}

type Client struct {
	opts    Options
	workDir string
	repo    *git.Repository
}

func NewClient(opts Options) (*Client, error) {
	if opts.URL == "" {
		return nil, fmt.Errorf("repository URL is required")
	}
	if opts.PAT == "" {
		return nil, fmt.Errorf("repository PAT is required")
	}

	tmpDir, err := os.MkdirTemp("", "argocd-sync-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	return &Client{
		opts:    opts,
		workDir: tmpDir,
	}, nil
}

func (c *Client) WorkDir() string {
	return c.workDir
}

func (c *Client) Cleanup() error {
	return os.RemoveAll(c.workDir)
}

func (c *Client) auth() *githttp.BasicAuth {
	return &githttp.BasicAuth{
		Username: "git",
		Password: c.opts.PAT,
	}
}

func (c *Client) Clone() error {
	repo, err := git.PlainClone(c.workDir, false, &git.CloneOptions{
		URL:  c.opts.URL,
		Auth: c.auth(),
	})
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}
	c.repo = repo
	return nil
}

func (c *Client) CommitAndPush(relativeFilePath, message string) error {
	if c.repo == nil {
		return fmt.Errorf("repository not cloned yet")
	}

	w, err := c.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	if _, err := w.Add(relativeFilePath); err != nil {
		return fmt.Errorf("failed to stage %q: %w", relativeFilePath, err)
	}

	status, err := w.Status()
	if err != nil {
		return fmt.Errorf("failed to get worktree status: %w", err)
	}
	if status.IsClean() {
		return fmt.Errorf("no changes detected in %q, nothing to commit", relativeFilePath)
	}

	_, err = w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  c.opts.AuthorName,
			Email: c.opts.AuthorEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	if err := c.repo.Push(&git.PushOptions{
		Auth: c.auth(),
	}); err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	return nil
}