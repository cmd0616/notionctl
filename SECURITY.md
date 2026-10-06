# Security Policy

## Supported Versions

| Version | Supported          |
|---------|--------------------|
| 1.x     | ✅ Yes             |
| < 1.0   | ❌ No              |

## Reporting a Vulnerability

If you discover a security vulnerability in notionctl, please report it responsibly.

**Do not open a public issue.**

Instead, email **radityajay@users.noreply.github.com** with:

1. Description of the vulnerability
2. Steps to reproduce
3. Impact assessment (if known)

You should receive an acknowledgment within 48 hours. We will work with you to understand and address the issue before any public disclosure.

## Scope

notionctl interacts with the Notion API using user-provided tokens (`NOTION_TOKEN`). Security concerns include:

- **Token handling** — tokens are read from environment variables, never stored in config or state files
- **State file** — `.notionctl/state.json` contains database IDs (not secrets), but should be in `.gitignore` for workspace-specific deployments
- **Network** — all API calls go to `api.notion.com` over HTTPS

## Best Practices

- Never commit `NOTION_TOKEN` to version control
- Add `.notionctl/state.json` to `.gitignore` if your state contains workspace-specific IDs
- Use environment variables or CI secrets for tokens in automation
