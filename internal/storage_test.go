package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileStoragePersistence(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "tui-roulette-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a custom FileStorage pointing to temp directory
	configDir := filepath.Join(tempDir, ".config", "tui-roulette")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	storage := &FileStorage{
		filePath: filepath.Join(configDir, "roulettes.json"),
	}

	// Create some test roulettes
	roulette1 := NewRoulette("Test Roulette 1", ModeRepeatableWinners)
	if err := roulette1.AddParticipant(NewParticipant("Alice")); err != nil {
		t.Fatalf("Failed to add participant: %v", err)
	}
	if err := roulette1.AddParticipant(NewParticipant("Bob")); err != nil {
		t.Fatalf("Failed to add participant: %v", err)
	}

	roulette2 := NewRoulette("Test Roulette 2", ModeNoRepeatWinners)
	if err := roulette2.AddParticipant(NewParticipant("Charlie")); err != nil {
		t.Fatalf("Failed to add participant: %v", err)
	}

	original := []*Roulette{roulette1, roulette2}

	// Save
	if err := storage.Save(original); err != nil {
		t.Fatalf("Failed to save roulettes: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(storage.filePath); err != nil {
		t.Fatalf("Persisted file does not exist: %v", err)
	}

	// Load
	loaded, err := storage.Load()
	if err != nil {
		t.Fatalf("Failed to load roulettes: %v", err)
	}

	// Verify counts
	if len(loaded) != len(original) {
		t.Fatalf("Expected %d roulettes, got %d", len(original), len(loaded))
	}

	// Verify data integrity
	if loaded[0].Name() != "Test Roulette 1" {
		t.Fatalf("Expected name 'Test Roulette 1', got '%s'", loaded[0].Name())
	}

	if len(loaded[0].Participants()) != 2 {
		t.Fatalf("Expected 2 participants, got %d", len(loaded[0].Participants()))
	}

	if loaded[0].Participants()[0].Name() != "Alice" {
		t.Fatalf("Expected participant 'Alice', got '%s'", loaded[0].Participants()[0].Name())
	}

	if loaded[1].Mode() != ModeNoRepeatWinners {
		t.Fatalf("Expected mode %s, got %s", ModeNoRepeatWinners, loaded[1].Mode())
	}

	t.Logf("Persistence test passed: saved and loaded %d roulettes successfully", len(loaded))
}

func TestFileStorageEmptyLoad(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tui-roulette-test-empty-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configDir := filepath.Join(tempDir, ".config", "tui-roulette")

	storage := &FileStorage{
		filePath: filepath.Join(configDir, "roulettes.json"),
	}

	// Load from non-existent file should return empty slice
	loaded, err := storage.Load()
	if err != nil {
		t.Fatalf("Expected no error for missing file, got: %v", err)
	}

	if len(loaded) != 0 {
		t.Fatalf("Expected empty slice for missing file, got %d roulettes", len(loaded))
	}

	t.Logf("Empty load test passed: correctly handled missing file")
}

func TestFileStorageDelete(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tui-roulette-test-delete-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configDir := filepath.Join(tempDir, ".config", "tui-roulette")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	storage := &FileStorage{
		filePath: filepath.Join(configDir, "roulettes.json"),
	}

	// Create and save
	roulette1 := NewRoulette("Roulette 1", ModeRepeatableWinners)
	roulette2 := NewRoulette("Roulette 2", ModeNoRepeatWinners)
	original := []*Roulette{roulette1, roulette2}

	if err := storage.Save(original); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Store ID for later deletion
	idToDelete := roulette1.ID()

	// Delete
	if err := storage.Delete(idToDelete); err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	// Load and verify deletion
	loaded, err := storage.Load()
	if err != nil {
		t.Fatalf("Failed to load after delete: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("Expected 1 roulette after deletion, got %d", len(loaded))
	}

	if loaded[0].Name() != "Roulette 2" {
		t.Fatalf("Expected remaining roulette to be 'Roulette 2', got '%s'", loaded[0].Name())
	}

	t.Logf("Delete test passed: successfully deleted and reloaded")
}
