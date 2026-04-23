package internal

import (
	"fmt"
	"math"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

const (
	ansiReset          = "\033[0m"
	ansiBold           = "\033[1m"
	ansiDim            = "\033[2m"
	ansiFgBase         = "\033[38;2;241;236;236m"
	ansiFgMuted        = "\033[38;2;167;160;160m"
	ansiFgAccent       = "\033[38;2;255;166;77m"
	ansiFgAccent2      = "\033[38;2;255;214;170m"
	ansiFgDanger       = "\033[38;2;255;107;107m"
	ansiFgSuccess      = "\033[38;2;122;214;163m"
	ansiBgBase         = "\033[48;2;33;30;30m"
	ansiBgSurface      = "\033[48;2;49;45;45m"
	ansiBgSurface2     = "\033[48;2;64;59;59m"
	minPanelInnerWidth = 60
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
	modeCreateMultiWinnerCount
	modeAddParticipant
)

type model struct {
	roulettes           []*Roulette
	selectedRoulette    int
	selectedParticipant int
	createName          string
	createMode          Mode
	createMultiCount    int
	lastWinners         map[int]string
	spinning            bool
	spinToken           int
	spinRouletteIndex   int
	spinWinnerName      string
	spinWinnerSlice     int
	spinPendingWinners  []string
	spinParticipants    []Participant
	spinCurrentSlice    int
	spinTotalSteps      int
	spinStep            int
	spinVisibleWinners  int
	spinVisibleElim     int
	spinOutcomeWinner   bool
	focus               focusArea
	mode                screenMode
	width               int
	height              int
	input               string
	errorMessage        string
	infoMessage         string
	storage             Storage
	showSpinPopup       bool
	planColor           string
	planRouletteID      string
}

// InitialModel builds the Bubble Tea model used by the roulette TUI.
func InitialModel(storage Storage) tea.Model {
	// Load roulettes from storage
	roulettes := []*Roulette{}
	if storage != nil {
		if loaded, err := storage.Load(); err == nil {
			roulettes = loaded
		}
	}

	infoMsg := "Create your first roulette to get started."
	screenMode := modeCreateRoulette
	if len(roulettes) > 0 {
		infoMsg = "Browse your roulettes or create a new one."
		screenMode = modeBrowse
	}

	planRouletteID := ""
	if len(roulettes) > 0 {
		planRouletteID = roulettes[0].ID()
	}

	return model{
		roulettes:           roulettes,
		selectedRoulette:    0,
		selectedParticipant: 0,
		createName:          "",
		createMode:          ModeRepeatableWinners,
		createMultiCount:    0,
		lastWinners:         map[int]string{},
		focus:               focusRoulettes,
		mode:                screenMode,
		infoMessage:         infoMsg,
		storage:             storage,
		planColor:           randomPlanColor(),
		planRouletteID:      planRouletteID,
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
	case spinTickMsg:
		if !m.spinning || msg.token != m.spinToken {
			return m, nil
		}

		return m.advanceSpin()
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

		if m.mode == modeBrowse && m.showSpinPopup {
			var cmd tea.Cmd
			m, cmd = m.updateSpinPopup(key)
			return m, cmd
		}

		switch m.mode {
		case modeCreateRoulette:
			m = m.updateCreateRoulette(key)
		case modeCreateMultiWinnerCount:
			m = m.updateCreateMultiWinnerCount(key)
		case modeAddParticipant:
			m = m.updateAddParticipant(key)
		default:
			var cmd tea.Cmd
			m, cmd = m.updateBrowse(key)
			return m, cmd
		}
	}

	return m, nil
}

func (m model) View() tea.View {
	var s strings.Builder
	contentWidth := max(72, min(118, m.width-4))
	if m.width <= 0 {
		contentWidth = 96
	}

	s.WriteString(centerLines(m.renderTopBar(), contentWidth))
	s.WriteString("\n\n")
	s.WriteString(centerLines(m.renderRouletteLogo(), contentWidth))
	s.WriteString("\n\n")
	s.WriteString(centerLines(panel("INPUT", m.renderPrompt()), contentWidth))
	s.WriteString("\n\n")

	if m.errorMessage != "" {
		msg := fmt.Sprintf("%s %s", paint("✕", ansiFgDanger, ansiBold), paint(m.errorMessage, ansiFgDanger))
		s.WriteString(centerLines(msg, contentWidth))
		s.WriteString("\n\n")
	}

	if m.infoMessage != "" {
		msg := fmt.Sprintf("%s %s", paint("●", ansiFgSuccess, ansiBold), paint(m.infoMessage, ansiFgAccent2))
		s.WriteString(centerLines(msg, contentWidth))
		s.WriteString("\n\n")
	}

	s.WriteString(centerLines(m.renderDashboard(), contentWidth))
	s.WriteString("\n")
	if m.showSpinPopup {
		s.WriteString(centerLines(m.renderSpinPopup(), contentWidth))
		s.WriteString("\n")
	}

	helpWidth := max(36, min(88, contentWidth-2))
	wrappedHelp := wrapText(m.renderHelp(), helpWidth)
	s.WriteString(centerLines(paint(wrappedHelp, ansiDim, ansiFgMuted), contentWidth))

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

func (m model) updateBrowse(key string) (model, tea.Cmd) {
	if m.spinning {
		return m, nil
	}

	switch key {
	case "n":
		m.mode = modeCreateRoulette
		m.input = ""
		m.infoMessage = "Type a roulette name and press enter."
	case "a":
		if m.currentRoulette() == nil {
			m.errorMessage = "Create a roulette before adding participants."
			return m, nil
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
	case "s":
		next, cmd := m.startSpinCurrentRoulette()
		if cmd != nil {
			next.showSpinPopup = true
		}
		return next, cmd
	case "r":
		m.resetCurrentRoulette()
	case "d", "x", "backspace":
		m.removeCurrentParticipant()
	case "space":
		m.toggleCurrentParticipant()
	}

	return m, nil
}

func (m model) updateSpinPopup(key string) (model, tea.Cmd) {
	switch key {
	case "esc":
		m.showSpinPopup = false
		m.spinning = false
		m.spinParticipants = nil
		m.spinPendingWinners = nil
		m.spinVisibleWinners = 0
		m.spinVisibleElim = 0
		m.infoMessage = "Spin popup closed."
		return m, nil
	case "s":
		if m.spinning {
			return m, nil
		}
		return m.startSpinCurrentRoulette()
	default:
		return m, nil
	}
}

func (m *model) toggleCurrentParticipant() {
	if m.focus != focusParticipants {
		m.errorMessage = "Switch to the participants panel to enable/disable someone."
		return
	}

	r := m.currentRoulette()
	participant := m.currentParticipant()
	if r == nil || participant == nil {
		m.errorMessage = "Select a participant to toggle."
		return
	}

	disabled := r.Disabled()
	isDisabled := false
	for _, d := range disabled {
		if d.Name() == participant.Name() {
			isDisabled = true
			break
		}
	}

	if isDisabled {
		r.EnableParticipant(*participant)
		m.infoMessage = fmt.Sprintf("%s is now enabled.", participant.Name())
	} else {
		r.DisableParticipant(*participant)
		m.infoMessage = fmt.Sprintf("%s is now disabled.", participant.Name())
	}

	if m.storage != nil {
		m.storage.Save(m.roulettes)
	}
}

func (m model) updateCreateRoulette(key string) model {
	switch key {
	case "tab":
		m.createMode = nextMode(m.createMode)
		m.infoMessage = fmt.Sprintf("Creation mode: %s", renderModeLabel(m.createMode, 0))
	case "shift+tab", "backtab":
		m.createMode = prevMode(m.createMode)
		m.infoMessage = fmt.Sprintf("Creation mode: %s", renderModeLabel(m.createMode, 0))
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

		if m.createMode == ModeMultiWinner {
			m.createName = name
			m.mode = modeCreateMultiWinnerCount
			m.input = ""
			m.infoMessage = "Type the number of winners and press enter."
			return m
		}

		opts := []Option{}
		m.roulettes = append(m.roulettes, NewRoulette(name, m.createMode, opts...))
		m.selectedRoulette = len(m.roulettes) - 1
		m.refreshPlanColor()
		m.selectedParticipant = 0
		m.focus = focusParticipants
		m.mode = modeAddParticipant
		m.input = ""
		m.createName = ""
		m.createMode = ModeRepeatableWinners
		m.createMultiCount = 0
		m.infoMessage = fmt.Sprintf("Roulette %s created with mode %s. Add participants and press esc when done.", name, renderModeLabel(m.roulettes[m.selectedRoulette].Mode(), m.roulettes[m.selectedRoulette].MultiWinnerCounter()))
		if m.storage != nil {
			m.storage.Save(m.roulettes)
		}
	case "backspace":
		m.input = deleteLastRune(m.input)
	case "space":
		m.input += " "
	default:
		m.input = appendInput(m.input, key)
	}

	return m
}

func (m model) updateCreateMultiWinnerCount(key string) model {
	switch key {
	case "esc":
		m.mode = modeCreateRoulette
		m.input = m.createName
		m.infoMessage = "Type a roulette name and press enter."
	case "enter":
		count, err := strconv.Atoi(strings.TrimSpace(m.input))
		if err != nil || count <= 0 {
			m.errorMessage = "Winner count must be a positive number."
			return m
		}

		m.createMultiCount = count
		m.roulettes = append(m.roulettes, NewRoulette(m.createName, ModeMultiWinner, WithMultiWinnerCounter(m.createMultiCount)))
		m.selectedRoulette = len(m.roulettes) - 1
		m.refreshPlanColor()
		m.selectedParticipant = 0
		m.focus = focusParticipants
		m.mode = modeAddParticipant
		m.input = ""
		m.infoMessage = fmt.Sprintf("Roulette %s created with mode %s. Add participants and press esc when done.", m.createName, renderModeLabel(m.roulettes[m.selectedRoulette].Mode(), m.roulettes[m.selectedRoulette].MultiWinnerCounter()))
		m.createName = ""
		m.createMode = ModeRepeatableWinners
		m.createMultiCount = 0
	case "backspace":
		m.input = deleteLastRune(m.input)
	default:
		if len([]rune(key)) == 1 && key >= "0" && key <= "9" {
			m.input += key
		}
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
		if m.storage != nil {
			m.storage.Save(m.roulettes)
		}
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
	m.refreshPlanColor()
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
	if m.storage != nil {
		m.storage.Save(m.roulettes)
	}
}

func (m *model) resetCurrentRoulette() {
	r := m.currentRoulette()
	if r == nil {
		m.errorMessage = "Select a roulette to reset."
		return
	}

	r.Reset()
	delete(m.lastWinners, m.selectedRoulette)
	m.infoMessage = fmt.Sprintf("Roulette %s was reset.", r.Name())
	if m.storage != nil {
		m.storage.Save(m.roulettes)
	}
}

func nextMode(current Mode) Mode {
	modes := []Mode{ModeRepeatableWinners, ModeNoRepeatWinners, ModeElimination, ModeMultiWinner}
	for i, mode := range modes {
		if mode == current {
			return modes[(i+1)%len(modes)]
		}
	}

	return ModeRepeatableWinners
}

func prevMode(current Mode) Mode {
	modes := []Mode{ModeRepeatableWinners, ModeNoRepeatWinners, ModeElimination, ModeMultiWinner}
	for i, mode := range modes {
		if mode == current {
			prev := i - 1
			if prev < 0 {
				prev = len(modes) - 1
			}
			return modes[prev]
		}
	}

	return ModeRepeatableWinners
}

func renderModeLabel(mode Mode, multiCount int) string {
	switch mode {
	case ModeRepeatableWinners:
		return "repeatable"
	case ModeNoRepeatWinners:
		return "no-repeat"
	case ModeElimination:
		return "elimination"
	case ModeMultiWinner:
		if multiCount < 2 {
			multiCount = 2
		}
		return fmt.Sprintf("multi-winner (%d)", multiCount)
	default:
		return string(mode)
	}
}

func (m model) startSpinCurrentRoulette() (model, tea.Cmd) {
	r := m.currentRoulette()
	if r == nil {
		m.errorMessage = "Select or create a roulette before spinning."
		return m, nil
	}

	participantsBeforeSpin := append([]Participant(nil), r.Candidates()...)
	if len(participantsBeforeSpin) == 0 {
		m.errorMessage = "cannot spin roulette without participants"
		return m, nil
	}

	if r.Mode() == ModeMultiWinner {
		r.Reset()
		delete(m.lastWinners, m.selectedRoulette)
	}

	previousWinnerCount := len(r.Winners())
	previousEliminatedCount := len(r.Eliminated())

	if err := r.Spin(); err != nil {
		if len(r.Winners()) == previousWinnerCount && len(r.Eliminated()) == previousEliminatedCount {
			m.errorMessage = err.Error()
			return m, nil
		}

		m.infoMessage = err.Error()
	}

	participants := participantsBeforeSpin
	resultName := ""
	resultIsWinner := false
	var pendingWinners []string

	// Winner was produced by this spin.
	winners := r.Winners()
	if len(winners) > previousWinnerCount {
		resultIsWinner = true
		if r.Mode() == ModeMultiWinner {
			// Queue all winners for sequential animation.
			for _, w := range winners[previousWinnerCount:] {
				pendingWinners = append(pendingWinners, w.Name())
			}
			resultName = pendingWinners[0]
			pendingWinners = pendingWinners[1:]
		} else {
			resultName = winners[len(winners)-1].Name()
		}
	}

	// Elimination mode can produce an eliminated participant instead of a winner.
	if resultName == "" && r.Mode() == ModeElimination {
		eliminated := r.Eliminated()
		if len(eliminated) > previousEliminatedCount {
			resultName = eliminated[len(eliminated)-1].Name()
		}
	}

	if resultName == "" {
		m.errorMessage = "spin completed but no result was recorded"
		return m, nil
	}

	winnerIndex := -1
	for i, p := range participants {
		if p.Name() == resultName {
			winnerIndex = i
			break
		}
	}

	if winnerIndex < 0 {
		m.errorMessage = "spin result not found in participants"
		return m, nil
	}

	startIndex := rand.IntN(len(participants))
	spins := rand.IntN(3) + 3
	totalSteps := spins*len(participants) + ((winnerIndex - startIndex + len(participants)) % len(participants))
	if totalSteps == 0 {
		totalSteps = len(participants)
	}

	m.spinning = true
	m.spinToken++
	m.spinRouletteIndex = m.selectedRoulette
	m.spinWinnerName = resultName
	m.spinWinnerSlice = winnerIndex
	m.spinPendingWinners = pendingWinners
	m.spinParticipants = participantsBeforeSpin
	m.spinCurrentSlice = startIndex
	m.spinTotalSteps = totalSteps
	m.spinStep = 0
	m.spinVisibleWinners = previousWinnerCount
	m.spinVisibleElim = previousEliminatedCount
	m.spinOutcomeWinner = resultIsWinner
	m.infoMessage = fmt.Sprintf("Spinning %s...", r.Name())

	return m, spinTickCmd(m.spinToken, spinDelay(0, totalSteps))
}

func (m model) advanceSpin() (model, tea.Cmd) {
	if m.spinRouletteIndex < 0 || m.spinRouletteIndex >= len(m.roulettes) {
		m.spinning = false
		m.spinParticipants = nil
		return m, nil
	}

	participants := m.spinParticipants
	if len(participants) == 0 {
		participants = m.roulettes[m.spinRouletteIndex].Candidates()
	}
	if len(participants) == 0 {
		m.spinning = false
		m.spinParticipants = nil
		return m, nil
	}

	m.spinCurrentSlice = (m.spinCurrentSlice + 1) % len(participants)
	m.spinStep++

	if m.spinStep >= m.spinTotalSteps {
		m.spinCurrentSlice = m.spinWinnerSlice
		m.lastWinners[m.spinRouletteIndex] = m.spinWinnerName

		// If there are more winners to animate, chain the next spin animation.
		if m.spinOutcomeWinner && len(m.spinPendingWinners) > 0 {
			nextName := m.spinPendingWinners[0]
			m.spinPendingWinners = m.spinPendingWinners[1:]

			nextIndex := -1
			for i, p := range participants {
				if p.Name() == nextName {
					nextIndex = i
					break
				}
			}

			if nextIndex < 0 {
				// Fallback: just finish if winner index not found.
				m.spinPendingWinners = nil
			} else {
				spins := rand.IntN(3) + 3
				totalSteps := spins*len(participants) + ((nextIndex - m.spinCurrentSlice + len(participants)) % len(participants))
				if totalSteps == 0 {
					totalSteps = len(participants)
				}
				m.spinWinnerName = nextName
				m.spinWinnerSlice = nextIndex
				m.spinTotalSteps = totalSteps
				m.spinStep = 0
				m.infoMessage = fmt.Sprintf("Spinning %s...", m.roulettes[m.spinRouletteIndex].Name())
				return m, spinTickCmd(m.spinToken, spinDelay(0, totalSteps))
			}
		}

		m.spinning = false
		m.spinParticipants = nil
		m.spinPendingWinners = nil
		m.spinVisibleWinners = 0
		m.spinVisibleElim = 0
		if m.spinOutcomeWinner {
			r := m.roulettes[m.spinRouletteIndex]
			newWinners := r.Winners()
			if len(newWinners) > 1 {
				names := make([]string, len(newWinners))
				for i, w := range newWinners {
					names[i] = w.Name()
				}
				m.infoMessage = fmt.Sprintf("🎉 %s won in %s", strings.Join(names, ", "), r.Name())
			} else {
				m.infoMessage = fmt.Sprintf("🎉 %s won in %s", m.spinWinnerName, r.Name())
			}
		} else {
			delete(m.lastWinners, m.spinRouletteIndex)
			m.infoMessage = fmt.Sprintf("✕ %s was eliminated in %s", m.spinWinnerName, m.roulettes[m.spinRouletteIndex].Name())
		}
		return m, nil
	}

	return m, spinTickCmd(m.spinToken, spinDelay(m.spinStep, m.spinTotalSteps))
}

func (m model) renderWheel() string {
	return m.renderWheelWithCanvas(64, 40)
}

func (m model) renderWheelWithCanvas(canvasWidth int, canvasHeight int) string {
	r := m.currentRoulette()
	if r == nil {
		return paint("Create a roulette to display the wheel.", ansiFgMuted)
	}

	participants := r.Candidates()
	if m.spinning && m.spinRouletteIndex == m.selectedRoulette && len(m.spinParticipants) > 0 {
		participants = m.spinParticipants
	}
	n := len(participants)

	type subPixel struct {
		on    bool
		color string
	}

	canvas := make([][]subPixel, canvasHeight)
	cx := float64(canvasWidth-1) / 2
	cy := float64(canvasHeight-1) / 2
	rx := float64(canvasWidth) * 0.45
	ry := float64(canvasHeight) * 0.45

	for y := 0; y < canvasHeight; y++ {
		canvas[y] = make([]subPixel, canvasWidth)
		for x := 0; x < canvasWidth; x++ {
			dx := (float64(x) - cx) / rx
			dy := (float64(y) - cy) / ry
			d2 := dx*dx + dy*dy

			if n == 0 {
				if d2 >= 0.92 && d2 <= 1.05 {
					canvas[y][x] = subPixel{on: true, color: ansiFgMuted}
				}
				continue
			}

			if d2 > 1 {
				continue
			}

			angle := math.Atan2(dy, dx)
			if angle < 0 {
				angle += 2 * math.Pi
			}

			idx := int(math.Floor(angle / (2 * math.Pi / float64(n))))
			if idx >= n {
				idx = n - 1
			}

			winner := m.lastWinners[m.selectedRoulette]
			isWinnerSlice := winner != "" && participants[idx].Name() == winner
			isSpinSlice := m.spinning && m.spinRouletteIndex == m.selectedRoulette && idx == m.spinCurrentSlice

			canvas[y][x] = subPixel{on: true, color: participantFgColor(idx, n, isWinnerSlice || isSpinSlice)}
		}
	}

	var out strings.Builder
	for y := 0; y < canvasHeight; y += 4 {
		for x := 0; x < canvasWidth; x += 2 {
			mask := 0
			colorHits := map[string]int{}

			for py := 0; py < 4; py++ {
				for px := 0; px < 2; px++ {
					sy := y + py
					sx := x + px
					if sy >= canvasHeight || sx >= canvasWidth {
						continue
					}

					s := canvas[sy][sx]
					if !s.on {
						continue
					}

					mask |= brailleDotMask(px, py)
					if s.color != "" {
						colorHits[s.color]++
					}
				}
			}

			if mask == 0 {
				out.WriteString(" ")
				continue
			}

			glyph := string(rune(0x2800 + mask))
			out.WriteString(paint(glyph, dominantColor(colorHits)))
		}

		if y < canvasHeight-4 {
			out.WriteString("\n")
		}
	}

	return centerLines(out.String(), minPanelInnerWidth-2)
}

func (m model) renderSpinPopup() string {
	r := m.currentRoulette()
	if r == nil {
		return ""
	}

	body := m.renderWheelWithCanvas(92, 56)
	title := fmt.Sprintf("SPIN ROULETTE • %s", r.Name())
	return panel(title, body)
}

func (m model) renderTopBar() string {
	modeBar := m.renderModeChips()
	plan := m.renderAgentPlan()

	return modeBar + strings.Repeat(" ", 6) + plan
}

func (m model) renderModeChips() string {
	modes := []Mode{ModeRepeatableWinners, ModeNoRepeatWinners, ModeElimination, ModeMultiWinner}
	parts := make([]string, 0, len(modes)+1)
	parts = append(parts, paint("MODE", ansiFgMuted, ansiBold))

	active := m.createMode
	if m.mode == modeBrowse {
		r := m.currentRoulette()
		if r != nil {
			active = r.Mode()
		}
	}

	for _, mode := range modes {
		chipText := strings.ToUpper(renderModeLabel(mode, m.createMultiCount))
		chipColor := modeChipColor(mode)
		if mode == active {
			parts = append(parts, paint(" "+truncateLabel(chipText, 18)+" ", ansiBold, ansiBgSurface2, chipColor))
		} else {
			parts = append(parts, paint(" "+truncateLabel(chipText, 18)+" ", ansiDim, chipColor))
		}
	}

	return strings.Join(parts, " ")
}

func (m model) renderAgentPlan() string {
	r := m.currentRoulette()
	if r == nil {
		return paint("ROULETTE: (none)", ansiFgMuted)
	}

	color := m.planColor
	if color == "" {
		color = ansiFgAccent2
	}
	return paint("ROULETTE: ", ansiFgMuted) + paint(r.Name(), ansiBold, color)
}

func (m model) renderRouletteLogo() string {
	logo := []string{
		"██████╗  ██████╗ ██╗   ██╗██╗     ███████╗████████╗████████╗███████╗",
		"██╔══██╗██╔═══██╗██║   ██║██║     ██╔════╝╚══██╔══╝╚══██╔══╝██╔════╝",
		"██████╔╝██║   ██║██║   ██║██║     █████╗     ██║      ██║   █████╗  ",
		"██╔══██╗██║   ██║██║   ██║██║     ██╔══╝     ██║      ██║   ██╔══╝  ",
		"██║  ██║╚██████╔╝╚██████╔╝███████╗███████╗   ██║      ██║   ███████╗",
		"╚═╝  ╚═╝ ╚═════╝  ╚═════╝ ╚══════╝╚══════╝   ╚═╝      ╚═╝   ╚══════╝",
	}

	for i := range logo {
		logo[i] = paint(logo[i], ansiFgAccent, ansiBold)
	}

	return strings.Join(logo, "\n")
}

func (m model) renderDashboard() string {
	var s strings.Builder
	s.WriteString(panel("ROULETTES", m.renderRoulettes()))
	s.WriteString("\n")
	s.WriteString(panel("PARTICIPANTS", m.renderParticipants()))
	s.WriteString("\n")
	s.WriteString(panel("WINNERS", m.renderWinners()))

	r := m.currentRoulette()
	if r != nil && r.Mode() == ModeElimination {
		s.WriteString("\n")
		s.WriteString(panel("ELIMINATED", m.renderEliminated()))
	}

	return s.String()
}

func (m model) ensurePlanColor() model {
	r := m.currentRoulette()
	if r == nil {
		m.planRouletteID = ""
		if m.planColor == "" {
			m.planColor = randomPlanColor()
		}
		return m
	}

	if m.planRouletteID != r.ID() || m.planColor == "" {
		m.planRouletteID = r.ID()
		m.planColor = randomPlanColor()
	}

	return m
}

func (m *model) refreshPlanColor() {
	r := m.currentRoulette()
	if r == nil {
		m.planRouletteID = ""
		m.planColor = randomPlanColor()
		return
	}

	if m.planRouletteID != r.ID() {
		m.planRouletteID = r.ID()
		m.planColor = randomPlanColor()
	}
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
		modeText := renderModeLabel(roulette.Mode(), roulette.MultiWinnerCounter())
		total := len(roulette.Participants())
		candidates := len(roulette.Candidates())
		countText := fmt.Sprintf("(%d participants)", total)
		if candidates < total {
			countText = fmt.Sprintf("(%d/%d candidates)", candidates, total)
		}
		fmt.Fprintf(
			&s,
			"%s %s %s %s\n",
			marker,
			paint(roulette.Name(), ansiBold),
			paint(fmt.Sprintf("[%s]", modeText), ansiFgAccent2),
			paint(countText, ansiFgMuted),
		)
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

	disabled := r.Disabled()
	isDisabled := func(p Participant) bool {
		for _, d := range disabled {
			if d.Name() == p.Name() {
				return true
			}
		}
		return false
	}

	for index, participant := range r.Participants() {
		marker := m.marker(m.focus == focusParticipants, m.selectedParticipant == index)
		if isDisabled(participant) {
			fmt.Fprintf(&s, "%s %s %s\n", marker, paint(participant.Name(), ansiDim, ansiFgMuted), paint("[disabled]", ansiDim, ansiFgMuted))
		} else {
			fmt.Fprintf(&s, "%s %s\n", marker, participant.Name())
		}
	}

	return s.String()
}

func (m model) renderWinners() string {
	var s strings.Builder
	r := m.currentRoulette()
	if r == nil {
		s.WriteString(paint("Create a roulette, add participants, then spin.", ansiFgMuted))
		s.WriteString("\n")
		return s.String()
	}

	winners := r.Winners()
	if m.spinning && m.spinRouletteIndex == m.selectedRoulette && m.spinVisibleWinners >= 0 && m.spinVisibleWinners < len(winners) {
		winners = winners[:m.spinVisibleWinners]
	}
	if len(winners) == 0 {
		s.WriteString(paint("(no winners yet — press s to spin)", ansiFgMuted, ansiDim))
		s.WriteString("\n")
		return s.String()
	}

	for index, winner := range winners {
		fmt.Fprintf(&s, "%s %s\n", paint("★", ansiFgSuccess), winner.Name())
		if index >= 4 {
			remaining := len(winners) - (index + 1)
			if remaining > 0 {
				fmt.Fprintf(&s, "%s\n", paint(fmt.Sprintf("... and %d more", remaining), ansiFgMuted, ansiDim))
			}
			break
		}
	}

	return s.String()
}

func (m model) renderEliminated() string {
	var s strings.Builder
	r := m.currentRoulette()
	if r == nil || r.Mode() != ModeElimination {
		s.WriteString(paint("Eliminated list is available only in elimination mode.", ansiFgMuted))
		s.WriteString("\n")
		return s.String()
	}

	eliminated := r.Eliminated()
	if m.spinning && m.spinRouletteIndex == m.selectedRoulette && m.spinVisibleElim >= 0 && m.spinVisibleElim < len(eliminated) {
		eliminated = eliminated[:m.spinVisibleElim]
	}

	if len(eliminated) == 0 {
		s.WriteString(paint("(none yet — press s to eliminate one participant)", ansiFgMuted, ansiDim))
		s.WriteString("\n")
		return s.String()
	}

	for index, participant := range eliminated {
		fmt.Fprintf(&s, "%s %s\n", paint("✕", ansiFgDanger), participant.Name())
		if index >= 4 {
			remaining := len(eliminated) - (index + 1)
			if remaining > 0 {
				fmt.Fprintf(&s, "%s\n", paint(fmt.Sprintf("... and %d more", remaining), ansiFgMuted, ansiDim))
			}
			break
		}
	}

	return s.String()
}

func (m model) renderPrompt() string {
	switch m.mode {
	case modeCreateRoulette:
		modeText := renderModeLabel(m.createMode, 0)
		return fmt.Sprintf(
			"%s %s %s %s",
			paint("New roulette name:", ansiFgAccent),
			paint(fmt.Sprintf("%s_", m.input), ansiBgSurface, ansiFgBase),
			paint("mode:", ansiFgMuted),
			paint(modeText, ansiFgAccent2, ansiBold),
		)
	case modeCreateMultiWinnerCount:
		return fmt.Sprintf(
			"%s %s %s %s",
			paint("Roulette:", ansiFgMuted),
			paint(m.createName, ansiBold),
			paint("winners:", ansiFgAccent),
			paint(fmt.Sprintf("%s_", m.input), ansiBgSurface, ansiFgBase),
		)
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
		return "type name • tab/shift+tab cycle mode • enter continue • esc cancel • q quit"
	case modeCreateMultiWinnerCount:
		return "type winners count • enter create roulette • esc back • q quit"
	case modeAddParticipant:
		return "enter add participant • esc finish • q quit"
	default:
		if m.spinning {
			return "spinning... please wait • q quit"
		}
		if m.showSpinPopup {
			return "s spin again • esc close popup • q quit"
		}
		return "n new roulette • a add participant • d remove participant • space enable/disable participant • s spin roulette • r reset winners • tab/←/→ switch panel • ↑/↓ move • q quit"
	}
}

func modeChipColor(mode Mode) string {
	switch mode {
	case ModeRepeatableWinners:
		return "\033[38;2;122;214;163m"
	case ModeNoRepeatWinners:
		return "\033[38;2;122;162;255m"
	case ModeElimination:
		return "\033[38;2;255;107;107m"
	case ModeMultiWinner:
		return "\033[38;2;255;214;170m"
	default:
		return ansiFgMuted
	}
}

func randomPlanColor() string {
	h := rand.Float64() * 360.0
	r, g, b := hsvToRGB(h, 0.60, 0.95)
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
}

type spinTickMsg struct {
	token int
}

func spinTickCmd(token int, delay time.Duration) tea.Cmd {
	return tea.Tick(delay, func(time.Time) tea.Msg {
		return spinTickMsg{token: token}
	})
}

func spinDelay(step int, total int) time.Duration {
	if total <= 1 {
		return 220 * time.Millisecond
	}

	const minDelayMs = 30
	const maxDelayMs = 220

	ratio := float64(step) / float64(total-1)
	eased := ratio * ratio
	delayMs := minDelayMs + int(float64(maxDelayMs-minDelayMs)*eased)
	return time.Duration(delayMs) * time.Millisecond
}

func brailleDotMask(px int, py int) int {
	switch {
	case px == 0 && py == 0:
		return 1 << 0 // dot 1
	case px == 0 && py == 1:
		return 1 << 1 // dot 2
	case px == 0 && py == 2:
		return 1 << 2 // dot 3
	case px == 1 && py == 0:
		return 1 << 3 // dot 4
	case px == 1 && py == 1:
		return 1 << 4 // dot 5
	case px == 1 && py == 2:
		return 1 << 5 // dot 6
	case px == 0 && py == 3:
		return 1 << 6 // dot 7
	case px == 1 && py == 3:
		return 1 << 7 // dot 8
	default:
		return 0
	}
}

func dominantColor(hits map[string]int) string {
	best := ""
	bestCount := -1
	for color, count := range hits {
		if count > bestCount {
			best = color
			bestCount = count
		}
	}

	if best == "" {
		return ansiFgMuted
	}

	return best
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

	innerWidth := minPanelInnerWidth
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

func centerLines(content string, targetWidth int) string {
	if targetWidth <= 0 {
		return content
	}

	lines := strings.Split(content, "\n")
	var out strings.Builder
	for i, line := range lines {
		lineWidth := visibleWidth(line)
		leftPad := max(0, (targetWidth-lineWidth)/2)
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

func wrapText(content string, maxWidth int) string {
	if maxWidth <= 0 || visibleWidth(content) <= maxWidth {
		return content
	}

	words := strings.Fields(content)
	if len(words) == 0 {
		return content
	}

	var lines []string
	line := words[0]

	for _, w := range words[1:] {
		candidate := line + " " + w
		if visibleWidth(candidate) <= maxWidth {
			line = candidate
			continue
		}
		lines = append(lines, line)
		line = w
	}

	lines = append(lines, line)
	return strings.Join(lines, "\n")
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

func participantFgColor(index int, total int, bright bool) string {
	if total <= 0 {
		return ansiFgMuted
	}

	h := (360.0 / float64(total)) * float64(index)
	s := 0.72
	v := 0.78
	if bright {
		v = 0.98
	}

	r, g, b := hsvToRGB(h, s, v)
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
}

func hsvToRGB(h float64, s float64, v float64) (int, int, int) {
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := v - c

	var rf float64
	var gf float64
	var bf float64

	switch {
	case h < 60:
		rf, gf, bf = c, x, 0
	case h < 120:
		rf, gf, bf = x, c, 0
	case h < 180:
		rf, gf, bf = 0, c, x
	case h < 240:
		rf, gf, bf = 0, x, c
	case h < 300:
		rf, gf, bf = x, 0, c
	default:
		rf, gf, bf = c, 0, x
	}

	r := int(math.Round((rf + m) * 255))
	g := int(math.Round((gf + m) * 255))
	b := int(math.Round((bf + m) * 255))

	return r, g, b
}

func truncateLabel(value string, maxLen int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= maxLen {
		return string(runes)
	}

	if maxLen <= 1 {
		return string(runes[:maxLen])
	}

	return string(runes[:maxLen-1]) + "…"
}
