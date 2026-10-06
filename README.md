# notionctl

[![CI](https://github.com/radityajay/notionctl/actions/workflows/ci.yml/badge.svg)](https://github.com/radityajay/notionctl/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/radityajay/notionctl.svg)](https://pkg.go.dev/github.com/radityajay/notionctl)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

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
      total_estimate:
        type: rollup
        relation: tasks           # ← relation property in this database
        rollup_property: estimate  # ← property in Tasks database
        function: sum

  - name: Tasks
    properties:
      project:
        type: relation
        relation: Projects  # ← two-way relation, by name
      estimate:
        type: number
      double_estimate:
        type: formula
        expression: 'prop("estimate") * 2'
```

## Quick Start

```bash
# Install via Homebrew (macOS/Linux)
brew install radityajay/tap/notionctl

# Or install via Go
go install github.com/radityajay/notionctl@latest

# Or download binary from GitHub Releases
# https://github.com/radityajay/notionctl/releases

# Set your Notion integration token
export NOTION_TOKEN="secret_..."

# Option A: Start from scratch
notionctl init
# Edit notionctl.yaml — set your page IDs and customize properties

# Option B: Import existing databases from a Notion page
notionctl import --page-id YOUR_PAGE_ID

# Preview changes
notionctl plan

# Apply to Notion
notionctl apply
```

## Commands

| Command | Description |
|---------|-------------|
| `notionctl init` | Generate starter `notionctl.yaml` |
| `notionctl import` | Import existing Notion databases into config |
| `notionctl plan` | Preview changes (diff config vs state) |
| `notionctl apply` | Create/update/destroy databases in Notion |
| `notionctl apply --auto-approve` | Skip confirmation prompts for destructive actions |
| `notionctl diff` | Compare remote Notion databases against local config |
| `notionctl sync` | Pull current Notion state back into config YAML |
| `notionctl validate` | Validate config syntax without connecting to Notion |
| `notionctl version` | Print version |

Databases removed from your YAML but still in state will be shown as **destroy** actions in `plan`. On `apply`, each destroy prompts for confirmation (archives the database in Notion).

## Templates

Start from a pre-built template instead of from scratch:

```bash
cp templates/crm.yaml notionctl.yaml
# Edit parent_page_id, then:
notionctl plan
```

| Template | Databases | Use Case |
|----------|-----------|----------|
| [`crm.yaml`](templates/crm.yaml) | Contacts, Deals | Sales pipeline |
| [`inventory.yaml`](templates/inventory.yaml) | Products, Suppliers | Stock management |
| [`project-tracker.yaml`](templates/project-tracker.yaml) | Projects, Milestones, Tasks | Project management |
| [`bug-tracker.yaml`](templates/bug-tracker.yaml) | Bugs, Components, Releases | Issue / defect tracking |

Want to add a template? See [CONTRIBUTING.md](CONTRIBUTING.md) — no Go required!

## Supported Property Types

| Type | Description | Configurable |
|------|-------------|--------------|
| `title` | Database title column | — |
| `rich_text` | Text content | — |
| `number` | Numeric values | `format` (dollar, percent, etc.) |
| `select` | Single choice | `options` (name + color) |
| `multi_select` | Multiple choices | `options` (name + color) |
| `relation` | Link to another database | `relation` (target name), `synced_property` |
| `checkbox` | Boolean toggle | — |
| `date` | Date or date range | — |
| `url` | URL link | — |
| `email` | Email address | — |
| `phone_number` | Phone number | — |
| `formula` | Computed value from expression | `expression` |
| `rollup` | Aggregate through a relation | `relation`, `rollup_property`, `function` |
| `status` | Kanban-style status | `options` (name + color), `groups` |
| `created_time` | Creation timestamp (managed by Notion) | — |
| `last_edited_time` | Last edit timestamp (managed by Notion) | — |
| `people` | User references (workspace members) | — |
| `files` | File & media attachments | — |
| `unique_id` | Auto-generated sequential ID | `prefix` |

## How It Works

1. **Write YAML** — Declare databases with properties and relations by name
2. **`notionctl plan`** — Compares your YAML against local state (`.notionctl/state.json`)
3. **`notionctl apply`** — Creates/updates databases via Notion API, saves state
4. **`notionctl diff`** — Fetches remote databases and detects drift from manual edits

### Environment Variables in Config

Use `${VAR_NAME}` to reference environment variables in your YAML config. Useful for multi-environment setups:

```yaml
databases:
  - name: Projects
    parent_page_id: "${NOTION_PAGE_ID}"
    properties:
      Name:
        type: title
```

```bash
# Staging
NOTION_PAGE_ID=abc123 notionctl apply

# Production
NOTION_PAGE_ID=def456 notionctl apply
```

All `${VAR}` references must be set — `notionctl` will error if any are missing.

### Example: `notionctl plan`

```
Plan: 2 action(s)

+ create "Projects"
    parent_page_id: abc123
    + property "Name" (title)
    + property "Status" (status)
    + property "tasks" (relation)

+ create "Tasks"
    parent_page_id: abc123
    + property "Name" (title)
    + property "project" (relation)
    + property "estimate" (number)

Applying...

✓ create "Projects"
  → created with ID 1a2b3c
  → relations linked
✓ create "Tasks"
  → created with ID 4d5e6f
  → relations linked

Done. State saved to .notionctl/state.json
```

### Example: `notionctl diff`

```
✓ "Projects" — in sync

⚠ "Tasks" — drifted
    + property "Priority" (select): in config but not in Notion
    - property "OldField" (rich_text): in Notion but not in config
    ~ property "estimate": format config=dollar remote=number

○ "Archive" — not deployed
```

State is stored in `.notionctl/state.json` (add to `.gitignore` — it contains workspace-specific IDs).

## CI/CD with GitHub Actions

Automate your Notion database management in CI:

```yaml
# .github/workflows/notion-sync.yml
name: Sync Notion Databases
on:
  push:
    branches: [main]
    paths: ['notionctl.yaml']

jobs:
  sync:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - run: go install github.com/radityajay/notionctl@latest

      - name: Plan changes
        env:
          NOTION_TOKEN: ${{ secrets.NOTION_TOKEN }}
        run: notionctl plan

      - name: Apply changes
        env:
          NOTION_TOKEN: ${{ secrets.NOTION_TOKEN }}
        run: notionctl apply --auto-approve

      - name: Commit updated state
        run: |
          git config user.name "github-actions[bot]"
          git config user.email "github-actions[bot]@users.noreply.github.com"
          git add .notionctl/state.json
          git diff --staged --quiet || git commit -m "chore: update notionctl state"
          git push
```

See [`examples/`](examples/) for more CI/CD configurations.

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
