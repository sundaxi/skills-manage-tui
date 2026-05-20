# Plugin 安装机制设计文档

> 最后更新: 2026-05-20  
> 基于 Claude Code 和 Copilot CLI 的实际文件系统分析和官方文档研究

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

### 关键差异总结

| 方面 | Claude Code | Copilot |
|------|-------------|---------|
| 安装目录 | `plugins/cache/<mp>/<p>/<ver>/` | `installed-plugins/<mp>/<p>/` |
| 版本在路径中 | ✅ | ❌ |
| 安装记录文件 | `installed_plugins.json` | `config.json` → `installedPlugins[]` |
| Marketplace 注册 | `known_marketplaces.json` + `settings.json` | 仅 `settings.json` → `extraKnownMarketplaces` |
| Marketplace 缓存 | `~/.claude/plugins/marketplaces/` | `~/Library/Caches/copilot/marketplaces/` |
| enabledPlugins 键格式 | `mp@plugin` | `plugin@mp` (注意顺序!) |
| config.json | 不存在 | 自动管理, 有注释行前缀 |

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
2. **按平台原生格式安装** — 每个平台写入各自预期的配置文件
3. **不创建平台不读取的文件** — 不在 Copilot 下写 `installed_plugins.json` 等

### 架构图

```
┌──────────────────────────────────────────────────────────┐
│  skill-tui 统一管理层                                      │
│                                                          │
│  plugins_path (e.g. _Shared/Plugins/)                    │
│  ├── ecc/              ← git clone                       │
│  ├── CLI-Anything/     ← git clone                       │
│  └── superpowers/      ← git clone                       │
│                                                          │
│  ~/.skill-tui/marketplaces.json  ← skill-tui 自身状态     │
└──────────────┬───────────────────────────────────────────┘
               │
        ┌──────┴──────┐
        │  Install to  │
        │  Platform    │
        └──┬───────┬──┘
           │       │
     ┌─────▼──┐  ┌─▼──────────┐
     │ Claude │  │  Copilot   │
     │ Code   │  │  CLI       │
     └────┬───┘  └─────┬──────┘
          │            │
          ▼            ▼

Claude Code 写入:                 Copilot 写入:
~/.claude/                        ~/.copilot/
├── plugins/                      ├── installed-plugins/<mp>/<p>/  ← 文件拷贝
│   ├── marketplaces/<name>       ├── plugin-data/<mp>/<p>/        ← 空目录
│   │   → symlink to plugins_path ├── config.json                  ← installedPlugins[]
│   ├── cache/<mp>/<p>/<ver>/     └── settings.json                ← enabledPlugins
│   │   ← 文件拷贝                                                   + extraKnownMarketplaces
│   ├── installed_plugins.json
│   └── known_marketplaces.json
└── settings.json
    ← enabledPlugins
    + extraKnownMarketplaces
```

### 安装流程 (Claude Code)

```
1. SymlinkPlugin
   ~/.claude/plugins/marketplaces/<name> → plugins_path/<repo>/

2. CopyPluginToCache
   plugins_path/<repo>/ → ~/.claude/plugins/cache/<mp>/<plugin>/<version>/
   version = git SHA 前 12 位 (无 semver 时)

3. RecordInstalledPlugins
   → installed_plugins.json: {"version":2, "plugins":{"mp@plugin":[{...}]}}

4. RecordKnownMarketplace
   → known_marketplaces.json: {"name":{"source":{...},"installLocation":"..."}}

5. EnablePluginInSettings
   → settings.json: enabledPlugins["mp@plugin"] = true
   → settings.json: extraKnownMarketplaces["name"] = {source:{...}}
```

### 安装流程 (Copilot)

```
1. CopyPluginToCopilot (不需要 symlink, Copilot 不读 marketplaces/)
   plugins_path/<repo>/ → ~/.copilot/installed-plugins/<mp>/<plugin>/

2. CreatePluginDataDir
   → ~/.copilot/plugin-data/<mp>/<plugin>/  (空目录)

3. UpdateCopilotConfig
   → config.json: 追加到 installedPlugins[] 数组
   (注意: config.json 开头有注释行, 需特殊解析)

4. UpdateCopilotSettings
   → settings.json: enabledPlugins["plugin@mp"] = true
   → settings.json: extraKnownMarketplaces["name"] = {source:{...}}
```

### 卸载流程

**Claude Code:**
1. 删除 `plugins/marketplaces/<name>` symlink
2. 删除 `plugins/cache/<mp>/` 目录
3. 从 `installed_plugins.json` 移除相关条目
4. 从 `known_marketplaces.json` 移除
5. 从 `settings.json` 移除 `enabledPlugins` 和 `extraKnownMarketplaces` 条目

**Copilot:**
1. 删除 `installed-plugins/<mp>/` 目录
2. 删除 `plugin-data/<mp>/` 目录
3. 从 `config.json` 的 `installedPlugins[]` 移除
4. 从 `settings.json` 移除 `enabledPlugins` 和 `extraKnownMarketplaces` 条目

### 平台类型分类

```go
func PluginInstallClass(name string) PluginInstallType {
    switch name {
    case "claude-code":
        return PluginInstallClaude    // 完整 Claude 安装流程
    case "copilot":
        return PluginInstallCopilot   // Copilot 原生流程 (config.json)
    case "hermes":
        return PluginInstallUnsupported // 完全不兼容
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

## 六、与旧代码的差异 (需修复)

| 当前实现 | 正确行为 | 影响 |
|---------|---------|------|
| Copilot: `installedPlugins` 写入 `settings.json` | 应写入 `config.json` | 安装记录位置错误 |
| Copilot: 写入 `installed_plugins.json` | Copilot 不读取此文件 | 无效操作 |
| Copilot: 写入 `known_marketplaces.json` | Copilot 不读取此文件 | 无效操作 |
| Copilot: symlink 到 `plugins/marketplaces/` | Copilot 不读此目录 | 无效操作 |
| Copilot: 安装到 `plugins/cache/` | 应安装到 `installed-plugins/` | 路径错误 |
