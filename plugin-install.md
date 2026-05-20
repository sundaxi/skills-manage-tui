# Plugin 安装机制设计文档

> 最后更新: 2026-07-09  
> 基于 Claude Code、Copilot CLI 和 Hermes 的实际文件系统分析和 CLI 验证

## 一、平台插件系统对比

### Claude Code (`~/.claude/`)

**目录结构:**
```
~/.claude/
├── settings.json                        ← 用户配置 (enabledPlugins, extraKnownMarketplaces)
├── plugins/
│   ├── marketplaces/                    ← marketplace 仓库的 clone/symlink
│   │   ├── claude-plugins-official/     ← 自动添加的官方 marketplace
│   │   ├── ecc/                         ← 第三方 marketplace
│   │   └── cli-anything -> /path/to/src ← skill-tui 创建的 symlink
│   ├── cache/                           ← 已安装 plugin 的版本化副本
│   │   └── <marketplace>/<plugin>/<version>/
│   ├── data/                            ← plugin 持久数据 (CLAUDE_PLUGIN_DATA)
│   │   └── <plugin-id>/
│   ├── installed_plugins.json           ← 安装记录 (v2 格式)
│   ├── known_marketplaces.json          ← marketplace 注册表
│   └── plugin-catalog-cache.json        ← 插件目录缓存
├── skills/                              ← 全局用户 skills
├── commands/                            ← 全局 slash commands
└── agents/                              ← 全局 agents
```

**installed_plugins.json** (v2 格式):
```json
{
  "version": 2,
  "plugins": {
    "ecc@ecc": [
      {
        "scope": "user",
        "installPath": "/Users/xxx/.claude/plugins/cache/ecc/ecc/c2471fe5c535",
        "version": "c2471fe5c535",
        "installedAt": "2026-05-20T02:41:08.027Z",
        "lastUpdated": "2026-05-20T02:41:08.027Z",
        "gitCommitSha": "c2471fe5c535310f8a8008c9ed7ea9f6757b33f2"
      }
    ]
  }
}
```

**known_marketplaces.json**:
```json
{
  "ecc": {
    "source": { "source": "github", "repo": "affaan-m/everything-claude-code" },
    "installLocation": "/Users/xxx/.claude/plugins/marketplaces/ecc",
    "lastUpdated": "2026-05-20T02:43:14.940Z"
  }
}
```

**settings.json** (plugin 相关键):
```json
{
  "enabledPlugins": {
    "ecc@ecc": true,
    "cli-anything@cli-anything": true
  },
  "extraKnownMarketplaces": {
    "ecc": {
      "source": { "repo": "affaan-m/everything-claude-code", "source": "github" }
    }
  }
}
```

**安装命令:** `claude plugin install <name>@<marketplace>` 或 `/plugin install`

**plugin.json 发现顺序:** `.claude-plugin/plugin.json`

**marketplace.json 发现顺序:** `.claude-plugin/marketplace.json`

---

### Copilot CLI (`~/.copilot/`)

**⚠️ Copilot 的文件布局与 Claude Code 有显著差异**

**目录结构:**
```
~/.copilot/
├── settings.json                        ← 用户配置 (enabledPlugins, extraKnownMarketplaces)
├── config.json                          ← 自动管理 (installedPlugins[], trustedFolders)
├── installed-plugins/                   ← 已安装 plugin (文件拷贝, 无版本子目录)
│   └── <marketplace>/<plugin>/
├── plugin-data/                         ← plugin 持久数据 (COPILOT_PLUGIN_DATA)
│   └── <marketplace>/<plugin>/
├── skills/                              ← 全局用户 skills
├── agents/                              ← 全局 agents
├── hooks/                               ← 用户 hook 脚本
├── mcp-config.json                      ← MCP 服务器配置
└── lsp-config.json                      ← LSP 服务器配置

~/Library/Caches/copilot/                ← macOS 系统缓存目录 (不在 ~/.copilot/ 内!)
└── marketplaces/                        ← marketplace 仓库缓存
    ├── HKUDS-CLI-Anything/
    └── affaan-m-everything-claude-code/
```

