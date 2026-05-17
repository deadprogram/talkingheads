package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── chanWriter ────────────────────────────────────────────────────────────────

// chanWriter implements io.Writer by forwarding each write as a trimmed string
// to the events channel. Used to redirect log output into the TUI viewport.
type chanWriter struct {
	ch chan<- string
}

func (w *chanWriter) Write(p []byte) (int, error) {
	msg := strings.TrimRight(string(p), "\n")
	if msg != "" {
		select {
		case w.ch <- msg:
		default:
		}
	}
	return len(p), nil
}

// ── styles ────────────────────────────────────────────────────────────────────

var (
	bannerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")) // bright cyan

	separatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238"))

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	directionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")) // orange — incoming prompt

	responseStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("35")) // green — actor response

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")) // dim — system info
)

// ── messages ──────────────────────────────────────────────────────────────────

// logEventMsg carries a single display line emitted by the actor or listener.
type logEventMsg struct{ text string }

// inputSentMsg is sent after typed text has been forwarded to the actor.
type inputSentMsg struct{ text string }

// ── commands ──────────────────────────────────────────────────────────────────

// waitForEventCmd blocks until one event arrives on eventsCh and returns it as
// a logEventMsg, then re-arms itself automatically.
func waitForEventCmd(eventsCh <-chan string) tea.Cmd {
	return func() tea.Msg {
		return logEventMsg{text: <-eventsCh}
	}
}

// sendInputCmd forwards text to the actor's input channel and confirms via
// inputSentMsg so the TUI can display "You: <text>" in the viewport.
func sendInputCmd(inputCh chan<- string, text string) tea.Cmd {
	return func() tea.Msg {
		inputCh <- text
		return inputSentMsg{text: text}
	}
}

// ── model ─────────────────────────────────────────────────────────────────────

type tuiModel struct {
	// static header
	banner    string
	actorName string
	modelName string

	// scrollable output
	vp    viewport.Model
	lines []string

	// layout
	width, height int
	ready         bool

	// event source (both modes)
	eventsCh <-chan string

	// stdin mode — nil in MQTT mode
	input    textinput.Model
	inputCh  chan<- string
	hasInput bool
}

// newTUIModel constructs the initial model.
//
//   - bannerText is the pre-rendered ASCII banner string.
//   - actorName is the actor's canonical name.
//   - modelName is the short model file name shown in the status bar.
//   - eventsCh receives human-readable lines from the actor or MQTT listener.
//   - inputCh, when non-nil, enables a text input field (stdin mode); typed
//     lines are forwarded to inputCh for the actor's moreFunc to consume.
func newTUIModel(bannerText, actorName, modelName string, eventsCh <-chan string, inputCh chan<- string) tuiModel {
	m := tuiModel{
		banner:    bannerText,
		actorName: actorName,
		modelName: modelName,
		eventsCh:  eventsCh,
		inputCh:   inputCh,
		hasInput:  inputCh != nil,
	}
	if m.hasInput {
		ti := textinput.New()
		ti.Placeholder = "type a message…"
		ti.Focus()
		ti.CharLimit = 512
		m.input = ti
	}
	return m
}

func (m tuiModel) Init() tea.Cmd {
	cmds := []tea.Cmd{waitForEventCmd(m.eventsCh)}
	if m.hasInput {
		cmds = append(cmds, textinput.Blink)
	}
	return tea.Batch(cmds...)
}

// ── update ────────────────────────────────────────────────────────────────────

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		vpCmd    tea.Cmd
		inputCmd tea.Cmd
	)

	switch msg := msg.(type) {

	// ── window resize ────────────────────────────────────────────────────────
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		bannerLines := strings.Count(m.banner, "\n") + 1
		// 1 sep after banner + 1 sep above status + 1 status line
		// + (1 sep + 1 input line) when in stdin mode
		reserved := bannerLines + 3
		if m.hasInput {
			reserved += 2
		}
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

		case "enter":
			if !m.hasInput {
				break
			}
			text := strings.TrimSpace(m.input.Value())
			m.input.Reset()
			if text == "" {
				return m, nil
			}
			return m, sendInputCmd(m.inputCh, text)

		default:
			if m.hasInput {
				m.input, inputCmd = m.input.Update(msg)
				return m, inputCmd
			}
			m.vp, vpCmd = m.vp.Update(msg)
			return m, vpCmd
		}

	// ── actor / listener events ───────────────────────────────────────────────
	case logEventMsg:
		var line string
		switch {
		case strings.HasPrefix(msg.text, "Director:"):
			line = directionStyle.Render(msg.text)
		case strings.HasPrefix(msg.text, "You:"):
			line = directionStyle.Render(msg.text)
		case strings.HasPrefix(msg.text, "ready"):
			line = infoStyle.Render(msg.text)
		default:
			line = responseStyle.Render(fmt.Sprintf("%s: %s", m.actorName, msg.text))
		}
		m.appendLine(line)
		m.vp, vpCmd = m.vp.Update(msg)
		return m, tea.Batch(vpCmd, waitForEventCmd(m.eventsCh))

	// ── stdin input confirmed ─────────────────────────────────────────────────
	case inputSentMsg:
		m.appendLine(directionStyle.Render(fmt.Sprintf("You: %q", msg.text)))
		m.vp, vpCmd = m.vp.Update(msg)
		return m, vpCmd
	}

	// Forward remaining events to viewport (and input if present).
	m.vp, vpCmd = m.vp.Update(msg)
	if m.hasInput {
		m.input, inputCmd = m.input.Update(msg)
		return m, tea.Batch(vpCmd, inputCmd)
	}
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
	statusBar := statusStyle.Render(fmt.Sprintf("actor: %s  ·  model: %s  ·  Ctrl+C: quit", m.actorName, m.modelName))

	parts := []string{
		bannerStyle.Render(m.banner),
		sep,
		m.vp.View(),
		sep,
	}
	if m.hasInput {
		parts = append(parts, m.input.View(), sep)
	}
	parts = append(parts, statusBar)

	return strings.Join(parts, "\n")
}
