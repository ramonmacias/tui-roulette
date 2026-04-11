package roulette

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

const (
	ansiReset      = "\033[0m"
	ansiBold       = "\033[1m"
	ansiDim        = "\033[2m"
	ansiFgBase     = "\033[38;2;241;236;236m"
	ansiFgMuted    = "\033[38;2;167;160;160m"
	ansiFgAccent   = "\033[38;2;255;166;77m"
	ansiFgAccent2  = "\033[38;2;255;214;170m"
	ansiFgDanger   = "\033[38;2;255;107;107m"
	ansiFgSuccess  = "\033[38;2;122;214;163m"
	ansiBgBase     = "\033[48;2;33;30;30m"
	ansiBgSurface  = "\033[48;2;49;45;45m"
	ansiBgSurface2 = "\033[48;2;64;59;59m"
)

type focusArea int

const (
	focusRoulettes focusArea = iota
	focusParticipants
)

type screenMode int

const (
	modeBrowse screenMode = iota
	modeCreateRoulette
	modeAddParticipant
)

type model struct {
	roulettes           []*Roulette
	selectedRoulette    int
	selectedParticipant int
	focus               focusArea
	mode                screenMode
	width               int
	height              int
	input               string
	errorMessage        string
	infoMessage         string
}

// InitialModel builds the Bubble Tea model used by the roulette TUI.
func InitialModel() tea.Model {
	return model{
		roulettes:           []*Roulette{},
		selectedRoulette:    0,
		selectedParticipant: 0,
		focus:               focusRoulettes,
		mode:                modeCreateRoulette,
		infoMessage:         "Create your first roulette to get started.",
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		key := msg.String()

		switch key {
		case "ctrl+c":
			return m, tea.Quit
		}

		if key == "q" && m.mode == modeBrowse {
			return m, tea.Quit
		}

		m.clearMessages()

		switch m.mode {
		case modeCreateRoulette:
			m = m.updateCreateRoulette(key)
		case modeAddParticipant:
			m = m.updateAddParticipant(key)
		default:
			m = m.updateBrowse(key)
		}
	}

	return m, nil
}

func (m model) View() tea.View {
	var s strings.Builder
	s.WriteString(paint(" ROULETTE MANAGER ", ansiBold, ansiBgSurface2, ansiFgAccent))
	s.WriteString("\n\n")

	if m.errorMessage != "" {
		fmt.Fprintf(&s, "%s %s\n\n", paint("✕", ansiFgDanger, ansiBold), paint(m.errorMessage, ansiFgDanger))
	}

	if m.infoMessage != "" {
		fmt.Fprintf(&s, "%s %s\n\n", paint("●", ansiFgSuccess, ansiBold), paint(m.infoMessage, ansiFgAccent2))
	}

	s.WriteString(panel("ROULETTES", m.renderRoulettes()))
	s.WriteString("\n")
	s.WriteString(panel("PARTICIPANTS", m.renderParticipants()))
	s.WriteString("\n")
	s.WriteString(panel("INPUT", m.renderPrompt()))
	s.WriteString("\n")
	s.WriteString(paint(m.renderHelp(), ansiDim, ansiFgMuted))

	content := centerBlock(s.String(), m.width, m.height)
	v := tea.NewView(paint(content, ansiFgBase))
	v.AltScreen = true
	return v
}

func (m *model) clearMessages() {
	m.errorMessage = ""
	m.infoMessage = ""
}

func (m model) currentRoulette() *Roulette {
	if len(m.roulettes) == 0 || m.selectedRoulette < 0 || m.selectedRoulette >= len(m.roulettes) {
		return nil
	}

	return m.roulettes[m.selectedRoulette]
}

func (m model) currentParticipant() *Participant {
	roulette := m.currentRoulette()
	if roulette == nil || len(roulette.Participants()) == 0 || m.selectedParticipant < 0 || m.selectedParticipant >= len(roulette.Participants()) {
		return nil
	}

	return &roulette.Participants()[m.selectedParticipant]
}

func (m model) updateBrowse(key string) model {
	switch key {
	case "n":
		m.mode = modeCreateRoulette
		m.input = ""
		m.infoMessage = "Type a roulette name and press enter."
	case "a":
		if m.currentRoulette() == nil {
			m.errorMessage = "Create a roulette before adding participants."
			return m
		}

		m.mode = modeAddParticipant
		m.input = ""
		m.focus = focusParticipants
		m.infoMessage = fmt.Sprintf("Add participants to %s.", m.currentRoulette().Name())
	case "tab":
		m.toggleFocus()
	case "left", "h":
		m.focus = focusRoulettes
	case "right", "l":
		if m.currentRoulette() != nil {
			m.focus = focusParticipants
		}
	case "up", "k":
		m.moveSelection(-1)
	case "down", "j":
		m.moveSelection(1)
	case "d", "x", "backspace":
		m.removeCurrentParticipant()
	}

	return m
}

