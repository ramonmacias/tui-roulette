package main

import (
	"fmt"
	"os"
	"roulette/roulette"
	"strings"

	tea "charm.land/bubbletea/v2"
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
	roulettes           []*roulette.Roulette
	selectedRoulette    int
	selectedParticipant int
	focus               focusArea
	mode                screenMode
	input               string
	errorMessage        string
	infoMessage         string
}

func initialModel() model {
	return model{
		roulettes:           []*roulette.Roulette{},
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
	case tea.KeyPressMsg:
		key := msg.String()

		switch key {
		case "ctrl+c", "q":
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
	s.WriteString("Roulette Manager\n\n")

	if m.errorMessage != "" {
		fmt.Fprintf(&s, "Error: %s\n\n", m.errorMessage)
	}

	if m.infoMessage != "" {
		fmt.Fprintf(&s, "%s\n\n", m.infoMessage)
	}

	s.WriteString(m.renderRoulettes())
	s.WriteString("\n")
	s.WriteString(m.renderParticipants())
	s.WriteString("\n")
	s.WriteString(m.renderPrompt())
	s.WriteString("\n")
	s.WriteString(m.renderHelp())

	v := tea.NewView(s.String())
	v.AltScreen = true
	return v
}

func (m *model) clearMessages() {
	m.errorMessage = ""
	m.infoMessage = ""
}

func (m model) currentRoulette() *roulette.Roulette {
	if len(m.roulettes) == 0 || m.selectedRoulette < 0 || m.selectedRoulette >= len(m.roulettes) {
		return nil
	}

	return m.roulettes[m.selectedRoulette]
}

func (m model) currentParticipant() *roulette.Participant {
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

		m.roulettes = append(m.roulettes, roulette.NewRoulette(name))
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

		if err := r.AddParticipant(roulette.NewParticipant(name)); err != nil {
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
	if err := r.RemoveParticipant(roulette.NewParticipant(participantName)); err != nil {
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
	s.WriteString("Roulettes\n")

	if len(m.roulettes) == 0 {
		s.WriteString("  (none yet)\n")
		return s.String()
	}

	for index, roulette := range m.roulettes {
		marker := m.marker(m.focus == focusRoulettes, m.selectedRoulette == index)
		fmt.Fprintf(&s, "%s %s (%d participants)\n", marker, roulette.Name(), len(roulette.Participants()))
	}

	return s.String()
}

func (m model) renderParticipants() string {
	var s strings.Builder
	r := m.currentRoulette()
	if r == nil {
		s.WriteString("Participants\n  Create a roulette to add participants.\n")
		return s.String()
	}

	fmt.Fprintf(&s, "Participants for %s\n", r.Name())
	if len(r.Participants()) == 0 {
		s.WriteString("  (no participants yet)\n")
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
		return fmt.Sprintf("New roulette name: %s_", m.input)
	case modeAddParticipant:
		roulette := m.currentRoulette()
		if roulette == nil {
			return "New participant: _"
		}

		return fmt.Sprintf("Add participant to %s: %s_", roulette.Name(), m.input)
	default:
		if m.currentRoulette() == nil {
			return "Press n to create a roulette."
		}

		return fmt.Sprintf("Selected roulette: %s", m.currentRoulette().Name())
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
		return ">"
	}

	if isSelected {
		return "*"
	}

	return " "
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

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
