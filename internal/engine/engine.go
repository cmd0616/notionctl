// Package engine implements the diff & reconcile logic between YAML config and Notion state.
package engine

import (
	"fmt"
	"strings"

	"github.com/radityajay/notionctl/internal/config"
	"github.com/radityajay/notionctl/internal/notion"
	"github.com/radityajay/notionctl/internal/property"
	"github.com/radityajay/notionctl/internal/state"
)

// Action represents a planned change.
type Action struct {
	Type         string // "create" or "update"
	DatabaseName string
	Details      []string // human-readable list of changes
}

// Engine coordinates planning and applying changes.
type Engine struct {
	config *config.Config
	state  *state.State
	client *notion.Client
	root   string
}

// New creates a new engine.
func New(cfg *config.Config, st *state.State, client *notion.Client, root string) *Engine {
	return &Engine{
		config: cfg,
		state:  st,
		client: client,
		root:   root,
	}
}

// Plan computes the diff between desired config and current state.
func (e *Engine) Plan() ([]Action, error) {
	var actions []Action

	for _, db := range e.config.Databases {
		_, exists := e.state.ResolveDatabase(db.Name)
		if !exists {
			// New database — needs creation
			details := []string{fmt.Sprintf("parent_page_id: %s", db.ParentPageID)}
			for name, prop := range db.Properties {
				details = append(details, fmt.Sprintf("+ property %q (%s)", name, prop.Type))
			}
			actions = append(actions, Action{
				Type:         "create",
				DatabaseName: db.Name,
				Details:      details,
			})
			continue
		}

		// Existing database — check for property changes
		// For v0.1, we detect new properties (additions).
		// Full diff against Notion API state is a future enhancement.
		var details []string
		for name, prop := range db.Properties {
			handler, err := property.Get(prop.Type)
			if err != nil {
				return nil, fmt.Errorf("database %q, property %q: %w", db.Name, name, err)
			}
			_ = handler // used for diff in future versions
		}

		if len(details) > 0 {
			actions = append(actions, Action{
				Type:         "update",
				DatabaseName: db.Name,
				Details:      details,
			})
		}
	}

	return actions, nil
}

// Apply executes the planned actions against the Notion API.
//
// Two-pass strategy for fresh creates:
//   - Pass 1: Create all databases with non-relation properties only.
//     This ensures every database has a Notion ID in state.
//   - Pass 2: Update all databases that have relation properties,
//     now that all IDs are resolvable.
func (e *Engine) Apply() ([]Action, error) {
	actions, err := e.Plan()
	if err != nil {
		return nil, err
	}

	if len(actions) == 0 {
		return actions, nil
	}

	// Collect which databases need relation updates after creation.
	type pendingRelation struct {
		db    *config.Database
		index int // index into actions slice
	}
	var pending []pendingRelation

	// --- Pass 1: Create/update databases (skip relation properties for new databases) ---
	for i, action := range actions {
		switch action.Type {
		case "create":
			db := e.findDatabase(action.DatabaseName)
			if db == nil {
				return nil, fmt.Errorf("database %q not found in config", action.DatabaseName)
			}

			if db.ParentPageID == "" {
				return nil, fmt.Errorf("database %q: parent_page_id required for new databases", db.Name)
			}

			// Build properties WITHOUT relations first
			props, hasRelations, err := e.buildPropertiesFiltered(db, false)
			if err != nil {
				return nil, fmt.Errorf("database %q: %w", db.Name, err)
			}

			// Create in Notion
			id, err := e.client.CreateDatabase(db.ParentPageID, db.Name, props)
			if err != nil {
				return nil, err
			}

			// Update state immediately so other databases can reference this ID
			e.state.SetDatabase(db.Name, id)
			actions[i].Details = append(actions[i].Details, fmt.Sprintf("→ created with ID %s", id))

			if hasRelations {
				pending = append(pending, pendingRelation{db: db, index: i})
			}

		case "update":
			db := e.findDatabase(action.DatabaseName)
			if db == nil {
				return nil, fmt.Errorf("database %q not found in config", action.DatabaseName)
			}

			dbID, _ := e.state.ResolveDatabase(db.Name)
			props, err := e.buildProperties(db)
			if err != nil {
				return nil, fmt.Errorf("database %q: %w", db.Name, err)
			}

			if err := e.client.UpdateDatabase(dbID, props); err != nil {
				return nil, err
			}

			actions[i].Details = append(actions[i].Details, "→ updated")
		}
	}

	// --- Pass 2: Update newly created databases with relation properties ---
	for _, p := range pending {
		dbID, _ := e.state.ResolveDatabase(p.db.Name)

		// Build only relation properties
		relationProps, _, err := e.buildPropertiesFiltered(p.db, true)
		if err != nil {
			return nil, fmt.Errorf("database %q (pass 2): %w", p.db.Name, err)
		}

		if len(relationProps) > 0 {
			if err := e.client.UpdateDatabase(dbID, relationProps); err != nil {
				return nil, fmt.Errorf("database %q: setting relations: %w", p.db.Name, err)
			}
			actions[p.index].Details = append(actions[p.index].Details, "→ relations linked")
		}
	}

	// Save state after all operations
	if err := e.state.Save(e.root); err != nil {
		return nil, fmt.Errorf("saving state: %w", err)
	}

	return actions, nil
}

