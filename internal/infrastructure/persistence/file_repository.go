package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"dnd-char-generator/internal/domain"
)

type FileRepository struct {
	mu       sync.RWMutex
	filePath string
}

func NewFileRepository(filePath string) *FileRepository {
	return &FileRepository{
		filePath: filePath,
	}
}

func (r *FileRepository) loadFromFile() ([]*domain.Character, error) {
	data, err := os.ReadFile(r.filePath)

	// If the file doesn't exist, return empty slice
	if os.IsNotExist(err) {
		return []*domain.Character{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error reading character file: %w", err)
	}

	var chars []*domain.Character
	// Handle empty file
	if len(data) > 0 {
		err = json.Unmarshal(data, &chars)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling character data: %w", err)
		}
	}
	return chars, nil
}

func (r *FileRepository) Save(ctx context.Context, char *domain.Character) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Load all characters
	chars, err := r.loadFromFile()
	if err != nil {
		return err
	}

	// Update character
	found := false
	for i, existing := range chars {
		if existing.Name == char.Name {
			chars[i] = char
			found = true
			break
		}
	}
	if !found {
		chars = append(chars, char)
	}

	// Convert back to JSON
	data, err := json.MarshalIndent(chars, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(r.filePath, data, 0644)
}

func (r *FileRepository) FindByID(ctx context.Context, name string) (*domain.Character, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	chars, err := r.loadFromFile()
	if err != nil {
		return nil, err
	}

	for _, char := range chars {
		if char.Name == name {
			return char, nil
		}
	}
	return nil, fmt.Errorf("character '%s' not found", name)
}

func (r *FileRepository) FindAll(ctx context.Context) ([]*domain.Character, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.loadFromFile()
}

func (r *FileRepository) Delete(ctx context.Context, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Load all characters
	chars, err := r.loadFromFile()
	if err != nil {
		return err
	}

	// Find and filter out the character to be deleted
	found := false
	var updatedChars []*domain.Character
	for _, char := range chars {
		if char.Name == name {
			found = true
			// Skip adding character to the new list
			continue
		}
		updatedChars = append(updatedChars, char)
	}

	if !found {
		return fmt.Errorf("character '%s' not found", name)
	}

	// Back to JSON
	data, err := json.MarshalIndent(updatedChars, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling data after delete: %w", err)
	}

	// Save the new list to the file
	return os.WriteFile(r.filePath, data, 0644)
}
