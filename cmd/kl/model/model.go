package model

import (
	"context"
	"fmt"
	"kf/internal/kl"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	viewport   viewport.Model
	kl         *kl.KL
	lines      []string
	autoscroll bool
	ready      bool
	logCh      <-chan string
	errCh      <-chan error
}

type logErrMsg struct{ err error }
type logsClosedMsg struct{}

func New(kl *kl.KL) Model {
	logCh, errCh, err := kl.ReadLogs(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ups: %s", err.Error())
		os.Exit(1)
	}

	return Model{
		kl:         kl,
		logCh:      logCh,
		errCh:      errCh,
		autoscroll: true,
	}
}

func waitForLog(logCh <-chan string, errCh <-chan error) tea.Cmd {
	return func() tea.Msg {
		select {
		case line, ok := <-logCh:
			if !ok {
				return logsClosedMsg{}
			}
			return logMsg(line)
		case err, ok := <-errCh:
			if !ok {
				return logsClosedMsg{}
			}
			return logErrMsg{err}
		}
	}
}

func (m Model) Init() tea.Cmd {
	return waitForLog(m.logCh, m.errCh)
}

func (m *Model) appendLine(s string) {
	m.lines = append(m.lines, colorLine(s))
	m.viewport.SetContent(strings.Join(m.lines, "\n"))
	if m.autoscroll {
		m.viewport.GotoBottom()
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(m.headerView())
		// Since this program is using the full size of the viewport we
		// need to wait until we've received the window dimensions before
		// we can initialize the viewport. The initial dimensions come in
		// quickly, though asynchronously, which is why we wait for them
		// here.
		m.viewport = viewport.New(msg.Width, msg.Height-headerHeight)
		m.viewport.YPosition = headerHeight
		m.ready = true
		return m, nil
	case tea.KeyMsg:
		var cmd tea.Cmd

		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "s":
			// toggle autoscroll
			m.autoscroll = !m.autoscroll
			return m, cmd
		case "up", "k":
			m.viewport.ScrollUp(1)
			return m, cmd
		case "down", "j":
			m.viewport.ScrollDown(1)
		case "left", "h":
			m.viewport.ScrollLeft(1)
			return m, cmd
		case "right", "l":
			m.viewport.ScrollRight(1)
		}

		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	case logMsg:
		m.appendLine(string(msg))
		return m, waitForLog(m.logCh, m.errCh)
	case logErrMsg:
		m.appendLine("ERROR: " + msg.err.Error())
		return m, waitForLog(m.logCh, m.errCh)
	case logsClosedMsg:
		return m, nil
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}
