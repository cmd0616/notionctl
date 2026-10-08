// Package importer converts existing Notion databases into notionctl YAML config and state.
package importer

import (
	"fmt"
	"strings"

	"github.com/radityajay/notionctl/internal/config"
	"github.com/radityajay/notionctl/internal/notion"
	"github.com/radityajay/notionctl/internal/state"
	"gopkg.in/yaml.v3"
)

// supportedTypes lists the property types notionctl can manage.
var supportedTypes = map[string]bool{
	"title": true, "rich_text": true, "number": true,
	"select": true, "multi_select": true, "relation": true,
	"checkbox": true, "date": true, "url": true,
	"email": true, "phone_number": true, "status": true,
	"created_time": true, "last_edited_time": true,
	"formula": true, "rollup": true,
	"people": true, "files": true, "unique_id": true,
}

// Result holds the output of an import operation.
type Result struct {
	Config   *config.Config
	State    *state.State
	Warnings []string
}

// Importer converts Notion databases to notionctl config.
type Importer struct {
	client *notion.Client
}

// New creates a new Importer.
func New(client *notion.Client) *Importer {
	return &Importer{client: client}
}

// FromPage imports all databases that are children of the given page.
func (imp *Importer) FromPage(pageID string) (*Result, error) {
	blocks, err := imp.client.ListBlockChildren(pageID)
	if err != nil {
		return nil, fmt.Errorf("listing page children: %w", err)
	}

	// Collect database IDs from child_database blocks
	var dbIDs []string
	for _, block := range blocks {
		blockType, _ := block["type"].(string)
		if blockType == "child_database" {
			if id, ok := block["id"].(string); ok {
				dbIDs = append(dbIDs, id)
			}
		}
	}

	if len(dbIDs) == 0 {
		return nil, fmt.Errorf("no databases found under page %s — make sure the page is shared with your integration", pageID)
	}

	return imp.fromDatabaseIDs(dbIDs, pageID)
}

// FromState syncs existing managed databases from Notion back to config.
// Uses the current state to know which databases to fetch, and preserves
// parent_page_id from the existing config.
func (imp *Importer) FromState(existingCfg *config.Config, st *state.State) (*Result, error) {
	if len(st.Databases) == 0 {
		return nil, fmt.Errorf("no databases in state — run 'notionctl apply' first")
	}

	// Build ID→name map and parent_page_id map from existing config
	var dbIDs []string
	nameToParent := map[string]string{}
	for _, db := range existingCfg.Databases {
		nameToParent[db.Name] = db.ParentPageID
	}
	for name, dbState := range st.Databases {
		dbIDs = append(dbIDs, dbState.ID)
		_ = name // used via idToName in fromDatabaseIDs
	}

	// Use empty parentPageID — we'll fix it after
	result, err := imp.fromDatabaseIDs(dbIDs, "")
	if err != nil {
		return nil, err
	}

	// Restore parent_page_id from existing config
	for i, db := range result.Config.Databases {
		if parent, ok := nameToParent[db.Name]; ok {
			result.Config.Databases[i].ParentPageID = parent
		}
	}

	return result, nil
}

