package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── styles ────────────────────────────────────────────────────────────────────

var (
	bannerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")) // bright cyan

	separatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238"))

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	speakStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("35")) // green

	warnStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")) // red
)

// ── messages ──────────────────────────────────────────────────────────────────

// logEventMsg carries a single line of text received from the Listener.
type logEventMsg struct{ text string }

// ── commands ──────────────────────────────────────────────────────────────────

// waitForEventCmd blocks until one event arrives on eventsCh and returns it as
// a logEventMsg, re-arming itself so the viewport stays up to date.
func waitForEventCmd(eventsCh <-chan string) tea.Cmd {
	return func() tea.Msg {
		return logEventMsg{text: <-eventsCh}
	}
}

// ── model ─────────────────────────────────────────────────────────────────────

type tuiModel struct {
	// static header
	banner string

	// scrollable output
	vp    viewport.Model
	lines []string

	// layout
	width, height int
	ready         bool

	// event source
	eventsCh <-chan string

	// metadata for the status bar
	voiceNames []string
}

// newTUIModel constructs the initial model.
//
//   - bannerText is the pre-rendered ASCII banner string.
//   - eventsCh receives human-readable lines from the Listener.
//   - voiceNames are the canonical names of the configured voices.
func newTUIModel(bannerText string, eventsCh <-chan string, voiceNames []string) tuiModel {
	return tuiModel{
		banner:     bannerText,
		eventsCh:   eventsCh,
		voiceNames: voiceNames,
	}
}

func (m tuiModel) Init() tea.Cmd {
	return waitForEventCmd(m.eventsCh)
}

// ── update ────────────────────────────────────────────────────────────────────

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var vpCmd tea.Cmd

	switch msg := msg.(type) {

	// ── window resize ────────────────────────────────────────────────────────
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		bannerLines := strings.Count(m.banner, "\n") + 1
		// 1 separator after banner + 1 separator above status + 1 status line
		reserved := bannerLines + 3
		vpHeight := m.height - reserved
		if vpHeight < 2 {
			vpHeight = 2
		}

		if !m.ready {
			m.vp = viewport.New(m.width, vpHeight)
			m.vp.SetContent("")
			m.ready = true
		} else {
			m.vp.Width = m.width
			m.vp.Height = vpHeight
		}
		m.vp, vpCmd = m.vp.Update(msg)
		return m, vpCmd

	// ── keyboard ─────────────────────────────────────────────────────────────
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		default:
			m.vp, vpCmd = m.vp.Update(msg)
			return m, vpCmd
		}

	// ── listener events ───────────────────────────────────────────────────────
	case logEventMsg:
		var line string
		if strings.HasPrefix(msg.text, "WARNING") {
			line = warnStyle.Render(msg.text)
		} else {
			line = speakStyle.Render(msg.text)
		}
		m.appendLine(line)
		m.vp, vpCmd = m.vp.Update(msg)
		return m, tea.Batch(vpCmd, waitForEventCmd(m.eventsCh))
	}

	m.vp, vpCmd = m.vp.Update(msg)
	return m, vpCmd
}

// appendLine adds a line to the viewport and scrolls to the bottom.
func (m *tuiModel) appendLine(line string) {
	m.lines = append(m.lines, line)
	m.vp.SetContent(strings.Join(m.lines, "\n"))
	m.vp.GotoBottom()
}

// ── view ──────────────────────────────────────────────────────────────────────

func (m tuiModel) View() string {
	if !m.ready {
		return "initialising…\n"
	}

	sep := separatorStyle.Render(strings.Repeat("─", m.width))

	statusBar := statusStyle.Render(fmt.Sprintf("voices: %v  ·  Ctrl+C: quit", m.voiceNames))

	return strings.Join([]string{
		bannerStyle.Render(m.banner),
		sep,
		m.vp.View(),
		sep,
		statusBar,
	}, "\n")
}