**config.json** (自动管理, 开头有注释行):
```jsonc
// User settings belong in settings.json.
// This file is managed automatically.
{
  "firstLaunchAt": "2026-04-09T07:09:55.651Z",
  "trustedFolders": ["/Users/xxx"],
  "loggedInUsers": [...],
  "installedPlugins": [
    {
      "name": "cli-anything",
      "marketplace": "cli-anything",
      "version": "1.0.0",
      "installed_at": "2026-05-19T09:58:27.321Z",
      "enabled": true,
      "cache_path": "/Users/xxx/.copilot/installed-plugins/cli-anything/cli-anything"
    }
  ]
}
```

**settings.json** (用户可编辑):
```json
{
  "enabledPlugins": {
    "cli-anything@cli-anything": true
  },
  "extraKnownMarketplaces": {
    "cli-anything": {
      "source": { "source": "github", "repo": "HKUDS/CLI-Anything" }
    }
  }
}
```

**安装命令:** `copilot plugin install <name>@<marketplace>` 或 `/plugin install`

**plugin.json 发现顺序:** `.plugin/plugin.json` > `plugin.json` > `.github/plugin/plugin.json` > `.claude-plugin/plugin.json`

**marketplace.json 发现顺序:** `.github/plugin/marketplace.json` > `marketplace.json` > `.plugin/marketplace.json` > `.claude-plugin/marketplace.json`

**默认 marketplace:** `copilot-plugins` (GitHub 官方) + `awesome-copilot` (社区)

---

### Hermes (`~/.hermes/`)

**⚠️ Hermes 使用完全不兼容的插件系统 (Python-based)**

**目录结构:**
```
~/.hermes/
├── plugins/
│   ├── <plugin-name>/
│   │   ├── plugin.yaml              ← 元数据 (name, version, description, author)
│   │   ├── __init__.py              ← 入口文件, 必须定义 register(ctx)
│   │   └── <marketplace-clone>/     ← symlink 到 plugins_path/<repo>/
│   └── ...
├── skills/                           ← 全局 skills
└── config.yaml                       ← Hermes 配置
```

**plugin.yaml:**
```yaml
name: ecc
version: "1.0.0"
description: "Everything Claude Code"
author: "affaan-m"
```

**__init__.py:**
```python
"""Skill-tui managed plugin: ecc"""

def register(ctx):
    pass
```

**CLI 命令:**
- `hermes plugins install <git-url>` — 从 Git URL 安装 (60s timeout, 不支持本地路径)
- `hermes plugins enable <name>` — 启用已存在的插件
- `hermes plugins disable <name>` — 禁用
- `hermes plugins remove <name>` — 删除插件目录
- `hermes plugins list` — 列表 (Name/Status/Version/Description/Source)

**适配策略:** skill-tui 绕过 `hermes plugins install` (timeout 太短, 不支持本地路径),
直接创建 `~/.hermes/plugins/<name>/` 目录 (plugin.yaml + __init__.py + symlink),
然后用 `hermes plugins enable` 激活。

---

### 关键差异总结

| 方面 | Claude Code | Copilot | Hermes |
|------|-------------|---------|--------|
| 安装目录 | `plugins/cache/<mp>/<p>/<ver>/` | `installed-plugins/<mp>/<p>/` | `plugins/<name>/` |
| 版本在路径中 | ✅ | ❌ | ❌ |
| 安装记录文件 | `installed_plugins.json` | `config.json` → `installedPlugins[]` | 目录即记录 |
| Marketplace 注册 | `known_marketplaces.json` + `settings.json` | 仅 `settings.json` | 不适用 |
| Marketplace 缓存 | `~/.claude/plugins/marketplaces/` | `~/Library/Caches/copilot/marketplaces/` | 不适用 |
| 安装方式 | CLI (`claude plugin`) | CLI (`copilot plugin`) | 适配器 (直接写文件) |
| enabledPlugins 键格式 | `mp@plugin` | `mp@plugin` | 不适用 |
| config.json | 不存在 | 自动管理, 有注释行前缀 | 不适用 |

---

## 二、Plugin 文件格式

### marketplace.json

定义一个 marketplace (插件目录), 包含多个 plugin:

