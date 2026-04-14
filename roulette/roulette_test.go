package roulette_test

import (
	"errors"
	"roulette/roulette"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddRouletteParticipant(t *testing.T) {
	duplicatedParticipant := roulette.NewParticipant("John")
	testCases := map[string]struct {
		roulette           *roulette.Roulette
		participant        roulette.Participant
		expectedErr        error
		assertParticipants func(r *roulette.Roulette)
	}{
		"it should fail if the roulette already have the participant added": {
			roulette: func() *roulette.Roulette {
				r := roulette.NewRoulette("test-roulette", roulette.ModeNoRepeatWinners)
				err := r.AddParticipant(duplicatedParticipant)
				assert.Nil(t, err)
				return r
			}(),
			participant: duplicatedParticipant,
			expectedErr: errors.New("the given participant is already added into the roulette participants list"),
			assertParticipants: func(r *roulette.Roulette) {
				assert.Len(t, r.Participants(), 1)
				assert.Equal(t, "John", r.Participants()[0].Name())
			},
		},
		"it should fail if the roulette has already a participant with the same name": {
			roulette: func() *roulette.Roulette {
				r := roulette.NewRoulette("test-roulette", roulette.ModeNoRepeatWinners)
				err := r.AddParticipant(duplicatedParticipant)
				assert.Nil(t, err)
				return r
			}(),
			participant: roulette.NewParticipant("John"),
			expectedErr: errors.New("the given participant is already added into the roulette participants list"),
			assertParticipants: func(r *roulette.Roulette) {
				assert.Len(t, r.Participants(), 1)
				assert.Equal(t, "John", r.Participants()[0].Name())
			},
		},
		"it should not fail and add the participant if the roulette has no participants added yet": {
			roulette:    roulette.NewRoulette("test-roulette", roulette.ModeNoRepeatWinners),
			participant: roulette.NewParticipant("Carl"),
			assertParticipants: func(r *roulette.Roulette) {
				assert.Len(t, r.Participants(), 1)
				assert.Equal(t, "Carl", r.Participants()[0].Name())
			},
		},
		"it should not fail and add the participant if the roulette has already some participants": {
			roulette: func() *roulette.Roulette {
				r := roulette.NewRoulette("test-roulette", roulette.ModeNoRepeatWinners)
				err := r.AddParticipant(roulette.NewParticipant("Prime"))
				assert.Nil(t, err)
				err = r.AddParticipant(roulette.NewParticipant("Agen"))
				assert.Nil(t, err)
				return r
			}(),
			participant: roulette.NewParticipant("The Legend"),
			assertParticipants: func(r *roulette.Roulette) {
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
	removeParticipantCandidate := roulette.NewParticipant("Carl")
	testCases := map[string]struct {
		roulette           *roulette.Roulette
		participant        roulette.Participant
		expectedErr        error
		assertParticipants func(r *roulette.Roulette)
	}{
		"it should fail if the participant is not already added as participant for this roulette": {
			roulette:    roulette.NewRoulette("test-roulette", roulette.ModeNoRepeatWinners),
			participant: roulette.NewParticipant("John"),
			expectedErr: errors.New("the given participant is not part of the roulette participants list"),
			assertParticipants: func(r *roulette.Roulette) {
				assert.Len(t, r.Participants(), 0)
			},
		},
		"it should remove the correct participant": {
			roulette: func() *roulette.Roulette {
				r := roulette.NewRoulette("test-roulette", roulette.ModeNoRepeatWinners)
				r.AddParticipant(removeParticipantCandidate)
				return r
			}(),
			participant: removeParticipantCandidate,
			assertParticipants: func(r *roulette.Roulette) {
				assert.Len(t, r.Participants(), 0)
			},
		},
		"it should remove the correct participants when multiple participants in the roulette": {
			roulette: func() *roulette.Roulette {
				r := roulette.NewRoulette("test-roulette", roulette.ModeNoRepeatWinners)
				r.AddParticipant(roulette.NewParticipant("Jose"))
				r.AddParticipant(roulette.NewParticipant("Marie"))
				return r
			}(),
			participant: roulette.NewParticipant("Jose"),
			assertParticipants: func(r *roulette.Roulette) {
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

func TestSpinModeRepeatableWinnersRoulette(t *testing.T) {
	testCases := map[string]struct {
		roulette     *roulette.Roulette
		expectedErr  error
		assertWinner func(r *roulette.Roulette)
	}{
		"it should return an error if no participants are found in the roulette": {
			roulette:    roulette.NewRoulette("test-roulette", roulette.ModeRepeatableWinners),
			expectedErr: errors.New("cannot spin roulette without participants"),
			assertWinner: func(r *roulette.Roulette) {
				assert.Len(t, r.Winners(), 0)
			},
		},
		"it should return the winner if only one participant": {
			roulette: func() *roulette.Roulette {
				r := roulette.NewRoulette("test-roulette", roulette.ModeRepeatableWinners)
				r.AddParticipant(roulette.NewParticipant("John"))
				return r
			}(),
			assertWinner: func(r *roulette.Roulette) {
				assert.Len(t, r.Winners(), 1)
				assert.Equal(t, "John", r.Winners()[0].Name())
			},
		},
		"it should randomly pick up one participant as a winner and add it into the winners list": {
			roulette: func() *roulette.Roulette {
				r := roulette.NewRoulette("test-roulette", roulette.ModeRepeatableWinners)
				r.AddParticipant(roulette.NewParticipant("John"))
				r.AddParticipant(roulette.NewParticipant("Prime"))
				r.AddParticipant(roulette.NewParticipant("TJ"))
				r.AddParticipant(roulette.NewParticipant("Rodrigo"))
				return r
			}(),
			assertWinner: func(r *roulette.Roulette) {
				assert.Len(t, r.Winners(), 1)
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			err := tc.roulette.Spin()
			assert.Equal(t, tc.expectedErr, err)
			tc.assertWinner(tc.roulette)
		})
	}
}

func TestSpinModeNoRepeatWinnersRoulette(t *testing.T) {
	testCases := map[string]struct {
		roulette     *roulette.Roulette
		expectedErr  error
		assertWinner func(r *roulette.Roulette)
	}{
		"it should return an error if there are no participants": {
			roulette:    roulette.NewRoulette("test-roulette", roulette.ModeNoRepeatWinners),
			expectedErr: errors.New("cannot spin roulette without participants"),
			assertWinner: func(r *roulette.Roulette) {
				assert.Len(t, r.Winners(), 0)
			},
		},
		"it should return no more winners left error if the last participant already won": {
			roulette: func() *roulette.Roulette {
				r := roulette.NewRoulette("test-roulette", roulette.ModeNoRepeatWinners)
				r.AddParticipant(roulette.NewParticipant("John"))
				err := r.Spin()
				assert.Nil(t, err)
				return r
			}(),
			expectedErr: errors.New("no more winners left"),
			assertWinner: func(r *roulette.Roulette) {
				assert.Len(t, r.Winners(), 1)
			},
		},
		"it should never pickup one of the winners": {
			roulette: func() *roulette.Roulette {
				r := roulette.NewRoulette("test-roulette", roulette.ModeNoRepeatWinners)
				r.AddParticipant(roulette.NewParticipant("John"))
				r.AddParticipant(roulette.NewParticipant("Alfred"))
				err := r.Spin()
				assert.Nil(t, err)
				return r
			}(),
			assertWinner: func(r *roulette.Roulette) {
				assert.Len(t, r.Winners(), 2)
				var foundJohn, foundAlfred int
				for i := range r.Winners() {
					if r.Winners()[i].Name() == "John" {
						foundJohn++
					} else if r.Winners()[i].Name() == "Alfred" {
						foundAlfred++
					}
				}
				assert.Equal(t, 1, foundAlfred)
				assert.Equal(t, 1, foundJohn)
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			err := tc.roulette.Spin()
			assert.Equal(t, tc.expectedErr, err)
			tc.assertWinner(tc.roulette)
		})
	}
}

func TestSpinModeEliminationRoulette(t *testing.T) {
	testCases := map[string]struct {
		roulette     *roulette.Roulette
		expectedErr  error
		assertWinner func(r *roulette.Roulette)
	}{
		"it should return an error if there are no participants": {
			roulette:    roulette.NewRoulette("test-roulette", roulette.ModeElimination),
			expectedErr: errors.New("cannot spin roulette without participants"),
			assertWinner: func(r *roulette.Roulette) {
				assert.Len(t, r.Winners(), 0)
			},
		},
		"it should return an error if we have already a winner": {
			roulette: func() *roulette.Roulette {
				r := roulette.NewRoulette("test-roulette", roulette.ModeElimination)
				r.AddParticipant(roulette.NewParticipant("John"))
				r.AddParticipant(roulette.NewParticipant("Alfred"))
				err := r.Spin()
				assert.Nil(t, err)
				return r
			}(),
			expectedErr: errors.New("we have already a winner"),
			assertWinner: func(r *roulette.Roulette) {
				assert.Len(t, r.Winners(), 1)
			},
		},
		"it should return an error when we have only one participant": {
			roulette: func() *roulette.Roulette {
				r := roulette.NewRoulette("test-roulette", roulette.ModeElimination)
				r.AddParticipant(roulette.NewParticipant("John"))
				return r
			}(),
			expectedErr: errors.New("we have already a winner"),
			assertWinner: func(r *roulette.Roulette) {
				assert.Len(t, r.Winners(), 1)
			},
		},
		"it should not have any winner if we still have some participants that are not eliminated yet": {
			roulette: func() *roulette.Roulette {
				r := roulette.NewRoulette("test-roulette", roulette.ModeElimination)
				r.AddParticipant(roulette.NewParticipant("John"))
				r.AddParticipant(roulette.NewParticipant("Alfred"))
				r.AddParticipant(roulette.NewParticipant("Ramon"))
				return r
			}(),
			assertWinner: func(r *roulette.Roulette) {
				assert.Len(t, r.Winners(), 0)
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			err := tc.roulette.Spin()
			assert.Equal(t, tc.expectedErr, err)
			tc.assertWinner(tc.roulette)
		})
	}
}
