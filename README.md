# skills-manage-tui

> A CLI tool for managing AI coding agent skills across multiple platforms
>
> [中文文档](README_zh.md)

> **Note:** The author is lazy. This project is unlikely to receive further updates. Feel free to fork and carry it forward.

## Development Status

| Feature | Status |
|---------|--------|
| Skill management (list / install / sync / remove) | **Available** |
| Interactive TUI | **Available** |
| Multi-platform symlink | **Available** |
| Marketplace | Coming soon |
| Collection | Coming soon |
| GitHub import | Coming soon |
| AI explain | Coming soon |
| Local discover | Coming soon |

## Features

- **Central skill registry** — `~/.agents/skills/` as single source of truth, distributed via symlinks
- **28+ platforms** — Claude Code, Cursor, Gemini CLI, Copilot, Windsurf, Aider, etc.
- **Interactive TUI** — Multi-select, real-time search, Markdown detail view, Catppuccin theme, mouse support
- **Platform matrix** — Per-skill install status across all platforms (checkmark / dot)
- **Auto scroll/pagination** — Automatic scrolling when skills exceed screen height
- **Bilingual** — Auto-detects system language (Chinese / English)

## Quick Start

### Install

```bash
git clone https://github.com/sundaxi/skills-manage-tui.git
cd skills-manage-tui
make build
```

### Prerequisites

- Go 1.22+

### 30-Second Tour

```bash
# Launch interactive TUI
skill-tui

# Or use CLI mode
skill-tui list                          # List all skills
skill-tui list --verbose                # Verbose (with platform install info)
skill-tui list --platform claude-code   # Filter by platform

skill-tui install my-skill              # Choose target platforms
skill-tui install my-skill --all        # Install to all detected platforms
skill-tui install my-skill -p cursor    # Install to specific platform

skill-tui sync                          # Sync all platforms
skill-tui sync --dry-run                # Preview changes

skill-tui remove my-skill               # Remove from all platforms
skill-tui remove my-skill --purge       # Also delete from central registry
```

## Command Reference

### `skill-tui` (Interactive Mode)

Launches the TUI with four tabs:

```
 1 Skills   2 Marketplace   3 Collections   4 Settings
```

**Keyboard shortcuts (Skills tab):**

| Key | Action |
|-----|--------|
| Up/k  Down/j | Navigate |
| Space | Toggle selection |
| a | Select / deselect all |
| Enter / d | View detail |
| o | Open skill directory in Finder |
| p | Install selected to chosen platforms |
| x | Remove selected |
| / | Search |
| r | Refresh |
| i | Install (in detail view) |
| u | Uninstall (in detail view) |
| Esc | Go back |
| Tab | Switch tab |
| 1-4 | Jump to tab |
| q | Quit |

**Mouse support:**

| Action | Effect |
|--------|--------|
| Click tab bar | Switch to that tab |
| Click skill row | Move cursor to that row |
| Double-click skill row | Toggle selection |

**Platform matrix in Skills tab:**

Each skill row shows install status across platforms:

```
                         claude  codex   copilot hermes
──────────────────────────────────────────────────────────
○ ai-digest               ✓       ✓       ✓       ✓
○ mermaid-visualizer      ✓       ·       ✓       ✓
```

**Auto pagination:**

When the skill list exceeds screen height, scrolling activates automatically with a position indicator at the bottom (e.g. `─── 1-15 / 30 ───`).

### `skill-tui list`

```bash
skill-tui list                          # Basic list
skill-tui list -v                       # Verbose (author, platform count)
skill-tui list -p claude-code           # Filter by platform
```

### `skill-tui install`

Install a skill to one or more platforms via symlink creation.

```bash
skill-tui install my-skill              # Choose target platforms
skill-tui install my-skill --all        # All detected platforms
skill-tui install my-skill -p cursor    # Specific platform
```

### `skill-tui sync`

Sync the central registry with all platform symlinks.

```bash
skill-tui sync                          # Sync all platforms
skill-tui sync -p claude-code           # Specific platform
skill-tui sync --dry-run                # Preview only
```

