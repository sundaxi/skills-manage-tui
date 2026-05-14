# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-05-13

### Added

**核心功能**
- 中央技能库管理 (`~/.agents/skills/`)，支持自定义路径
- 28+ AI 编程平台定义 (Claude Code, Cursor, Gemini CLI, Copilot, Windsurf, Aider, Augment, Trae, Hermes, Kiro, KiloCode 等)
- 符号链接安装/卸载机制，实现单一事实来源
- SKILL.md YAML frontmatter 元数据解析

**CLI 命令**
- `skill-tui list` — 列出 Skills，支持 `--verbose` 和 `--platform` 过滤
- `skill-tui install` — 安装到指定平台 (`--platform`) 或所有平台 (`--all`)
- `skill-tui sync` — 全量同步符号链接，`--dry-run` 预览，自动修复断裂链接
- `skill-tui remove` — 从平台移除，`--purge` 从中央库删除，`--force` 跳过确认
- `skill-tui config get/set/platforms` — 配置管理
- `skill-tui import <github-url>` — 从 GitHub 仓库导入 Skills，`--path` 指定子目录
- `skill-tui marketplace browse/search/install` — 技能市场浏览和安装
- `skill-tui collection create/list/delete/install/add/remove` — 技能集合管理
- `skill-tui discover` — 本地目录扫描，`--recursive` 递归扫描

**TUI 交互模式**
- 5 Tab 页导航: Skills / Marketplace / Collections / Discover / Settings
- 多选列表 (Space 选择, a 全选)
- 实时搜索过滤 (/)
- Skill 详情页 (Markdown 内容, 平台安装状态)
- 平台选择器 (p 键批量安装)
- 底部状态栏 (Skills 数, 平台数, 路径)
- Catppuccin Mocha / Latte 双主题

**基础设施**
- GitHub REST API v3 客户端 (Tree/Contents, Token 认证)
- Marketplace JSON Registry + 内存缓存 (10 min TTL)
- Skill Collection JSON 存储
- 本地技能发现扫描器 (15 种平台目录)
- AI 解释器 (OpenAI / Anthropic API, 本地缓存)
- i18n 中英双语支持 (LANG 自动检测)
- Viper 配置管理 (YAML + 环境变量)
- Makefile (build/test/lint/install/clean)