func (m model) updateCreateRoulette(key string) model {
	switch key {
	case "esc":
		if len(m.roulettes) > 0 {
			m.mode = modeBrowse
			m.input = ""
		}
	case "enter":
		name := strings.TrimSpace(m.input)
		if name == "" {
			m.errorMessage = "Roulette name cannot be empty."
			return m
		}

		m.roulettes = append(m.roulettes, NewRoulette(name))
		m.selectedRoulette = len(m.roulettes) - 1
		m.selectedParticipant = 0
		m.focus = focusParticipants
		m.mode = modeAddParticipant
		m.input = ""
		m.infoMessage = fmt.Sprintf("Roulette %s created. Add participants and press esc when done.", name)
	case "backspace":
		m.input = deleteLastRune(m.input)
	case "space":
		m.input += " "
	default:
		m.input = appendInput(m.input, key)
	}

	return m
}

func (m model) updateAddParticipant(key string) model {
	r := m.currentRoulette()
	if r == nil {
		m.mode = modeCreateRoulette
		m.errorMessage = "Create a roulette before adding participants."
		return m
	}

	switch key {
	case "esc":
		m.mode = modeBrowse
		m.input = ""
		m.infoMessage = fmt.Sprintf("Done editing %s.", r.Name())
	case "enter":
		name := strings.TrimSpace(m.input)
		if name == "" {
			m.errorMessage = "Participant name cannot be empty."
			return m
		}

		if err := r.AddParticipant(NewParticipant(name)); err != nil {
			m.errorMessage = err.Error()
			return m
		}

		m.selectedParticipant = len(r.Participants()) - 1
		m.input = ""
		m.focus = focusParticipants
		m.infoMessage = fmt.Sprintf("Participant %s added to %s. Press enter to add another or esc when done.", name, r.Name())
	case "backspace":
		m.input = deleteLastRune(m.input)
	case "space":
		m.input += " "
	default:
		m.input = appendInput(m.input, key)
	}

	return m
}

func (m *model) toggleFocus() {
	if m.focus == focusRoulettes {
		if m.currentRoulette() != nil {
			m.focus = focusParticipants
		}
		return
	}

	m.focus = focusRoulettes
}

func (m *model) moveSelection(step int) {
	if m.focus == focusParticipants {
		r := m.currentRoulette()
		if r == nil || len(r.Participants()) == 0 {
			return
		}

		m.selectedParticipant = clamp(m.selectedParticipant+step, 0, len(r.Participants())-1)
		return
	}

	if len(m.roulettes) == 0 {
		return
	}

	m.selectedRoulette = clamp(m.selectedRoulette+step, 0, len(m.roulettes)-1)
	r := m.currentRoulette()
	if r == nil || len(r.Participants()) == 0 {
		m.selectedParticipant = 0
		return
	}

	m.selectedParticipant = clamp(m.selectedParticipant, 0, len(r.Participants())-1)
}

func (m *model) removeCurrentParticipant() {
	if m.focus != focusParticipants {
		m.errorMessage = "Switch to the participants panel to remove someone."
		return
	}

	r := m.currentRoulette()
	participant := m.currentParticipant()
	if r == nil || participant == nil {
		m.errorMessage = "Select a participant to remove."
		return
	}

	participantName := participant.Name()
	if err := r.RemoveParticipant(NewParticipant(participantName)); err != nil {
		m.errorMessage = err.Error()
		return
	}

	if len(r.Participants()) == 0 {
		m.selectedParticipant = 0
	} else {
		m.selectedParticipant = clamp(m.selectedParticipant, 0, len(r.Participants())-1)
	}

	m.infoMessage = fmt.Sprintf("Participant %s removed from %s.", participantName, r.Name())
}

func (m model) renderRoulettes() string {
	var s strings.Builder

	if len(m.roulettes) == 0 {
		s.WriteString(paint("(none yet)", ansiFgMuted, ansiDim))
		s.WriteString("\n")
		return s.String()
	}

	for index, roulette := range m.roulettes {
		marker := m.marker(m.focus == focusRoulettes, m.selectedRoulette == index)
		fmt.Fprintf(&s, "%s %s %s\n", marker, paint(roulette.Name(), ansiBold), paint(fmt.Sprintf("(%d participants)", len(roulette.Participants())), ansiFgMuted))
	}

	return s.String()
}

func (m model) renderParticipants() string {
	var s strings.Builder
	r := m.currentRoulette()
	if r == nil {
		s.WriteString(paint("Create a roulette to add participants.", ansiFgMuted))
		s.WriteString("\n")
		return s.String()
	}

	fmt.Fprintf(&s, "%s %s\n", paint("for", ansiFgMuted), paint(r.Name(), ansiFgAccent2, ansiBold))
	if len(r.Participants()) == 0 {
		s.WriteString(paint("(no participants yet)", ansiFgMuted, ansiDim))
		s.WriteString("\n")
		return s.String()
	}

	for index, participant := range r.Participants() {
		marker := m.marker(m.focus == focusParticipants, m.selectedParticipant == index)
		fmt.Fprintf(&s, "%s %s\n", marker, participant.Name())
	}

	return s.String()
}