Auto-fixes:
- Creates missing symlinks
- Repairs broken links (pointing to deleted skills)
- Cleans up stale links

### `skill-tui remove`

Remove a skill from platforms (deletes symlinks).

```bash
skill-tui remove my-skill               # From all platforms (confirm)
skill-tui remove my-skill -p cursor     # From specific platform
skill-tui remove my-skill --purge       # Also delete from central registry
skill-tui remove my-skill --force       # Skip confirmation
```

### `skill-tui config`

View and modify configuration.

```bash
skill-tui config get                    # Show all config
skill-tui config get theme              # Show single value
skill-tui config set theme latte        # Switch theme
skill-tui config platforms              # List all known platforms
```

## Configuration

Config file: `~/.skill-tui/config.yaml`. Environment variables override with `SKILL_CLI_` prefix.

| Key | Default | Description |
|-----|---------|-------------|
| `skills_path` | `~/.agents/skills/` | Central skill registry path |
| `theme` | `mocha` | TUI theme (`mocha` / `latte`) |
| `accent_color` | `mauve` | 10 Catppuccin accent colors |
| `language` | `auto` | Language (`auto` / `zh` / `en`) |
| `github_token` | (empty) | GitHub API token (higher rate limit) |

## Supported Platforms

| Category | Platform | Skills Directory |
|----------|----------|-----------------|
| Coding | Claude Code | `~/.claude/skills/` |
| Coding | Codex CLI | `~/.agents/skills/` |
| Coding | Cursor | `~/.cursor/skills/` |
| Coding | Gemini CLI | `~/.gemini/skills/` |
| Coding | Copilot | `~/.copilot/skills/` |
| Coding | Windsurf | `~/.windsurf/skills/` |
| Coding | Aider | `~/.aider/skills/` |
| Coding | Augment | `~/.augment/skills/` |
| Coding | Trae | `~/.trae/skills/` |
| Coding | Hermes | `~/.hermes/skills/` |
| Coding | Factory Droid | `~/.factory/skills/` |
| Coding | Junie | `~/.junie/skills/` |
| Coding | KiloCode | `~/.kilocode/skills/` |
| Coding | OpenCode | `~/.opencode/skills/` |
| Coding | Amp | `~/.amp/skills/` |
| Coding | Kiro | `~/.kiro/skills/` |
| Lobster | OpenClaw | `~/.openclaw/skills/` |
| Lobster | QClaw | `~/.qclaw/skills/` |
| Lobster | EasyClaw | `~/.easyclaw/skills/` |
| Lobster | WorkBuddy | `~/.workbuddy/skills-marketplace/skills/` |

Custom platforms can be added via Settings or by editing `configs/platforms.yaml`.

## How It Works

```
Central Registry                    Platform Skills Directories
~/.agents/skills/                   ~/.claude/skills/
├── skill-a/                        ├── skill-a → ~/.agents/skills/skill-a
│   └── SKILL.md                    └── skill-b → ~/.agents/skills/skill-b
└── skill-b/
    └── SKILL.md              ~/.cursor/skills/
                              ├── skill-a → ~/.agents/skills/skill-a
                              └── skill-b → ~/.agents/skills/skill-b
```

1. Skills live in the central registry (default `~/.agents/skills/`)
2. Installing creates a symlink in the platform's skills directory
3. One source file is shared across all platforms
4. Editing the central copy is instantly reflected everywhere

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.22+ |
| CLI | cobra |
| TUI | bubbletea + lipgloss + huh + bubbles |
| Markdown | glamour |
| Config | viper |
| Theme | Catppuccin (Mocha / Latte) |

## Development

```bash
make build       # Build binary
make test        # Run tests
make lint        # go vet + gofmt
make run         # Build and run
make install     # Install to /usr/local/bin
make clean       # Remove binary
```

## Acknowledgements

Inspired by [skills-manage](https://github.com/iamzhihuix/skills-manage) — a Tauri-based desktop skill manager.

## License

Apache License 2.0