// buildProperties builds the full Notion properties payload for a database.
func (e *Engine) buildProperties(db *config.Database) (map[string]interface{}, error) {
	props, _, err := e.buildPropertiesFiltered(db, false)
	if err != nil {
		return nil, err
	}
	// Also add relations
	relProps, _, err := e.buildPropertiesFiltered(db, true)
	if err != nil {
		return nil, err
	}
	for k, v := range relProps {
		props[k] = v
	}
	return props, nil
}

// buildPropertiesFiltered builds properties, either relations-only or non-relations-only.
// Returns the properties map, whether the database has any relation properties, and any error.
func (e *Engine) buildPropertiesFiltered(db *config.Database, relationsOnly bool) (map[string]interface{}, bool, error) {
	props := map[string]interface{}{}
	hasRelations := false

	for name, propDef := range db.Properties {
		isRelation := propDef.Type == "relation"

		if isRelation {
			hasRelations = true
		}

		// Filter: skip based on mode
		if relationsOnly && !isRelation {
			continue
		}
		if !relationsOnly && isRelation {
			continue
		}

		handler, err := property.Get(propDef.Type)
		if err != nil {
			return nil, hasRelations, fmt.Errorf("property %q: %w", name, err)
		}

		notionCfg, err := handler.ToNotion(propDef.Extra)
		if err != nil {
			return nil, hasRelations, fmt.Errorf("property %q: %w", name, err)
		}

		// Resolve relation references
		if isRelation {
			e.resolveRelations(notionCfg)
		}

		props[name] = notionCfg
	}

	return props, hasRelations, nil
}

func (e *Engine) resolveRelations(cfg property.NotionPropertyConfig) {
	for _, val := range cfg {
		if m, ok := val.(map[string]interface{}); ok {
			if dbID, ok := m["database_id"].(string); ok {
				if strings.HasPrefix(dbID, "{{resolve:") {
					name := strings.TrimPrefix(dbID, "{{resolve:")
					name = strings.TrimSuffix(name, "}}")
					if resolved, ok := e.state.ResolveDatabase(name); ok {
						m["database_id"] = resolved
					}
				}
			}
		}
	}
}

func (e *Engine) findDatabase(name string) *config.Database {
	for i := range e.config.Databases {
		if e.config.Databases[i].Name == name {
			return &e.config.Databases[i]
		}
	}
	return nil
}

// FormatPlan returns a human-readable plan output.
func FormatPlan(actions []Action) string {
	if len(actions) == 0 {
		return "No changes. Your Notion workspace matches the config."
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Plan: %d action(s)\n\n", len(actions)))

	for _, a := range actions {
		icon := "+"
		if a.Type == "update" {
			icon = "~"
		}
		b.WriteString(fmt.Sprintf("%s %s %q\n", icon, a.Type, a.DatabaseName))
		for _, d := range a.Details {
			b.WriteString(fmt.Sprintf("    %s\n", d))
		}
		b.WriteString("\n")
	}
	return b.String()
}
