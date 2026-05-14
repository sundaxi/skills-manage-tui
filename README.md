# skill-tui

> 一个用于统一管理多平台 AI 编程 Agent Skills 的命令行工具
>
> A CLI tool for managing AI coding agent skills across multiple platforms

## 功能概览

- **中央技能库** — 使用 `~/.agents/skills/` 作为单一事实来源，通过符号链接分发到各平台
- **28+ 平台支持** — Claude Code、Cursor、Gemini CLI、Copilot、Windsurf、Aider 等
- **交互式 TUI** — 多选列表、实时搜索、Markdown 详情、Catppuccin 主题、鼠标点击/双击
- **平台矩阵** — Skills 列表内置各平台安装状态矩阵 (✓/·)
- **Marketplace** — 浏览和安装社区发布的 Skills
- **GitHub 导入** — 从任意 GitHub 仓库导入 Skills
- **技能集合** — 批量安装和分组管理
- **AI 解释** — 调用 LLM 自动解释 Skill 内容
- **本地发现** — 扫描项目目录发现已安装的 Skills
- **中英双语** — 自动检测系统语言

## 快速开始

### 安装

```bash
# 从源码构建
git clone https://github.com/ying-sun1/skill-tui.git
cd skill-tui
make install

# 或直接构建
make build
./skill-tui --help
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

skill-tui install my-skill              # 安装到所有已检测平台
skill-tui install my-skill -p cursor    # 安装到指定平台

skill-tui sync                          # 同步所有平台
skill-tui sync --dry-run                # 预览变更

skill-tui remove my-skill               # 从所有平台移除
skill-tui remove my-skill --purge       # 同时从中央库删除
```

## 命令参考

### `skill-tui` (交互模式)

不带参数运行进入 TUI 界面：

```
 1 Skills   2 Marketplace   3 Collections   4 Settings
```

| 快捷键 | 功能 |
|--------|------|
| `↑`/`k` `↓`/`j` | 上下导航 |
| `Space` | 选择/取消当前项 |
| `a` | 全选/取消全选 |
| `Enter` / `d` | 查看详情 |
| `p` | 批量安装到选定平台 |
| `x` | 移除选中项 |
| `/` | 搜索 |
| `r` | 刷新 |
| `i` | 在详情页安装到所有平台 |
| `u` | 在详情页卸载 |
| `Esc` | 返回上级 |
| `Tab` | 切换 Tab 页 |
| `1`-`4` | 直接跳转到指定 Tab |
| `q` | 退出 |

**鼠标支持：**

| 操作 | 功能 |
|------|------|
| 单击 Tab 栏 | 切换到对应 Tab |
| 单击技能行 | 移动光标到该行 |
| 双击技能行 | 切换选中状态 |

**Skills Tab 平台矩阵：**

Skills 列表内置每平台安装状态矩阵，以 `✓`/`·` 标记各技能在各平台的安装状态：

```
                         claude  codex   copilot hermes
───────────────────────────────────────────────────────
○ ai-digest               ✓       ✓       ✓       ✓
○ mermaid-visualizer      ✓       ·       ✓       ✓
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
skill-tui install my-skill              # 显示可用平台列表
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

### `skill-tui marketplace`

浏览和安装市场中的 Skills。

```bash
skill-tui marketplace browse            # 浏览发布者
skill-tui marketplace search react      # 搜索 Skills
skill-tui marketplace install graphify  # 安装一个 Skill
```

### `skill-tui import`

从 GitHub 仓库导入 Skills。

```bash
skill-tui import https://github.com/user/skills-repo
skill-tui import https://github.com/user/repo --path skills/my-skill
```

自动扫描仓库中的 `SKILL.md` 文件并导入到中央技能库。

### `skill-tui collection`

管理技能集合（批量操作）。

```bash
skill-tui collection create my-set -d "My常用集合" --skills skill1,skill2
skill-tui collection list
skill-tui collection add my-set new-skill
skill-tui collection remove my-set old-skill
skill-tui collection install my-set        # 批量安装
skill-tui collection delete my-set
```

### `skill-tui discover`

扫描本地目录发现项目级 Skills。

```bash
skill-tui discover                       # 扫描当前目录
skill-tui discover /path/to/project      # 扫描指定目录
skill-tui discover -r                    # 递归扫描子目录
```

### `skill-tui config`

查看和修改配置。

```bash
skill-tui config get                     # 查看所有配置
skill-tui config get theme               # 查看单项
skill-tui config set theme latte         # 切换主题
skill-tui config set github_token ghp_xxx
skill-tui config set ai_provider anthropic
skill-tui config set ai_key sk-ant-xxx
skill-tui config platforms               # 列出所有已知平台及检测状态
```

## 配置

配置文件位于 `~/.skill-tui/config.yaml`，支持环境变量覆盖（`SKILL_CLI_` 前缀）。

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `skills_path` | `~/.agents/skills/` | 中央技能库路径 |
| `theme` | `mocha` | TUI 主题 (`mocha` / `latte`) |
| `accent_color` | `mauve` | 强调色 (10 种 Catppuccin 色: rosewater, flamingo, pink, mauve, red, maroon, peach, yellow, green, teal) |
| `language` | `auto` | 语言 (`auto` / `zh` / `en`) |
| `github_token` | (空) | GitHub API Token（提高 rate limit） |
| `ai_provider` | (空) | AI 提供商 (`openai` / `anthropic`) |
| `ai_key` | (空) | AI API Key |
| `ai_endpoint` | (空) | 自定义 AI Endpoint |

### 环境变量

```bash
export SKILL_CLI_SKILLS_PATH=/custom/skills/path
export SKILL_CLI_GITHUB_TOKEN=ghp_xxx
export SKILL_CLI_AI_KEY=sk-ant-xxx
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
| Central | 中央技能库 | (configurable) |

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
| HTTP 客户端 | net/http |
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
