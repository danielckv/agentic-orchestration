package dispatcher

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type WorktreeManager struct {
	repoPath    string
	worktreeDir string
}

func NewWorktreeManager(repoPath, worktreeDir string) *WorktreeManager {
	if worktreeDir == "" {
		worktreeDir = ".worktrees"
	}
	return &WorktreeManager{
		repoPath:    repoPath,
		worktreeDir: worktreeDir,
	}
}

func (w *WorktreeManager) Create(taskID string) (string, error) {
	wtPath := w.Path(taskID)
	branch := fmt.Sprintf("task/%s", taskID)
	cmd := exec.Command("git", "worktree", "add", wtPath, "-b", branch, "main")
	cmd.Dir = w.repoPath
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git worktree add: %w: %s", err, out)
	}
	return wtPath, nil
}

func (w *WorktreeManager) Remove(taskID string) error {
	wtPath := w.Path(taskID)
	branch := fmt.Sprintf("task/%s", taskID)

	rmCmd := exec.Command("git", "worktree", "remove", wtPath)
	rmCmd.Dir = w.repoPath
	if out, err := rmCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree remove: %w: %s", err, out)
	}

	brCmd := exec.Command("git", "branch", "-d", branch)
	brCmd.Dir = w.repoPath
	if out, err := brCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git branch -d: %w: %s", err, out)
	}

	return nil
}

func (w *WorktreeManager) Path(taskID string) string {
	dir := w.worktreeDir
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(w.repoPath, dir)
	}
	return filepath.Join(dir, fmt.Sprintf("task-%s", taskID))
}

func (w *WorktreeManager) List() ([]string, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = w.repoPath
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git worktree list: %w", err)
	}

	var paths []string
	for _, line := range strings.Split(string(out), "\n") {
		if after, ok := strings.CutPrefix(line, "worktree "); ok {
			paths = append(paths, after)
		}
	}
	return paths, nil
}
