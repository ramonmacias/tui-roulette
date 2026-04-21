package internal

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// rouletteDTO is the JSON-serializable representation of Roulette.
type rouletteDTO struct {
	ID                 uuid.UUID        `json:"id"`
	Name               string           `json:"name"`
	Mode               Mode             `json:"mode"`
	Participants       []participantDTO `json:"participants"`
	Winners            []participantDTO `json:"winners"`
	Eliminated         []participantDTO `json:"eliminated"`
	MultiWinnerCounter int              `json:"multiWinnerCounter"`
}

// participantDTO is the JSON-serializable representation of Participant.
type participantDTO struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// toDTO converts a Roulette to its DTO representation.
func toDTO(r *Roulette) rouletteDTO {
	participants := make([]participantDTO, len(r.participants))
	for i, p := range r.participants {
		participants[i] = participantDTO{ID: p.id, Name: p.name}
	}

	winners := make([]participantDTO, len(r.winners))
	for i, p := range r.winners {
		winners[i] = participantDTO{ID: p.id, Name: p.name}
	}

	eliminated := make([]participantDTO, len(r.eliminated))
	for i, p := range r.eliminated {
		eliminated[i] = participantDTO{ID: p.id, Name: p.name}
	}

	return rouletteDTO{
		ID:                 r.id,
		Name:               r.name,
		Mode:               r.mode,
		Participants:       participants,
		Winners:            winners,
		Eliminated:         eliminated,
		MultiWinnerCounter: r.multiWinnerCounter,
	}
}

// fromDTO converts a DTO to a Roulette.
func fromDTO(dto rouletteDTO) (*Roulette, error) {
	r := &Roulette{
		id:                 dto.ID,
		name:               dto.Name,
		mode:               dto.Mode,
		multiWinnerCounter: dto.MultiWinnerCounter,
	}

	r.participants = make([]Participant, len(dto.Participants))
	for i, pdto := range dto.Participants {
		r.participants[i] = Participant{id: pdto.ID, name: pdto.Name}
	}

	r.winners = make([]Participant, len(dto.Winners))
	for i, pdto := range dto.Winners {
		r.winners[i] = Participant{id: pdto.ID, name: pdto.Name}
	}

	r.eliminated = make([]Participant, len(dto.Eliminated))
	for i, pdto := range dto.Eliminated {
		r.eliminated[i] = Participant{id: pdto.ID, name: pdto.Name}
	}

	return r, nil
}

// Storage defines the interface for persisting roulettes.
type Storage interface {
	Save(roulettes []*Roulette) error
	Load() ([]*Roulette, error)
	Delete(id string) error
}

// FileStorage implements Storage using JSON files.
type FileStorage struct {
	filePath string
}

// NewFileStorage creates a new FileStorage instance that persists to ~/.config/tui-roulette/roulettes.json.
func NewFileStorage() (*FileStorage, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configDir := filepath.Join(homeDir, ".config", "tui-roulette")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, err
	}

	return &FileStorage{
		filePath: filepath.Join(configDir, "roulettes.json"),
	}, nil
}

// Save persists roulettes to disk as JSON.
func (fs *FileStorage) Save(roulettes []*Roulette) error {
	dtos := make([]rouletteDTO, len(roulettes))
	for i, r := range roulettes {
		dtos[i] = toDTO(r)
	}

	data, err := json.MarshalIndent(dtos, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(fs.filePath, data, 0644)
}

// Load reads roulettes from disk.
func (fs *FileStorage) Load() ([]*Roulette, error) {
	data, err := os.ReadFile(fs.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Roulette{}, nil
		}
		return nil, err
	}

	var dtos []rouletteDTO
	if err := json.Unmarshal(data, &dtos); err != nil {
		return nil, err
	}

	roulettes := make([]*Roulette, len(dtos))
	for i, dto := range dtos {
		r, err := fromDTO(dto)
		if err != nil {
			return nil, err
		}
		roulettes[i] = r
	}

	return roulettes, nil
}

// Delete removes a roulette by ID from storage.
// For file-based storage, this reloads, filters, and re-saves.
func (fs *FileStorage) Delete(id string) error {
	roulettes, err := fs.Load()
	if err != nil {
		return err
	}

	filtered := []*Roulette{}
	for _, r := range roulettes {
		if r.ID() != id {
			filtered = append(filtered, r)
		}
	}

	return fs.Save(filtered)
}