func (m model) renderPrompt() string {
	switch m.mode {
	case modeCreateRoulette:
		return fmt.Sprintf("%s %s", paint("New roulette name:", ansiFgAccent), paint(fmt.Sprintf("%s_", m.input), ansiBgSurface, ansiFgBase))
	case modeAddParticipant:
		roulette := m.currentRoulette()
		if roulette == nil {
			return fmt.Sprintf("%s %s", paint("New participant:", ansiFgAccent), paint("_", ansiBgSurface, ansiFgBase))
		}

		return fmt.Sprintf("%s %s %s", paint("Add participant to", ansiFgAccent), paint(roulette.Name(), ansiBold), paint(fmt.Sprintf(": %s_", m.input), ansiBgSurface, ansiFgBase))
	default:
		if m.currentRoulette() == nil {
			return paint("Press n to create a roulette.", ansiFgMuted)
		}

		return fmt.Sprintf("%s %s", paint("Selected roulette:", ansiFgMuted), paint(m.currentRoulette().Name(), ansiBold, ansiFgAccent2))
	}
}

func (m model) renderHelp() string {
	switch m.mode {
	case modeCreateRoulette:
		return "enter create roulette • esc cancel • q quit"
	case modeAddParticipant:
		return "enter add participant • esc finish • q quit"
	default:
		return "n new roulette • a add participant • d remove participant • tab/←/→ switch panel • ↑/↓ move • q quit"
	}
}

func (m model) marker(hasFocus bool, isSelected bool) string {
	if hasFocus && isSelected {
		return paint("◆", ansiFgAccent, ansiBold)
	}

	if isSelected {
		return paint("•", ansiFgAccent2)
	}

	return paint("·", ansiFgMuted)
}

func paint(text string, codes ...string) string {
	if len(codes) == 0 {
		return text
	}

	return strings.Join(codes, "") + text + ansiReset
}

func panel(title string, body string) string {
	var s strings.Builder

	rows := strings.Split(strings.TrimSuffix(body, "\n"), "\n")
	if len(rows) == 0 {
		rows = []string{""}
	}

	minInnerWidth := 60
	innerWidth := minInnerWidth
	titleCell := " " + title + " "
	innerWidth = max(innerWidth, visibleWidth(titleCell))
	for _, row := range rows {
		innerWidth = max(innerWidth, visibleWidth(row)+2)
	}

	header := paint(titleCell, ansiBold, ansiFgAccent, ansiBgSurface)
	topFill := paint(strings.Repeat("─", max(0, innerWidth-visibleWidth(titleCell))), ansiFgMuted)
	s.WriteString(paint("╭", ansiFgMuted))
	s.WriteString(header)
	s.WriteString(topFill)
	s.WriteString(paint("╮", ansiFgMuted))
	s.WriteString("\n")

	for _, row := range rows {
		padding := max(0, innerWidth-2-visibleWidth(row))
		s.WriteString(paint("│", ansiFgMuted))
		s.WriteString(" ")
		s.WriteString(row)
		s.WriteString(strings.Repeat(" ", padding))
		s.WriteString(" ")
		s.WriteString(paint("│", ansiFgMuted))
		s.WriteString("\n")
	}

	s.WriteString(paint("╰"+strings.Repeat("─", innerWidth)+"╯", ansiFgMuted))
	return s.String()
}

func centerBlock(content string, width int, height int) string {
	if width <= 0 || height <= 0 {
		return content
	}

	lines := strings.Split(content, "\n")
	contentHeight := len(lines)
	contentWidth := 0
	for _, line := range lines {
		contentWidth = max(contentWidth, visibleWidth(line))
	}

	leftPad := max(0, (width-contentWidth)/2)
	topPad := max(0, (height-contentHeight)/2)

	var out strings.Builder
	if topPad > 0 {
		out.WriteString(strings.Repeat("\n", topPad))
	}

	for i, line := range lines {
		if leftPad > 0 {
			out.WriteString(strings.Repeat(" ", leftPad))
		}
		out.WriteString(line)
		if i < len(lines)-1 {
			out.WriteString("\n")
		}
	}

	return out.String()
}

func visibleWidth(s string) int {
	width := 0
	inEscape := false

	for _, r := range s {
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}

		if r == '\x1b' {
			inEscape = true
			continue
		}

		if r == '\n' || r == '\r' {
			continue
		}

		width++
	}

	return width
}

func appendInput(current string, key string) string {
	if len([]rune(key)) == 1 {
		return current + key
	}

	return current
}

func deleteLastRune(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return value
	}

	return string(runes[:len(runes)-1])
}

func clamp(value int, minValue int, maxValue int) int {
	if value < minValue {
		return minValue
	}

	if value > maxValue {
		return maxValue
	}

	return value
}
