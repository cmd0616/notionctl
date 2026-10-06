---
title: "Managing Notion Databases as Code with notionctl"
published: false
description: "Define Notion databases in YAML, sync with one command. Like Terraform, but for Notion."
tags: notion, go, devtools, automation
---

# Managing Notion Databases as Code with notionctl

If you've ever tried to automate Notion databases — setting up relations, managing IDs across workspaces, or keeping databases consistent across environments — you know the pain. Database IDs change when you copy, differ between workspaces, and are impossible to read in scripts.

I built **notionctl** to fix this: declare your databases in YAML, reference relations by name, and sync everything with one command.

## The Problem

Let's say you have a project tracker with Projects, Milestones, and Tasks databases — all linked by relations. Setting this up via the Notion API means:

1. Creating each database with raw JSON payloads
2. Storing UUIDs to wire up relations
3. Handling the chicken-and-egg problem (you need Database B's ID to create Database A's relation, but B doesn't exist yet)
4. Manually tracking what's deployed vs. what's changed

It's painful, error-prone, and impossible to version control.

## The Solution: Declare, Don't Script

With notionctl, you write YAML:

```yaml
version: "1"
databases:
  - name: Projects
    parent_page_id: "abc123..."
    properties:
      Name:
        type: title
      Status:
        type: status
        options:
          - name: Not Started
            color: default
          - name: In Progress
            color: blue
          - name: Done
            color: green
      tasks:
        type: relation
        relation: Tasks    # ← by name, not UUID
      total_estimate:
        type: rollup
        relation: tasks
        rollup_property: estimate
        function: sum

  - name: Tasks
    parent_page_id: "abc123..."
    properties:
      Name:
        type: title
      project:
        type: relation
        relation: Projects  # ← two-way relation
      estimate:
        type: number
        format: number
```

Then run:

```bash
$ notionctl plan

Plan: 2 action(s)

+ create "Projects"
    + property "Name" (title)
    + property "Status" (status)
    + property "tasks" (relation)
    + property "total_estimate" (rollup)

+ create "Tasks"
    + property "Name" (title)
    + property "project" (relation)
    + property "estimate" (number)

$ notionctl apply

✓ create "Projects"
  → created with ID 1a2b3c
  → relations linked
✓ create "Tasks"
  → created with ID 4d5e6f
  → relations linked

Done. State saved to .notionctl/state.json
```

notionctl handles the two-pass relation resolution automatically — no UUID juggling required.

## Key Features

### 19 Property Types (100% Coverage)

Every Notion property type is supported: title, rich_text, number, select, multi_select, status, relation, rollup, formula, checkbox, date, url, email, phone_number, created_time, last_edited_time, people, files, and unique_id.

### Drift Detection

Someone edited a database directly in Notion? Detect it:

```bash
$ notionctl diff

✓ "Projects" — in sync

⚠ "Tasks" — drifted
    + property "Priority" (select): in config but not in Notion
    - property "OldField" (rich_text): in Notion but not in config
```

Then pull the changes back:

```bash
$ notionctl sync
✓ Synced 2 database(s) from Notion → notionctl.yaml

Review changes with 'git diff' before committing.
```

### CI/CD Ready

Automate with GitHub Actions:

```yaml
on:
  push:
    branches: [main]
    paths: ['notionctl.yaml']
jobs:
  sync:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: brew install radityajay/tap/notionctl
      - run: notionctl apply --auto-approve
        env:
          NOTION_TOKEN: ${{ secrets.NOTION_TOKEN }}
```

### Templates

Start fast with pre-built templates:

```bash
cp templates/project-tracker.yaml notionctl.yaml
# Edit parent_page_id, then:
notionctl plan && notionctl apply
```

Available: CRM, inventory, project tracker, bug tracker. Or [contribute your own](https://github.com/radityajay/notionctl/blob/main/CONTRIBUTING.md) — no Go required!

## Install

```bash
# Homebrew
brew install radityajay/tap/notionctl

# Go
go install github.com/radityajay/notionctl@latest

# Binary
# Download from https://github.com/radityajay/notionctl/releases
```

## Links

- **GitHub**: [github.com/radityajay/notionctl](https://github.com/radityajay/notionctl)
- **Releases**: [github.com/radityajay/notionctl/releases](https://github.com/radityajay/notionctl/releases)

---

notionctl is MIT-licensed and open for contributions. If you have a Notion database setup you use often, consider [adding it as a template](https://github.com/radityajay/notionctl/issues/15) — it's a single YAML file, no Go required.
