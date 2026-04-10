package main_test

import (
	"errors"
	main "roulette"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddRouletteParticipant(t *testing.T) {
	duplicatedParticipant := main.NewParticipant("John")
	testCases := map[string]struct {
		roulette           *main.Roulette
		participant        main.Participant
		expectedErr        error
		assertParticipants func(r *main.Roulette)
	}{
		"it should fail if the roulette already have the participant added": {
			roulette: func() *main.Roulette {
				r := main.NewRoulette()
				err := r.AddParticipant(duplicatedParticipant)
				assert.Nil(t, err)
				return r
			}(),
			participant: duplicatedParticipant,
			expectedErr: errors.New("the given participant is already added into the roulette participants list"),
			assertParticipants: func(r *main.Roulette) {
				assert.Len(t, r.Participants(), 1)
				assert.Equal(t, "John", r.Participants()[0].Name())
			},
		},
		"it should fail if the roulette has already a participant with the same name": {
			roulette: func() *main.Roulette {
				r := main.NewRoulette()
				err := r.AddParticipant(duplicatedParticipant)
				assert.Nil(t, err)
				return r
			}(),
			participant: main.NewParticipant("John"),
			expectedErr: errors.New("the given participant is already added into the roulette participants list"),
			assertParticipants: func(r *main.Roulette) {
				assert.Len(t, r.Participants(), 1)
				assert.Equal(t, "John", r.Participants()[0].Name())
			},
		},
		"it should not fail and add the participant if the roulette has no participants added yet": {
			roulette:    main.NewRoulette(),
			participant: main.NewParticipant("Carl"),
			assertParticipants: func(r *main.Roulette) {
				assert.Len(t, r.Participants(), 1)
				assert.Equal(t, "Carl", r.Participants()[0].Name())
			},
		},
		"it should not fail and add the participant if the roulette has already some participants": {
			roulette: func() *main.Roulette {
				r := main.NewRoulette()
				err := r.AddParticipant(main.NewParticipant("Prime"))
				assert.Nil(t, err)
				err = r.AddParticipant(main.NewParticipant("Agen"))
				assert.Nil(t, err)
				return r
			}(),
			participant: main.NewParticipant("The Legend"),
			assertParticipants: func(r *main.Roulette) {
				assert.Len(t, r.Participants(), 3)
				assert.Equal(t, "Prime", r.Participants()[0].Name())
				assert.Equal(t, "Agen", r.Participants()[1].Name())
				assert.Equal(t, "The Legend", r.Participants()[2].Name())
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			err := tc.roulette.AddParticipant(tc.participant)
			assert.Equal(t, tc.expectedErr, err)
			tc.assertParticipants(tc.roulette)
		})
	}
}

func TestRemoveRouletteParticipant(t *testing.T) {
	removeParticipantCandidate := main.NewParticipant("Carl")
	testCases := map[string]struct {
		roulette           *main.Roulette
		participant        main.Participant
		expectedErr        error
		assertParticipants func(r *main.Roulette)
	}{
		"it should fail if the participant is not already added as participant for this roulette": {
			roulette:    main.NewRoulette(),
			participant: main.NewParticipant("John"),
			expectedErr: errors.New("the given participant is not part of the roulette participants list"),
			assertParticipants: func(r *main.Roulette) {
				assert.Len(t, r.Participants(), 0)
			},
		},
		"it should remove the correct participant": {
			roulette: func() *main.Roulette {
				r := main.NewRoulette()
				r.AddParticipant(removeParticipantCandidate)
				return r
			}(),
			participant: removeParticipantCandidate,
			assertParticipants: func(r *main.Roulette) {
				assert.Len(t, r.Participants(), 0)
			},
		},
		"it should remove the correct participants when multiple participants in the roulette": {
			roulette: func() *main.Roulette {
				r := main.NewRoulette()
				r.AddParticipant(main.NewParticipant("Jose"))
				r.AddParticipant(main.NewParticipant("Marie"))
				return r
			}(),
			participant: main.NewParticipant("Jose"),
			assertParticipants: func(r *main.Roulette) {
				assert.Len(t, r.Participants(), 1)
				assert.Equal(t, "Marie", r.Participants()[0].Name())
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			err := tc.roulette.RemoveParticipant(tc.participant)
			assert.Equal(t, tc.expectedErr, err)
			tc.assertParticipants(tc.roulette)
		})
	}
}
