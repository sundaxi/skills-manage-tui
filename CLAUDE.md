# skills-manage-tui

## Git Configuration

- **Committer:** 孙大喜 <sundaxi@users.noreply.github.com>
- Do NOT use PC hostname or default system username as committer name
- Always verify `git config user.name` and `git config user.email` before committing

## Project Info

- **Language:** Go 1.22+
- **Repo:** https://github.com/sundaxi/skills-manage-tui
- **License:** Apache License 2.0
- **Version:** 0.3.0 (Plugin management release)

## Development

```bash
make build       # Build binary
make test        # Run tests (14 tests across 2 packages)
make lint        # go vet + gofmt
make run         # Build and run
```

## Architecture Overview

```
cmd/             # Cobra commands (CLI entry points)
internal/
  config/        # Config loading (Viper + YAML)
  skill/         # Skill model + central registry
  platform/      # Platform detection + plugin install (CLI + adapters)
  plugin/        # Plugin (marketplace) management (clone, scan, parse)
  tui/           # Bubbletea interactive TUI (4 tabs)
  github/        # GitHub API client
  marketplace/   # Registry client + cache
  collection/    # Skill collection storage
  ai/            # LLM API explainer
  i18n/          # Internationalization (zh/en)
```

## Key Design Decisions

### Plugin Installation
- **Claude Code / Copilot:** Use native CLI commands (`claude plugin`, `copilot plugin`)
  - Pass local clone path (not URL) to avoid SSH failures and timeout
- **Hermes:** Adapter pattern — generate `plugin.yaml` + `__init__.py` since hermes has incompatible plugin format
- **Detection:** Platform-specific (Claude: installed_plugins.json, Copilot: installed-plugins/ dir, Hermes: parse CLI output)
- **Clone timeout:** 300 seconds, with partial clone cleanup on failure

### Skills
- Symlink-based distribution from central registry to platform dirs
- Single source of truth at `~/.agents/skills/`

### macOS Compatibility
- Case-insensitive APFS protection: two-step rename via temp directory
- `strings.EqualFold` for directory name comparisons

## Current State (v0.3.0)

### What Works
- ✅ Skill management (list/install/sync/remove via CLI and TUI)
- ✅ Plugin management (clone/install/uninstall/delete via TUI)
- ✅ 3 plugin platforms: Claude Code, Copilot, Hermes
- ✅ Status bar with success/error messages
- ✅ 14 tests passing, 0 dead code

### What's Planned
- 📋 Codex-cli plugin support
- 📋 Marketplace tab (browse/search public plugins)
- 📋 Collection management
- 📋 Plugin update mechanism
- 📋 Error retry / partial success reporting (per-platform)

### Key Files
- `internal/platform/platform_cli.go` — CLI install/uninstall routing + hermes adapter
- `internal/platform/platform.go` — Platform types, detection, install helpers
- `internal/plugin/plugin.go` — Store, clone, marketplace parsing
- `internal/tui/app.go` — Main TUI model (~1600 lines)
- `internal/tui/components/statusbar.go` — Status bar with message support
- `plugin-install.md` — Detailed design doc (platform comparison, architecture)
