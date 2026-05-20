# skills-manage-tui

[![English](https://img.shields.io/badge/English-lightgrey?style=flat-square&labelColor=2196F3&color=2196F3&label=%F0%9F%87%AC%F0%9F%87%A7&link=README.md)](README.md) [![中文](https://img.shields.io/badge/%E4%B8%AD%E6%96%87-2196F3?style=flat-square&labelColor=E91E63&color=E91E63&label=%F0%9F%87%A8%F0%9F%87%B3&link=README_zh.md)](README_zh.md)

> 一个用于统一管理多平台 AI 编程 Agent Skills 和 Plugins 的命令行工具

## 开发状态

| 功能 | 状态 |
|------|------|
| Skill 管理（list / install / sync / remove） | **已可用** |
| Plugin 管理（marketplace 安装/卸载） | **已可用** |
| 交互式 TUI | **已可用** |
| 多平台符号链接 | **已可用** |
| Marketplace（技能市场） | 开发中 |
| Collection（技能集合） | 开发中 |
| GitHub 导入 | 开发中 |
| AI 解释 | 开发中 |
| 本地发现 | 开发中 |

## 功能概览

- **中央技能库** — 使用 `~/.agents/skills/` 作为单一事实来源，通过符号链接分发到各平台
- **Plugin 管理** — 通过原生 CLI 命令安装/卸载 marketplace 插件 (Claude Code, Copilot, Hermes)
- **28+ 平台支持** — Claude Code、Cursor、Gemini CLI、Copilot、Windsurf、Aider 等
- **交互式 TUI** — 多选列表、实时搜索、Markdown 详情、Catppuccin 主题、鼠标点击/双击
- **平台矩阵** — Skills 列表内置各平台安装状态矩阵 (✓/·)
- **状态栏反馈** — 所有操作的彩色成功/错误提示
- **自动翻页** — 技能数量超出屏幕时自动滚动，底部显示位置指示器
- **中英双语** — 自动检测系统语言

## 快速开始

### 安装

```bash
git clone https://github.com/sundaxi/skills-manage-tui.git
cd skills-manage-tui
make build
```

### 前置依赖

- Go 1.22+

### 30 秒上手

```bash
# 进入交互式 TUI
skill-tui

# 或使用命令模式
skill-tui list                          # 列出所有 skills
skill-tui list --verbose                # 详细信息（含已安装平台）
skill-tui list --platform claude-code   # 查看特定平台的 skills

skill-tui install my-skill              # 选择目标平台
skill-tui install my-skill --all        # 安装到所有已检测平台
skill-tui install my-skill -p cursor    # 安装到指定平台

skill-tui sync                          # 同步所有平台
skill-tui sync --dry-run                # 预览变更

skill-tui remove my-skill               # 从所有平台移除
skill-tui remove my-skill --purge       # 同时从中央库删除
```

## 命令参考

### `skill-tui`（交互模式）

不带参数运行进入 TUI 界面，包含四个 Tab 页：

```
 1 Skills   2 Marketplace   3 Plugin   4 Settings
```

**键盘快捷键（Skills Tab）：**

| 快捷键 | 功能 |
|--------|------|
| ↑/k  ↓/j | 上下导航 |
| Space | 选择/取消当前项 |
| a | 全选/取消全选 |
| Enter / d | 查看详情 |
| o | 在 Finder 中打开技能目录 |
| p | 批量安装到选定平台 |
| x | 移除选中项 |
| / | 搜索 |
| r | 刷新 |
| i | 在详情页安装到所有平台 |
| u | 在详情页卸载 |
| Esc | 返回上级 |
| Tab | 切换 Tab 页 |
| 1-4 | 直接跳转到指定 Tab |
| q | 退出 |

**鼠标支持：**

| 操作 | 功能 |
|------|------|
| 单击 Tab 栏 | 切换到对应 Tab |
| 单击技能行 | 移动光标到该行 |
| 双击技能行 | 切换选中状态 |

**Skills Tab 平台矩阵：**

Skills 列表内置每平台安装状态矩阵，以 ✓/· 标记：

```
                         claude  codex   copilot hermes
──────────────────────────────────────────────────────────
○ ai-digest               ✓       ✓       ✓       ✓
○ mermaid-visualizer      ✓       ·       ✓       ✓
```

**自动翻页：**

技能数量超出屏幕高度时，列表自动支持滚动翻页。光标移动到边界时视窗自动跟随，底部显示位置指示器（如 `─── 1-15 / 30 ───`）。

**键盘快捷键（Plugin Tab）：**

| 快捷键 | 功能 |
|--------|------|
| ↑/k  ↓/j | 上下导航 |
| Enter / d | 查看插件详情 |
| a | 添加 marketplace (GitHub URL 或 owner/repo) |
| i | 安装到选定平台 |
| u | 从所有平台卸载 |
| x | 删除 marketplace（卸载 + 删除 clone） |
| r | 刷新 |
| Esc | 返回上级 |

**Plugin 平台矩阵：**

```
                         claude  copilot hermes
──────────────────────────────────────────────────
  ECC                     ✓       ✓       ✓
  CLI-Anything            ✓       ·       ✓
```

### `skill-tui list`

列出中央技能库中的所有 Skills。

```bash
skill-tui list                          # 基础列表
skill-tui list -v                       # 详细模式（显示作者、已安装平台数）
skill-tui list -p claude-code           # 仅显示某平台已安装的 skills
```

### `skill-tui install`

将 Skill 安装到一个或多个平台。通过创建符号链接实现。

```bash
skill-tui install my-skill              # 选择目标平台
skill-tui install my-skill --all        # 安装到所有已检测平台
skill-tui install my-skill -p cursor    # 安装到指定平台
```

### `skill-tui sync`

同步中央技能库和各平台的符号链接。

```bash
skill-tui sync                          # 同步所有平台
skill-tui sync -p claude-code           # 仅同步指定平台
skill-tui sync --dry-run                # 预览模式（不执行变更）
```

自动修复：
- 创建缺失的符号链接
- 修复断裂的链接（指向已删除的 skill）
- 清理指向已移除 skill 的过期链接

### `skill-tui remove`

从平台移除 Skill（删除符号链接）。

```bash
skill-tui remove my-skill               # 从所有平台移除（需确认）
skill-tui remove my-skill -p cursor     # 从指定平台移除
skill-tui remove my-skill --purge       # 同时从中央库删除
skill-tui remove my-skill --force       # 跳过确认提示
```

### `skill-tui config`

查看和修改配置。

```bash
skill-tui config get                    # 查看所有配置
skill-tui config get theme              # 查看单项
skill-tui config set theme latte        # 切换主题
skill-tui config platforms              # 列出所有已知平台及检测状态
```

## 配置

配置文件位于 `~/.skill-tui/config.yaml`，支持环境变量覆盖（`SKILL_CLI_` 前缀）。

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `skills_path` | `~/.agents/skills/` | 中央技能库路径 |
| `plugins_path` | `~/.agents/Plugins/` | Plugin marketplace clone 目录 |
| `theme` | `mocha` | TUI 主题 (`mocha` / `latte`) |
| `accent_color` | `mauve` | 强调色 (10 种 Catppuccin 色) |
| `language` | `auto` | 语言 (`auto` / `zh` / `en`) |
| `github_token` | (空) | GitHub API Token（提高 rate limit） |

### 环境变量

```bash
export SKILL_CLI_SKILLS_PATH=/custom/skills/path
export SKILL_CLI_GITHUB_TOKEN=ghp_xxx
```

## 支持的平台

| 类别 | 平台 | Skills 目录 |
|------|------|------------|
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

可在 Settings 界面或通过修改 `configs/platforms.yaml` 添加自定义平台。

## 工作原理

```
中央技能库                          各平台 Skills 目录
~/.agents/skills/                   ~/.claude/skills/
├── skill-a/                        ├── skill-a → ~/.agents/skills/skill-a
│   └── SKILL.md                    └── skill-b → ~/.agents/skills/skill-b
└── skill-b/
    └── SKILL.md              ~/.cursor/skills/
                              ├── skill-a → ~/.agents/skills/skill-a
                              └── skill-b → ~/.agents/skills/skill-b
```

1. Skill 存放在中央技能库（默认 `~/.agents/skills/`）
2. 通过符号链接安装到各平台的 skills 目录
3. 同一份 Skill 是所有平台的单一事实来源
4. 修改中央库的 Skill 即时反映到所有平台

## 技术栈

| 层 | 技术 |
|---|---|
| 语言 | Go 1.22+ |
| CLI 框架 | cobra |
| TUI 框架 | bubbletea + lipgloss + huh + bubbles |
| Markdown 渲染 | glamour |
| 配置管理 | viper |
| 主题 | Catppuccin (Mocha / Latte) |

## 开发

```bash
make build       # 构建
make test        # 运行测试
make lint        # 代码检查 + 格式化
make run         # 构建并运行
make install     # 安装到 /usr/local/bin
make clean       # 清理构建产物
```

## 致谢

灵感来源 [skills-manage](https://github.com/iamzhihuix/skills-manage) — 基于 Tauri 的桌面版 Skill 管理器。

## License

Apache License 2.0
