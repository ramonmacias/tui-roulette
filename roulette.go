package main

import (
	"errors"
	"slices"

	"github.com/google/uuid"
)

// Roulette stores the participants and winners for a raffle draw.
type Roulette struct {
	id           uuid.UUID
	participants []Participant
	winners      []Participant
}

// NewRoulette creates an empty roulette with a generated identifier.
func NewRoulette() *Roulette {
	return &Roulette{
		id:           uuid.New(),
		participants: []Participant{},
		winners:      []Participant{},
	}
}

var existingParticipant = func(p Participant) func(existing Participant) bool {
	return func(existing Participant) bool {
		return existing.name == p.name
	}
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
