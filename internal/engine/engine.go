// Package engine implements the diff & reconcile logic between YAML config and Notion state.
package engine

import (
	"fmt"
	"strings"

	"github.com/radityajayantara/notionctl/internal/config"
	"github.com/radityajayantara/notionctl/internal/notion"
	"github.com/radityajayantara/notionctl/internal/property"
	"github.com/radityajayantara/notionctl/internal/state"
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
func (e *Engine) Apply() ([]Action, error) {
	actions, err := e.Plan()
	if err != nil {
		return nil, err
	}

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

			// Build Notion properties payload
			props, err := e.buildProperties(db)
			if err != nil {
				return nil, fmt.Errorf("database %q: %w", db.Name, err)
			}

			// Create in Notion
			id, err := e.client.CreateDatabase(db.ParentPageID, db.Name, props)
			if err != nil {
				return nil, err
			}

			// Update state
			e.state.SetDatabase(db.Name, id)
			actions[i].Details = append(actions[i].Details, fmt.Sprintf("→ created with ID %s", id))

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

	// Save state after all operations
	if err := e.state.Save(e.root); err != nil {
		return nil, fmt.Errorf("saving state: %w", err)
	}

	return actions, nil
}

func (e *Engine) buildProperties(db *config.Database) (map[string]interface{}, error) {
	props := map[string]interface{}{}

	for name, propDef := range db.Properties {
		handler, err := property.Get(propDef.Type)
		if err != nil {
			return nil, fmt.Errorf("property %q: %w", name, err)
		}

		notionCfg, err := handler.ToNotion(propDef.Extra)
		if err != nil {
			return nil, fmt.Errorf("property %q: %w", name, err)
		}

		// Resolve relation references
		e.resolveRelations(notionCfg)

		props[name] = notionCfg
	}

	return props, nil
}

func (e *Engine) resolveRelations(cfg property.NotionPropertyConfig) {
	for key, val := range cfg {
		if m, ok := val.(map[string]interface{}); ok {
			if dbID, ok := m["database_id"].(string); ok {
				if strings.HasPrefix(dbID, "{{resolve:") {
					name := strings.TrimPrefix(dbID, "{{resolve:")
					name = strings.TrimSuffix(name, "}}")
					if resolved, ok := e.state.ResolveDatabase(name); ok {
						m["database_id"] = resolved
					}
					// If not resolved yet, it will be resolved in a second pass
					// after all databases are created.
				}
			}
		}
		_ = key
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
