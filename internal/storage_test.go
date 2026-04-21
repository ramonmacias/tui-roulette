package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestStorage(t *testing.T) *FileStorage {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "tui-roulette-test-*")
	assert.NoError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(tempDir)
	})

	configDir := filepath.Join(tempDir, ".config", "tui-roulette")
	assert.NoError(t, os.MkdirAll(configDir, 0o755))

	return &FileStorage{filePath: filepath.Join(configDir, "roulettes.json")}
}

func TestFileStoragePersistence(t *testing.T) {
	testCases := map[string]struct {
		setupData func(t *testing.T) []*Roulette
		assertFn  func(t *testing.T, loaded []*Roulette)
	}{
		"it should persist and load roulettes with participants and modes": {
			setupData: func(t *testing.T) []*Roulette {
				roulette1 := NewRoulette("Test Roulette 1", ModeRepeatableWinners)
				assert.NoError(t, roulette1.AddParticipant(NewParticipant("Alice")))
				assert.NoError(t, roulette1.AddParticipant(NewParticipant("Bob")))

				roulette2 := NewRoulette("Test Roulette 2", ModeNoRepeatWinners)
				assert.NoError(t, roulette2.AddParticipant(NewParticipant("Charlie")))

				return []*Roulette{roulette1, roulette2}
			},
			assertFn: func(t *testing.T, loaded []*Roulette) {
				assert.Len(t, loaded, 2)
				assert.Equal(t, "Test Roulette 1", loaded[0].Name())
				assert.Len(t, loaded[0].Participants(), 2)
				assert.Equal(t, "Alice", loaded[0].Participants()[0].Name())
				assert.Equal(t, ModeNoRepeatWinners, loaded[1].Mode())
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			storage := newTestStorage(t)
			original := tc.setupData(t)

			err := storage.Save(original)
			assert.NoError(t, err)

			_, err = os.Stat(storage.filePath)
			assert.NoError(t, err)

			loaded, err := storage.Load()
			assert.NoError(t, err)

			tc.assertFn(t, loaded)
		})
	}
}

func TestFileStorageLoad(t *testing.T) {
	testCases := map[string]struct {
		prepareStorage func(t *testing.T, storage *FileStorage)
		assertFn       func(t *testing.T, loaded []*Roulette, err error)
	}{
		"it should return an empty slice when file does not exist": {
			prepareStorage: func(t *testing.T, storage *FileStorage) {},
			assertFn: func(t *testing.T, loaded []*Roulette, err error) {
				assert.NoError(t, err)
				assert.Empty(t, loaded)
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			storage := newTestStorage(t)
			tc.prepareStorage(t, storage)

			loaded, err := storage.Load()
			tc.assertFn(t, loaded, err)
		})
	}
}

func TestFileStorageDelete(t *testing.T) {
	testCases := map[string]struct {
		setupData     func(t *testing.T) ([]*Roulette, string)
		expectedNames []string
	}{
		"it should delete the roulette by id": {
			setupData: func(t *testing.T) ([]*Roulette, string) {
				roulette1 := NewRoulette("Roulette 1", ModeRepeatableWinners)
				roulette2 := NewRoulette("Roulette 2", ModeNoRepeatWinners)
				return []*Roulette{roulette1, roulette2}, roulette1.ID()
			},
			expectedNames: []string{"Roulette 2"},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			storage := newTestStorage(t)
			original, idToDelete := tc.setupData(t)

			err := storage.Save(original)
			assert.NoError(t, err)

			err = storage.Delete(idToDelete)
			assert.NoError(t, err)

			loaded, err := storage.Load()
			assert.NoError(t, err)
			assert.Len(t, loaded, len(tc.expectedNames))

			for i, expectedName := range tc.expectedNames {
				assert.Equal(t, expectedName, loaded[i].Name())
			}
		})
	}
}

func TestFileStorageUpdatePersistedRoulette(t *testing.T) {
	testCases := map[string]struct {
		setupData         func(t *testing.T) []*Roulette
		updatePersisted   func(t *testing.T, loaded []*Roulette)
		assertAfterReload func(t *testing.T, loaded []*Roulette)
	}{
		"it should persist updates after loading and saving again": {
			setupData: func(t *testing.T) []*Roulette {
				r := NewRoulette("Daily Standup", ModeRepeatableWinners)
				assert.NoError(t, r.AddParticipant(NewParticipant("Alice")))
				assert.NoError(t, r.AddParticipant(NewParticipant("Bob")))
				return []*Roulette{r}
			},
			updatePersisted: func(t *testing.T, loaded []*Roulette) {
				assert.Len(t, loaded, 1)
				assert.NoError(t, loaded[0].AddParticipant(NewParticipant("Carol")))
				assert.NoError(t, loaded[0].Spin())
			},
			assertAfterReload: func(t *testing.T, loaded []*Roulette) {
				assert.Len(t, loaded, 1)
				assert.Equal(t, "Daily Standup", loaded[0].Name())
				assert.Len(t, loaded[0].Participants(), 3)
				assert.Len(t, loaded[0].Winners(), 1)

				participantNames := []string{}
				for _, p := range loaded[0].Participants() {
					participantNames = append(participantNames, p.Name())
				}
				assert.ElementsMatch(t, []string{"Alice", "Bob", "Carol"}, participantNames)
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			storage := newTestStorage(t)
			original := tc.setupData(t)

			err := storage.Save(original)
			assert.NoError(t, err)

			loaded, err := storage.Load()
			assert.NoError(t, err)

			tc.updatePersisted(t, loaded)

			err = storage.Save(loaded)
			assert.NoError(t, err)

			reloaded, err := storage.Load()
			assert.NoError(t, err)

			tc.assertAfterReload(t, reloaded)
		})
	}
}
