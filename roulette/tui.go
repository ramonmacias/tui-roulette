package roulette

import (
	"fmt"
	"math"
	"math/rand/v2"
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
	modeAddParticipant
)

type model struct {
	roulettes           []*Roulette
	selectedRoulette    int
	selectedParticipant int
	lastWinners         map[int]string
	spinning            bool
	spinToken           int
	spinRouletteIndex   int
	spinWinnerName      string
	spinWinnerSlice     int
	spinCurrentSlice    int
	spinTotalSteps      int
	spinStep            int
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
		lastWinners:         map[int]string{},
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

		switch m.mode {
		case modeCreateRoulette:
			m = m.updateCreateRoulette(key)
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
	s.WriteString(panel("ROULETTE", m.renderWheel()))
	s.WriteString("\n")
	s.WriteString(panel("WINNERS", m.renderWinners()))
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
		return m.startSpinCurrentRoulette()
	case "d", "x", "backspace":
		m.removeCurrentParticipant()
	}

	return m, nil
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

func (m model) startSpinCurrentRoulette() (model, tea.Cmd) {
	r := m.currentRoulette()
	if r == nil {
		m.errorMessage = "Select or create a roulette before spinning."
		return m, nil
	}

	winner, err := r.Spin()
	if err != nil {
		m.errorMessage = err.Error()
		return m, nil
	}

	participants := r.Participants()
	winnerIndex := -1
	for i, p := range participants {
		if p.Name() == winner.Name() {
			winnerIndex = i
			break
		}
	}

	if winnerIndex < 0 {
		m.errorMessage = "winner not found in participants"
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
	m.spinWinnerName = winner.Name()
	m.spinWinnerSlice = winnerIndex
	m.spinCurrentSlice = startIndex
	m.spinTotalSteps = totalSteps
	m.spinStep = 0
	m.infoMessage = fmt.Sprintf("Spinning %s...", r.Name())

	return m, spinTickCmd(m.spinToken, spinDelay(0, totalSteps))
}

func (m model) advanceSpin() (model, tea.Cmd) {
	if m.spinRouletteIndex < 0 || m.spinRouletteIndex >= len(m.roulettes) {
		m.spinning = false
		return m, nil
	}

	participants := m.roulettes[m.spinRouletteIndex].Participants()
	if len(participants) == 0 {
		m.spinning = false
		return m, nil
	}

	m.spinCurrentSlice = (m.spinCurrentSlice + 1) % len(participants)
	m.spinStep++

	if m.spinStep >= m.spinTotalSteps {
		m.spinning = false
		m.spinCurrentSlice = m.spinWinnerSlice
		m.lastWinners[m.spinRouletteIndex] = m.spinWinnerName
		m.infoMessage = fmt.Sprintf("🎉 %s won in %s", m.spinWinnerName, m.roulettes[m.spinRouletteIndex].Name())
		return m, nil
	}

	return m, spinTickCmd(m.spinToken, spinDelay(m.spinStep, m.spinTotalSteps))
}

func (m model) renderWheel() string {
	r := m.currentRoulette()
	if r == nil {
		return paint("Create a roulette to display the wheel.", ansiFgMuted)
	}

	participants := r.Participants()
	n := len(participants)

	const canvasWidth = 64  // braille sub-pixels (2 per text cell)
	const canvasHeight = 40 // braille sub-pixels (4 per text row)

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

func (m model) renderWinners() string {
	var s strings.Builder
	r := m.currentRoulette()
	if r == nil {
		s.WriteString(paint("Create a roulette, add participants, then spin.", ansiFgMuted))
		s.WriteString("\n")
		return s.String()
	}

	winners := r.Winners()
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
		if m.spinning {
			return "spinning... please wait • q quit"
		}
		return "n new roulette • a add participant • d remove participant • s spin roulette • tab/←/→ switch panel • ↑/↓ move • q quit"
	}
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
