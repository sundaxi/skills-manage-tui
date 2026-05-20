# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.0] - 2026-07-09

### Added

- **Plugin (Marketplace) 管理** — 完整的插件安装/卸载系统，通过 Plugin Tab 操作
  - 从 GitHub 克隆 marketplace 仓库 (`a` 键添加)
  - 多平台安装选择 (`i` 键)
  - 一键卸载 (`u` 键) 和删除 (`x` 键)
  - 插件平台矩阵 — 每个 marketplace 显示在各平台的安装状态 (✓/·)
- **原生 CLI 安装** — 通过 `claude plugin` 和 `copilot plugin` 命令安装，确保完全兼容
- **Hermes 适配器** — 为不兼容的 Hermes 插件系统生成适配文件 (plugin.yaml + __init__.py)
- **状态栏消息** — install/uninstall/clone/delete 操作的彩色反馈 (✓ 绿色成功, ✗ 红色错误)
- **安装进度提示** — 安装过程中显示 "Installing... please wait" 覆盖层
- **`plugins_path` 配置** — 可自定义 marketplace 克隆目录
- **`commands_dir` 平台字段** — 支持平台自定义命令目录

### Changed

- Tab 结构: Collections → Plugin (第三个 Tab 变更为 Plugin 管理)
- 插件安装方式从手动文件写入改为原生 CLI 命令
- Clone timeout 增至 300 秒，支持大型 repo (如 ECC ~16MB)
- macOS 大小写不敏感文件系统保护 — 两步重命名防止误删

### Fixed

- 修复 SSH host key 验证失败 — 使用本地路径替代远程 URL
- 修复 clone timeout — 增加超时时间 + 部分 clone 清理
- 修复 macOS APFS 大小写冲突 — `strings.EqualFold` + temp 重命名
- 修复 Copilot 插件检测 — 检查 `installed-plugins/` 目录
- 修复 Claude 插件检测 — 检查 `installed_plugins.json`
- 修复 `extraKnownMarketplaces` 未写入 `settings.json`

### Removed

- 9 个死代码函数 (UnsymlinkPlugin, RecordInstalledPlugins 等)
- 旧 `test_copilot_install.go` 测试脚本

## [0.2.0] - 2025-07-17

### Added

- **平台安装矩阵** — Skills 列表内置每平台 ✓/· 安装状态列，可直观看到每个 skill 的覆盖情况
- **交互式 Settings Tab** — 可实时切换主题 (mocha/latte)、强调色 (10 种 Catppuccin 色)、编辑技能库路径
- **强调色系统** — 新增 `accent_color` 配置项，支持 rosewater, flamingo, pink, mauve, red, maroon, peach, yellow, green, teal
- **鼠标支持** — 单击 Tab 栏切换页面，单击技能行移动光标，双击技能行切换选中
- **快捷键 `o`** — 在 Finder (macOS) 或文件管理器 (Linux) 中打开当前技能目录
- **滚动翻页** — 技能列表超出屏幕时自动翻页，显示位置指示器 (如 `1-15 / 30`)
- **数字键快捷键** — `1`-`4` 直接跳转到对应 Tab

### Changed

- Tab 结构调整为 4 个: Skills / Marketplace / Collections / Settings (移除 Discover Tab)
- Mocha 主题色彩优化: 提高弱化文字对比度 (Overlay0 → Subtext0)，改善可读性
- Skills 列表增加 checkbox 与名称间距
- 项目重命名: skill-cli → skill-tui

### Removed

- 独立 Status Tab (功能已合并到 Skills 列表的平台矩阵)
- 独立 Discover Tab (功能可通过 CLI 命令 `skill-tui discover` 使用)

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
