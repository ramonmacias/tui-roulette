package internal

import (
	"errors"
	"math/rand/v2"
	"slices"

	"github.com/google/uuid"
)

// Mode defines the spinning behavior and winner selection strategy for a roulette.
type Mode string

const (
	// ModeRepeatableWinners allows the same participant to win multiple times.
	ModeRepeatableWinners Mode = "REPEATABLE_WINNERS"
	// ModeNoRepeatWinners ensures each participant can only win once.
	ModeNoRepeatWinners Mode = "NO_REPEAT_WINNERS"
	// ModeElimination removes participants one by one until only one remains.
	ModeElimination Mode = "ELIMINATION"
	// ModeMultiWinner selects multiple winners in a single spin.
	ModeMultiWinner Mode = "MULTI_WINNER"
)

// Option is a functional option for configuring a Roulette during creation.
type Option func(r *Roulette)

// WithMultiWinnerCounter sets the number of winners to select in MULTI_WINNER mode.
func WithMultiWinnerCounter(count int) Option {
	return func(r *Roulette) {
		r.multiWinnerCounter = count
	}
}

// Roulette stores the participants and winners for a raffle draw.
type Roulette struct {
	id           uuid.UUID
	name         string
	mode         Mode
	participants []Participant
	winners      []Participant
	// only used for the elimination mode to keep track on who is already
	// being eliminated
	eliminated []Participant
	// only used on the multi winner mode to know how many winners we need to pick
	// while spinning the roulette
	multiWinnerCounter int
}

// NewRoulette creates an empty roulette with a generated identifier.
func NewRoulette(name string, mode Mode, opts ...Option) *Roulette {
	r := &Roulette{
		id:           uuid.New(),
		name:         name,
		mode:         mode,
		participants: []Participant{},
		winners:      []Participant{},
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

func existingParticipant(p Participant) func(existing Participant) bool {
	return func(existing Participant) bool {
		return existing.name == p.name
	}
}

// Name returns the roulette display name.
func (r *Roulette) Name() string {
	return r.name
}

// ID returns the roulette unique identifier.
func (r *Roulette) ID() string {
	return r.id.String()
}

// Mode returns the configured spinning mode for the roulette.
func (r *Roulette) Mode() Mode {
	return r.mode
}

// MultiWinnerCounter returns the configured number of winners for MULTI_WINNER mode.
func (r *Roulette) MultiWinnerCounter() int {
	return r.multiWinnerCounter
}

// AddParticipant adds a participant to the roulette if no participant with the same name exists yet.
func (r *Roulette) AddParticipant(p Participant) error {
	if slices.ContainsFunc(r.participants, existingParticipant(p)) {
		return errors.New("the given participant is already added into the roulette participants list")
	}

	r.participants = append(r.participants, p)
	return nil
}

// RemoveParticipant removes the participant with the same name from the roulette.
func (r *Roulette) RemoveParticipant(p Participant) error {
	participantIndex := slices.IndexFunc(r.participants, existingParticipant(p))
	if participantIndex < 0 {
		return errors.New("the given participant is not part of the roulette participants list")
	}

	r.participants = slices.Delete(r.participants, participantIndex, participantIndex+1)
	return nil
}

// Participants returns the current list of participants in the roulette.
// The list will be adapted based on the roulette mode.
func (r *Roulette) Participants() []Participant {
	switch r.mode {
	case ModeNoRepeatWinners:
		return r.filterCandidates(r.winners)
	case ModeRepeatableWinners:
		return r.participants
	case ModeElimination:
		return r.filterCandidates(r.eliminated)
	case ModeMultiWinner:
		return r.participants
	default:
		return r.participants
	}
}

// Winners returns the current list of participants that won during that roulette spins.
func (r *Roulette) Winners() []Participant {
	return r.winners
}

// Eliminated returns the current list of participants that were already eliminated after
// spinning the roulette on the elimination mode.
func (r *Roulette) Eliminated() []Participant {
	return r.eliminated
}

// Reset clears all winners and eliminated participants from the roulette.
func (r *Roulette) Reset() {
	r.winners = []Participant{}
	r.eliminated = []Participant{}
}

// Spin randomly selects one or more participants based on the roulette mode and records them as winners.
func (r *Roulette) Spin() error {
	if len(r.participants) == 0 {
		return errors.New("cannot spin roulette without participants")
	}

	switch r.mode {
	case ModeNoRepeatWinners:
		return r.spinModeNoRepeatWinners()
	case ModeRepeatableWinners:
		return r.spinModeRepeatableWinners()
	case ModeElimination:
		return r.spinModeElimination()
	case ModeMultiWinner:
		return r.spinModeMultiWinner()
	default:
		return errors.New("roulette mode not allowed")
	}
}

// spinModeNoRepeatWinners selects a random participant who has never won before.
func (r *Roulette) spinModeNoRepeatWinners() error {
	candidates := r.filterCandidates(r.winners)
	if len(candidates) == 0 {
		return errors.New("no more winners left")
	}

	winner := r.pickRandomParticipant(candidates)
	r.winners = append(r.winners, winner)
	return nil
}

// spinModeRepeatableWinners selects a random participant allowing the same person to win multiple times.
func (r *Roulette) spinModeRepeatableWinners() error {
	winner := r.pickRandomParticipant(r.participants)
	r.winners = append(r.winners, winner)
	return nil
}

// spinModeElimination progressively eliminates participants until only one remains as the final winner.
// Subsequent spins after a winner is determined will return an error.
func (r *Roulette) spinModeElimination() error {
	candidates := r.filterCandidates(r.eliminated)
	if len(candidates) == 0 {
		return errors.New("all participants have been eliminated")
	}

	if len(r.winners) > 0 {
		return errors.New("we have already a winner")
	}

	if len(candidates) == 1 {
		r.winners = append(r.winners, candidates[0])
		return errors.New("we have already a winner")
	}

	eliminated := r.pickRandomParticipant(candidates)
	r.eliminated = append(r.eliminated, eliminated)

	return nil
}

// spinModeMultiWinner selects multiple winners in a single spin based on the configured counter.
func (r *Roulette) spinModeMultiWinner() error {
	if r.multiWinnerCounter == 0 {
		return errors.New("we need to define the number of winners")
	}
	if r.multiWinnerCounter >= len(r.participants) {
		return errors.New("we cannot config a multi winner counter equal or bigger than the number of participants")
	}

	candidates := append([]Participant(nil), r.participants...)
	for i := 0; i < r.multiWinnerCounter; i++ {
		winner := r.pickRandomParticipant(candidates)
		r.winners = append(r.winners, winner)
		candidates = slices.DeleteFunc(candidates, existingParticipant(winner))
	}
	return nil
}

// pickRandomParticipant returns a random participant from the candidates list.
func (r *Roulette) pickRandomParticipant(candidates []Participant) Participant {
	idx := rand.IntN(len(candidates))
	return candidates[idx]
}

// filterCandidates returns participants not in the exclusion list.
func (r *Roulette) filterCandidates(exclude []Participant) []Participant {
	var candidates []Participant
	for _, p := range r.participants {
		if !slices.ContainsFunc(exclude, existingParticipant(p)) {
			candidates = append(candidates, p)
		}
	}
	return candidates
}

// Participant represents a person that can be added to a roulette.
type Participant struct {
	id   uuid.UUID
	name string
}

// Name returns the participant display name.
func (p *Participant) Name() string {
	return p.name
}

// NewParticipant creates a participant with the provided name and a generated identifier.
func NewParticipant(name string) Participant {
	return Participant{
		id:   uuid.New(),
		name: name,
	}
}