```json
{
  "name": "ecc",
  "owner": { "name": "Author", "email": "..." },
  "metadata": { "description": "...", "version": "1.0.0" },
  "plugins": [
    {
      "name": "ecc",
      "source": "./",
      "description": "...",
      "version": "2.0.0-rc.1",
      "category": "productivity"
    }
  ]
}
```

**source 类型:**
```json
"source": "./"                              // 相对路径 (同 repo)
"source": "./plugins/code-review"           // 子目录
"source": { "source": "github", "repo": "owner/repo" }        // GitHub
"source": { "source": "url", "url": "https://..." }           // Git URL
"source": { "source": "git-subdir", "url": "...", "path": "..." }  // monorepo 子目录
```

### plugin.json

定义单个 plugin 的元数据:

```json
{
  "name": "ecc",
  "version": "2.0.0-rc.1",
  "description": "...",
  "author": { "name": "Author", "url": "..." },
  "skills": "skills/",
  "commands": "commands/",
  "agents": "agents/",
  "hooks": "hooks/hooks.json",
  "mcpServers": ".mcp.json",
  "lspServers": ".lsp.json"
}
```

### Plugin 目录标准布局

```
plugin-name/
├── .claude-plugin/
│   ├── plugin.json           ← 元数据 (唯一必需字段: name)
│   └── marketplace.json      ← 可选, 如果此 repo 是一个 marketplace
├── skills/
│   └── skill-name/SKILL.md   ← Skill 定义
├── commands/
│   └── command-name.md       ← Slash 命令
├── agents/
│   └── agent-name.md         ← Sub-agent 定义
├── hooks/
│   └── hooks.json            ← 生命周期 hooks
├── .mcp.json                 ← MCP 服务器配置
└── .lsp.json                 ← LSP 服务器配置
```

---

## 三、skill-tui 插件安装架构

### 核心原则

1. **skill-tui 统一管理源码** — 所有 plugin 源码 clone 到 `plugins_path` (用户配置)
2. **使用平台原生 CLI 安装** — 通过 `claude plugin`、`copilot plugin` 命令安装
3. **Hermes 适配器模式** — 直接创建兼容目录结构, 绕过不兼容的 CLI

### 架构图

```
┌──────────────────────────────────────────────────────────┐
│  skill-tui 统一管理层                                      │
│                                                          │
│  plugins_path (e.g. _Shared/Plugins/)                    │
│  ├── ECC/              ← git clone                       │
│  ├── CLI-Anything/     ← git clone                       │
│  └── superpowers/      ← git clone                       │
│                                                          │
│  ~/.skill-tui/marketplaces.json  ← skill-tui 自身状态     │
└──────────────┬───────────────────────────────────────────┘
               │
        ┌──────┴──────┐
        │  Install to  │
        │  Platform    │
        └──┬───┬────┬─┘
           │   │    │
     ┌─────▼┐ ┌▼───┐ ┌▼──────┐
     │Claude│ │Copi│ │Hermes │
     │ Code │ │lot │ │       │
     └──┬───┘ └─┬──┘ └──┬────┘
        │       │       │
        ▼       ▼       ▼

Claude Code:            Copilot:               Hermes:
$ claude plugin         $ copilot plugin       直接写文件 (适配器)
  marketplace add         marketplace add      ~/.hermes/plugins/<name>/
  <local-path>            <local-path>         ├── plugin.yaml
$ claude plugin         $ copilot plugin       ├── __init__.py
  install <p>@<mp>        install <p>@<mp>     └── <mp> → symlink
                                               $ hermes plugins enable
```

### 安装流程 (通用)

```
1. skill-tui clone repo → plugins_path/<repo-name>/
   (带 300s timeout, 部分 clone 清理, 大小写重命名保护)

2. 用户选择目标平台 (多选)

3. 对每个平台调用 InstallMarketplaceViaCLI():
   - Claude/Copilot: 传递本地 clone 路径给 CLI
   - Hermes: 创建适配器目录 + enable
```

### 安装流程 (Claude Code — 通过 CLI)

```
1. AddMarketplaceViaCLI("claude-code", localPath)
   → exec: claude plugin marketplace add <local-clone-path>

2. InstallPluginViaCLI("claude-code", pluginName)
   → exec: claude plugin install <plugin>@<marketplace>

(Claude CLI 自动处理: marketplaces/ symlink, cache/ 拷贝,
 installed_plugins.json, known_marketplaces.json, settings.json)
```

