package dispatcher

import (
	"os/exec"
	"strings"
)

type TmuxManager struct{}

func NewTmuxManager() *TmuxManager {
	return &TmuxManager{}
}

func (t *TmuxManager) CreateSession(name, command string) error {
	return exec.Command("tmux", "new-session", "-d", "-s", name, command).Run()
}

func (t *TmuxManager) KillSession(name string) error {
	return exec.Command("tmux", "kill-session", "-t", name).Run()
}

func (t *TmuxManager) ListSessions() ([]string, error) {
	out, err := exec.Command("tmux", "list-sessions", "-F", "#{session_name}").Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var sessions []string
	for _, line := range lines {
		if line != "" {
			sessions = append(sessions, line)
		}
	}
	return sessions, nil
}

func (t *TmuxManager) SessionExists(name string) bool {
	err := exec.Command("tmux", "has-session", "-t", name).Run()
	return err == nil
}

func (t *TmuxManager) SendKeys(session, keys string) error {
	return exec.Command("tmux", "send-keys", "-t", session, keys, "Enter").Run()
}