// fromDatabaseIDs fetches each database schema and builds config + state.
func (imp *Importer) fromDatabaseIDs(dbIDs []string, parentPageID string) (*Result, error) {
	result := &Result{
		Config: &config.Config{Version: "1"},
		State: &state.State{
			Version:   "1",
			Databases: map[string]state.DatabaseState{},
		},
	}

	// First pass: fetch all databases, build ID→name map
	type dbInfo struct {
		id         string
		name       string
		properties map[string]interface{}
	}
	var databases []dbInfo
	idToName := map[string]string{}

	for _, id := range dbIDs {
		resp, err := imp.client.GetDatabase(id)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("skipping database %s: %v", id, err))
			continue
		}

		name := extractTitle(resp)
		if name == "" {
			name = fmt.Sprintf("Untitled_%s", id[:8])
		}

		props, _ := resp["properties"].(map[string]interface{})
		databases = append(databases, dbInfo{id: id, name: name, properties: props})
		idToName[id] = name
	}

	// Second pass: convert properties to YAML format
	for _, db := range databases {
		cfgDB := config.Database{
			Name:         db.name,
			ParentPageID: parentPageID,
			Properties:   map[string]config.PropertyDef{},
		}

		stateProps := map[string]state.PropertyState{}

		for propName, rawProp := range db.properties {
			propMap, ok := rawProp.(map[string]interface{})
			if !ok {
				continue
			}

			propType, _ := propMap["type"].(string)

			if !supportedTypes[propType] {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("database %q, property %q: skipped unsupported type %q", db.name, propName, propType))
				continue
			}

			propDef, warnings := mapProperty(propName, propType, propMap, idToName)
			for _, w := range warnings {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("database %q, property %q: %s", db.name, propName, w))
			}

			if propDef != nil {
				cfgDB.Properties[propName] = *propDef
				propID, _ := propMap["id"].(string)
				stateProps[propName] = state.PropertyState{
					ID:   propID,
					Type: propType,
				}
			}
		}

		result.Config.Databases = append(result.Config.Databases, cfgDB)
		result.State.Databases[db.name] = state.DatabaseState{
			ID:         db.id,
			Properties: stateProps,
		}
	}

	return result, nil
}

// mapProperty converts a single Notion property to a config.PropertyDef.
func mapProperty(name, propType string, propMap map[string]interface{}, idToName map[string]string) (*config.PropertyDef, []string) {
	def := &config.PropertyDef{
		Type:  propType,
		Extra: map[string]interface{}{},
	}
	var warnings []string

	switch propType {
	case "title", "rich_text", "checkbox", "date", "url", "email",
		"phone_number", "created_time", "last_edited_time",
		"people", "files":
		// No extra config needed

	case "number":
		if numCfg, ok := propMap["number"].(map[string]interface{}); ok {
			if format, ok := numCfg["format"].(string); ok && format != "number" {
				def.Extra["format"] = format
			}
		}

	case "select":
		if selCfg, ok := propMap["select"].(map[string]interface{}); ok {
			def.Extra["options"] = extractOptions(selCfg)
		}

	case "multi_select":
		if msCfg, ok := propMap["multi_select"].(map[string]interface{}); ok {
			def.Extra["options"] = extractOptions(msCfg)
		}

	case "status":
		if stCfg, ok := propMap["status"].(map[string]interface{}); ok {
			def.Extra["options"] = extractOptions(stCfg)
			if groups := extractGroups(stCfg); len(groups) > 0 {
				def.Extra["groups"] = groups
			}
		}

	case "formula":
		if fCfg, ok := propMap["formula"].(map[string]interface{}); ok {
			if expr, ok := fCfg["expression"].(string); ok {
				def.Extra["expression"] = expr
			}
		}

	case "rollup":
		if rCfg, ok := propMap["rollup"].(map[string]interface{}); ok {
			if relName, ok := rCfg["relation_property_name"].(string); ok {
				def.Extra["relation"] = relName
			}
			if rpName, ok := rCfg["rollup_property_name"].(string); ok {
				def.Extra["rollup_property"] = rpName
			}
			if fn, ok := rCfg["function"].(string); ok {
				def.Extra["function"] = fn
			}
		}

	case "unique_id":
		if uidCfg, ok := propMap["unique_id"].(map[string]interface{}); ok {
			if prefix, ok := uidCfg["prefix"].(string); ok && prefix != "" {
				def.Extra["prefix"] = prefix
			}
		}

	case "relation":
		if relCfg, ok := propMap["relation"].(map[string]interface{}); ok {
			dbID, _ := relCfg["database_id"].(string)
			targetName, found := idToName[dbID]
			if found {
				def.Extra["relation"] = targetName
				// Check for two-way relation
				relType, _ := relCfg["type"].(string)
				if relType == "dual_property" {
					if dp, ok := relCfg["dual_property"].(map[string]interface{}); ok {
						if syncName, ok := dp["synced_property_name"].(string); ok {
							def.Extra["synced_property"] = syncName
						}
					}
				}
			} else {
				warnings = append(warnings, fmt.Sprintf("relation target database %s not in import scope, skipped", dbID))
				return nil, warnings
			}
		}
	}

	return def, warnings
}