### 安装流程 (Copilot — 通过 CLI)

```
1. AddMarketplaceViaCLI("copilot", localPath)
   → exec: copilot plugin marketplace add <local-clone-path>

2. InstallPluginViaCLI("copilot", pluginName)
   → exec: copilot plugin install <plugin>@<marketplace>

(Copilot CLI 自动处理: installed-plugins/ 拷贝, config.json,
 settings.json enabledPlugins + extraKnownMarketplaces)
```

### 安装流程 (Hermes — 适配器模式)

```
1. hermesCreatePluginDir(pluginName, localPath, marketplace)
   → 创建 ~/.hermes/plugins/<name>/
   → 写入 plugin.yaml (从 marketplace metadata 生成)
   → 写入 __init__.py (minimal: def register(ctx): pass)
   → symlink: <name>/<mp-name> → plugins_path/<repo>/

2. exec: hermes plugins enable <name>

为什么绕过 hermes plugins install:
- 60s timeout (大 repo 不够)
- 不支持本地路径 (只接受 git URL)
- 插件格式不兼容 (plugin.yaml + __init__.py vs .claude-plugin/)
```

### 卸载流程

```
Claude Code:  claude plugin uninstall <plugin>@<marketplace>
Copilot:      copilot plugin uninstall <name>
Hermes:       hermes plugins remove <name>
```

### 安装检测

```go
IsPluginInstalled(platformName, pluginName) bool

// Claude Code: 检查 ~/.claude/plugins/installed_plugins.json 中是否有 "<name>@<name>" key
// Copilot:     检查 ~/.copilot/installed-plugins/<name>/ 目录是否存在
// Hermes:      运行 hermes plugins list, 解析输出表格查找 plugin 名称
```

### 平台类型分类

```go
func PluginInstallClass(name string) PluginInstallType {
    switch name {
    case "claude-code":
        return PluginInstallClaude    // CLI: claude plugin
    case "copilot":
        return PluginInstallCopilot   // CLI: copilot plugin
    case "hermes":
        return PluginInstallHermes    // 适配器: 直接写文件 + enable
    default:
        return PluginInstallSymlinkOnly // 仅 skills 目录 symlink
    }
}
```

---

## 四、enabledPlugins 键格式

**⚠️ 重要:** Claude Code 和 Copilot 使用相同的 `marketplace@plugin` 格式。

基于实际文件系统验证:
- Claude: `"ecc@ecc": true`, `"cli-anything@cli-anything": true`
- Copilot: `"ecc@ecc": true`, `"cli-anything@cli-anything": true`
- 格式: `"<marketplace-name>@<plugin-name>"`

---

## 五、Copilot config.json 特殊处理

Copilot 的 `config.json` 开头有注释行 (非标准 JSON):

```
// User settings belong in settings.json.
// This file is managed automatically.
{
  "firstLaunchAt": "...",
  ...
}
```

解析时需要:
1. 跳过开头的 `//` 注释行
2. 找到第一个 `{` 开始解析
3. 写回时保留注释头

---

## 六、关键发现与解决方案

| 问题 | 原因 | 解决方案 |
|------|------|---------|
| SSH host key failure | 子进程 exec 用 owner/repo 时走 SSH | 传递本地 clone 路径而非 URL |
| Clone timeout | ECC ~16MB, 需 2+ 分钟 | timeout 增至 300s + 部分 clone 清理 |
| 大小写冲突 (macOS) | APFS 大小写不敏感, `os.RemoveAll("ecc")` 删除 `ECC` | `strings.EqualFold` 检查 + 两步重命名 (via temp) |
| Hermes timeout | `hermes plugins install` 只有 60s | 绕过 CLI, 直接写文件 |
| Hermes 不支持本地路径 | 只接受 git URL 或 owner/repo | 适配器模式: 创建兼容目录结构 |
| Hermes 格式不兼容 | plugin.yaml + __init__.py vs .claude-plugin/ | 从 marketplace metadata 生成兼容文件 |
