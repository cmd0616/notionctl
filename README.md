# notionctl

Declarative Notion database management. Define your databases in YAML, sync them to Notion with one command.

**The problem:** Managing Notion database IDs and relations across automations is painful. IDs change when you copy databases, differ between workspaces, and are impossible to read in scripts.

**The solution:** Declare databases and relations by **name**, not by ID. `notionctl` resolves symbolic references and manages state for you.

```yaml
databases:
  - name: Projects
    properties:
      tasks:
        type: relation
        relation: Tasks    # ← symbolic name, not a UUID

  - name: Tasks
    properties:
      project:
        type: relation
        relation: Projects  # ← two-way relation, by name
```

## Quick Start

```bash
# Install
go install github.com/radityajayantara/notionctl@latest

# Create a starter config
notionctl init

# Edit notionctl.yaml — set your page IDs and customize properties

# Set your Notion integration token
export NOTION_TOKEN="secret_..."

# Preview changes
notionctl plan

# Apply to Notion
notionctl apply
```

## Commands

| Command | Description |
|---------|-------------|
| `notionctl init` | Generate starter `notionctl.yaml` |
| `notionctl plan` | Preview changes (diff config vs state) |
| `notionctl apply` | Create/update databases in Notion |

## Supported Property Types

| Type | Description | Configurable |
|------|-------------|--------------|
| `title` | Database title column | — |
| `rich_text` | Text content | — |
| `number` | Numeric values | `format` (dollar, percent, etc.) |
| `select` | Single choice | `options` (name + color) |
| `multi_select` | Multiple choices | `options` (name + color) |
| `relation` | Link to another database | `relation` (target name), `synced_property` |

## How It Works

1. **Write YAML** — Declare databases with properties and relations by name
2. **`notionctl plan`** — Compares your YAML against local state (`.notionctl/state.json`)
3. **`notionctl apply`** — Creates/updates databases via Notion API, saves state

State is stored in `.notionctl/state.json` (add to `.gitignore` — it contains workspace-specific IDs).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for how to add new property types, templates, and more.

**Quick contribution: Add a property type in one file.** Every Notion property type (checkbox, date, url, email, etc.) is a single Go file implementing one interface. See `internal/property/title.go` for the minimal example.

## Setup Notion Integration

1. Go to [notion.so/my-integrations](https://www.notion.so/my-integrations)
2. Create a new integration
3. Copy the token → `export NOTION_TOKEN="secret_..."`
4. Share the parent page with your integration (click "..." → "Connections" → add your integration)

## License

MIT
