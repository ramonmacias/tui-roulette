package roulette

import (
	"errors"
	"math/rand/v2"
	"slices"

	"github.com/google/uuid"
)

type Mode string

const (
	ModeRepeatableWinners Mode = "REPEATABLE_WINNERS"
	ModeNoRepeatWinners   Mode = "NO_REPEAT_WINNERS"
	ModeElimination       Mode = "ELIMINATION"
	ModeMultiWinner       Mode = "MULTI_WINNER"
)

type Option func(r *Roulette)

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
func (r *Roulette) Participants() []Participant {
	return r.participants
}

// Winners returns the current list of participants that won during that roulette spins.
func (r *Roulette) Winners() []Participant {
	return r.winners
}

// Spin randomly selects one participant and records that participant as a winner.
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

func (r *Roulette) spinModeMultiWinner() error {
	if r.multiWinnerCounter == 0 {
		return errors.New("we need to define the number of winners")
	}
	if r.multiWinnerCounter >= len(r.participants) {
		return errors.New("we cannot config a multi winner counter equal or bigger than the number of participants")
	}

	candidates := r.participants
	for i := 0; i < r.multiWinnerCounter; i++ {
		i := rand.IntN(len(candidates))
		winner := candidates[i]
		r.winners = append(r.winners, winner)
		candidates = slices.DeleteFunc(candidates, existingParticipant(winner))
	}
	return nil
}

func (r *Roulette) spinModeElimination() error {
	candidates := []Participant{}
	if len(r.eliminated) == 0 {
		candidates = r.participants
	} else {
		for index := range r.participants {
			if !slices.ContainsFunc(r.eliminated, existingParticipant(r.participants[index])) {
				candidates = append(candidates, r.participants[index])
			}
		}
	}

	if len(r.participants) == 1 {
		r.winners = append(r.winners, r.participants[0])
	}

	if len(r.winners) > 0 {
		return errors.New("we have already a winner")
	}

	i := rand.IntN(len(candidates))
	eliminated := candidates[i]
	r.eliminated = append(r.eliminated, eliminated)

	candidates = slices.DeleteFunc(candidates, existingParticipant(eliminated))
	if len(candidates) == 1 {
		r.winners = append(r.winners, candidates[0])
	}
	return nil
}

func (r *Roulette) spinModeRepeatableWinners() error {
	candidates := r.participants

	i := rand.IntN(len(candidates))
	winner := candidates[i]
	r.winners = append(r.winners, winner)
	return nil
}

func (r *Roulette) spinModeNoRepeatWinners() error {
	candidates := []Participant{}
	if len(r.winners) == 0 {
		candidates = r.participants
	} else {
		for index := range r.participants {
			if !slices.ContainsFunc(r.winners, existingParticipant(r.participants[index])) {
				candidates = append(candidates, r.participants[index])
			}
		}
	}
	if len(candidates) <= 0 {
		return errors.New("no more winners left")
	}

	i := rand.IntN(len(candidates))
	winner := candidates[i]
	r.winners = append(r.winners, winner)
	return nil
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
