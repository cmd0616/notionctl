package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEmpty(t *testing.T) {
	dir := t.TempDir()
	st, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.Version != "1" {
		t.Errorf("expected version 1, got %s", st.Version)
	}
	if len(st.Databases) != 0 {
		t.Errorf("expected empty databases, got %d", len(st.Databases))
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()

	st := &State{
		Version: "1",
		Databases: map[string]DatabaseState{
			"Projects": {ID: "abc-123", Properties: map[string]PropertyState{"Name": {ID: "prop-1", Type: "title"}}},
		},
	}

	if err := st.Save(dir); err != nil {
		t.Fatalf("save error: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filepath.Join(dir, Dir, File)); err != nil {
		t.Fatalf("state file not created: %v", err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}

	if loaded.Version != "1" {
		t.Errorf("expected version 1, got %s", loaded.Version)
	}

	db, ok := loaded.Databases["Projects"]
	if !ok {
		t.Fatal("Projects not found in loaded state")
	}
	if db.ID != "abc-123" {
		t.Errorf("expected ID abc-123, got %s", db.ID)
	}
	if db.Properties["Name"].ID != "prop-1" {
		t.Errorf("expected property ID prop-1, got %s", db.Properties["Name"].ID)
	}
}

func TestResolveDatabase(t *testing.T) {
	st := &State{
		Version: "1",
		Databases: map[string]DatabaseState{
			"Tasks": {ID: "task-id"},
		},
	}

	id, ok := st.ResolveDatabase("Tasks")
	if !ok || id != "task-id" {
		t.Errorf("expected task-id, got %s (ok=%v)", id, ok)
	}

	_, ok = st.ResolveDatabase("NonExistent")
	if ok {
		t.Error("expected not found for NonExistent")
	}
}

func TestSetDatabase(t *testing.T) {
	st := &State{Version: "1", Databases: map[string]DatabaseState{}}
	st.SetDatabase("New", "new-id")

	id, ok := st.ResolveDatabase("New")
	if !ok || id != "new-id" {
		t.Errorf("expected new-id, got %s (ok=%v)", id, ok)
	}
}
