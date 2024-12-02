package tmux

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/wmatex/automux/internal/cmd_exec"
	"github.com/wmatex/automux/internal/utils"
)

const SEPARATOR = "|"

type Session struct {
	Name         string
	Active       bool
	Path         string
	LastActivity int
}

type Pane struct {
	Cmd string
}

type Window struct {
	Name  string
	Panes []*Pane
}

func (w *Window) Merge(o *Window) {
	panes := w.Panes
	panes = append(panes, o.Panes...)
	w.Panes = utils.Merge(panes)
}

func (w *Window) Id() string {
	return w.Name
}

func (p *Pane) Merge(o *Pane) {
	// No action required
}

func (p *Pane) Id() string {
	return p.Cmd
}

func createCmdBuilder(args []string) *cmd_exec.CmdExecBuilder {
	return cmd_exec.NewCmdExec("tmux", args)
}

func windowTarget(session string, window int) string {
	return fmt.Sprintf("%s:%d", session, window)
}

func paneTarget(windowID string, pane int) string {
	return fmt.Sprintf("%s.%d", windowID, pane)
}

func NewSession(name, path string) (error, int) {
	if path != "" {
		return createCmdBuilder([]string{"new-session", "-d", "-s", name, "-c", path}).Exec()
	} else {
		return createCmdBuilder([]string{"new-session", "-d", "-s", name}).Exec()
	}
}

func CreateWindowsForProject(session, path string, windows []*Window) error {
	for i, w := range windows {
		windowId := windowTarget(session, i+1)
		err, _ := CreateWindow(windowId, w.Name, path)
		if err != nil {
			return err
		}

		for j, p := range w.Panes {
			paneId := paneTarget(windowId, j)
			if j > 0 {
				err, _ := CreatePane(windowId, paneId, path)
				if err != nil {
					return err
				}
			}

			err, _ := RunCommandInPane(paneId, p.Cmd)
			if err != nil {
				return err
			}
		}

		SelectLayout(windowId)
	}

	SelectWindow(windowTarget(session, 1))

	return nil
}

func CreateWindow(id, name, path string) (error, int) {
	return createCmdBuilder([]string{
		"new-window", "-k",
		"-c", path,
		"-n", name,
		"-t", id,
	}).Exec()
}

func SelectWindow(id string) (error, int) {
	return createCmdBuilder([]string{
		"select-window",
		"-t", id,
	}).Exec()
}

func SelectLayout(windowId string) (error, int) {
	return createCmdBuilder([]string{
		"select-layout",
		"-t", windowId,
		"main-vertical",
	}).Exec()
}

func CreatePane(windowId, id, path string) (error, int) {
	return createCmdBuilder([]string{
		"split-window",
		"-t", windowId,
		"-c", path,
	}).Exec()
}

func RunCommandInPane(id, cmd string) (error, int) {
	return createCmdBuilder([]string{
		"send-keys",
		"-t", id,
		cmd, "C-m",
	}).Exec()
}

func SwitchToSession(name string) (error, int) {
	return createCmdBuilder([]string{"switch-client", "-t", name}).Exec()
}

func GetActiveSessions() ([]Session, error) {
	cmd := createCmdBuilder([]string{
		"list-sessions",
		"-F",
		fmt.Sprintf("#S%[1]s#{session_attached}%[1]s#{session_path}%[1]s#{session_activity}", SEPARATOR),
	})

	output, err, _ := cmd.ExecWithOutput()
	if err != nil {
		return nil, err
	}

	sessions := []Session{}
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		parts := strings.Split(line, SEPARATOR)
		active := false
		if parts[1] == "1" {
			active = true
		}

		lastActivity, err := strconv.Atoi(parts[3])
		if err != nil {
			log.Fatalf("cannot parse last activity time: %s\n", err)
		}
		session := Session{
			Name:         parts[0],
			Active:       active,
			Path:         parts[2],
			LastActivity: lastActivity,
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}
