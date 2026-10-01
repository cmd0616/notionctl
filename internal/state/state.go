// Package state manages the local state file that maps symbolic names to Notion IDs.
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// Dir is the directory where state is stored.
	Dir = ".notionctl"
	// File is the state file name.
	File = "state.json"
)

// State holds the mapping between symbolic database names and Notion database IDs.
type State struct {
	// Version of the state file format.
	Version string `json:"version"`

	// Databases maps symbolic name → Notion database ID.
	Databases map[string]DatabaseState `json:"databases"`
}

// DatabaseState holds the Notion-side info for a managed database.
type DatabaseState struct {
	// ID is the Notion database UUID.
	ID string `json:"id"`

	// Properties maps property name → Notion property ID.
	Properties map[string]string `json:"properties,omitempty"`
}

// Path returns the full path to the state file relative to the given root.
func Path(root string) string {
	return filepath.Join(root, Dir, File)
}

// Load reads the state file. Returns empty state if file doesn't exist.
func Load(root string) (*State, error) {
	p := Path(root)
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return &State{
			Version:   "1",
			Databases: map[string]DatabaseState{},
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading state: %w", err)
	}

	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing state: %w", err)
	}
	return &s, nil
}

// Save writes the state file to disk, creating the directory if needed.
func (s *State) Save(root string) error {
	dir := filepath.Join(root, Dir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating state dir: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding state: %w", err)
	}

	p := Path(root)
	if err := os.WriteFile(p, data, 0o644); err != nil {
		return fmt.Errorf("writing state: %w", err)
	}
	return nil
}

// ResolveDatabase returns the Notion database ID for a symbolic name.
func (s *State) ResolveDatabase(name string) (string, bool) {
	db, ok := s.Databases[name]
	if !ok {
		return "", false
	}
	return db.ID, true
}

// SetDatabase records a database in state.
func (s *State) SetDatabase(name, id string) {
	s.Databases[name] = DatabaseState{
		ID:         id,
		Properties: map[string]string{},
	}
}