// extractTitle gets the database title from a Notion response.
func extractTitle(db map[string]interface{}) string {
	titleArr, ok := db["title"].([]interface{})
	if !ok || len(titleArr) == 0 {
		return ""
	}
	var title strings.Builder
	for _, raw := range titleArr {
		if part, ok := raw.(map[string]interface{}); ok {
			text, _ := part["plain_text"].(string)
			title.WriteString(text)
		}
	}
	return title.String()
}

// extractOptions extracts select/multi_select/status options.
func extractOptions(cfg map[string]interface{}) []interface{} {
	rawOpts, ok := cfg["options"].([]interface{})
	if !ok {
		return nil
	}
	var opts []interface{}
	for _, raw := range rawOpts {
		if m, ok := raw.(map[string]interface{}); ok {
			opt := map[string]interface{}{}
			if name, ok := m["name"].(string); ok {
				opt["name"] = name
			}
			if color, ok := m["color"].(string); ok && color != "default" {
				opt["color"] = color
			}
			opts = append(opts, opt)
		}
	}
	return opts
}

// extractGroups extracts status groups.
func extractGroups(cfg map[string]interface{}) []interface{} {
	rawGroups, ok := cfg["groups"].([]interface{})
	if !ok {
		return nil
	}
	var groups []interface{}
	for _, raw := range rawGroups {
		if m, ok := raw.(map[string]interface{}); ok {
			group := map[string]interface{}{}
			if name, ok := m["name"].(string); ok {
				group["name"] = name
			}
			if color, ok := m["color"].(string); ok {
				group["color"] = color
			}
			if optIDs, ok := m["option_ids"].([]interface{}); ok {
				group["option_ids"] = optIDs
			}
			groups = append(groups, group)
		}
	}
	return groups
}

// MarshalYAML converts a Config to YAML bytes.
func MarshalYAML(cfg *config.Config) ([]byte, error) {
	// Build a clean YAML-friendly structure
	type yamlProp struct {
		Type string `yaml:"type"`
		// Extra fields are inlined
	}

	type yamlDB struct {
		Name         string                 `yaml:"name"`
		ParentPageID string                 `yaml:"parent_page_id"`
		Properties   map[string]interface{} `yaml:"properties"`
	}

	type yamlConfig struct {
		Version   string   `yaml:"version"`
		Databases []yamlDB `yaml:"databases"`
	}

	out := yamlConfig{Version: cfg.Version}

	for _, db := range cfg.Databases {
		ydb := yamlDB{
			Name:         db.Name,
			ParentPageID: db.ParentPageID,
			Properties:   map[string]interface{}{},
		}

		for propName, prop := range db.Properties {
			propMap := map[string]interface{}{
				"type": prop.Type,
			}
			for k, v := range prop.Extra {
				propMap[k] = v
			}
			ydb.Properties[propName] = propMap
		}

		out.Databases = append(out.Databases, ydb)
	}

	header := "# Generated by notionctl import\n# Review and customize, then run: notionctl plan\n"
	data, err := yaml.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("marshaling YAML: %w", err)
	}

	return []byte(header + string(data)), nil
}

// FormatWarnings returns a human-readable warning summary.
func FormatWarnings(warnings []string) string {
	if len(warnings) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("\n⚠ %d warning(s):\n", len(warnings)))
	for _, w := range warnings {
		b.WriteString(fmt.Sprintf("  • %s\n", w))
	}
	return b.String()
}
