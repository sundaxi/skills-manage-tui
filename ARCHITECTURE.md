# Architecture

## 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                        skill-tui                            │
├──────────────┬──────────────────────────────────────────────┤
│   cmd/       │  Cobra 命令层                                │
│   ┌────────┐ │  ┌──────┐ ┌───────┐ ┌────┐ ┌─────┐ ┌────┐  │
│   │ root   │ │  │ list │ │install│ │sync│ │remove│ │... │  │
│   └───┬────┘ │  └──┬───┘ └───┬───┘ └─┬──┘ └──┬──┘ └──┬─┘  │
│       │      │     │        │       │       │       │      │
├───────┼──────┼─────┼────────┼───────┼───────┼───────┼──────┤
│       ▼      │     ▼        ▼       ▼       ▼       ▼      │
│  internal/   │  核心业务层                                        │
│  ┌────────┐  │  ┌──────────┐ ┌──────────┐ ┌──────────────┐  │
│  │ skill  │  │  │ Registry │ │ Metadata │ │ Skill Model  │  │
│  └────────┘  │  └──────────┘ └──────────┘ └──────────────┘  │
│  ┌────────┐  │  ┌──────────┐ ┌──────────┐ ┌──────────────┐  │
│  │platform│  │  │ Detector │ │  Linker  │ │ Platform Def │  │
│  └────────┘  │  └──────────┘ └──────────┘ └──────────────┘  │
│  ┌────────┐  │  ┌──────────┐ ┌──────────┐ ┌──────────────┐  │
│  │github  │  │  │  Client  │ │ Importer │ │ Tree Parser  │  │
│  └────────┘  │  └──────────┘ └──────────┘ └──────────────┘  │
│  ┌────────────┴──────────────────────────────────────────┐  │
│  │  tui/                                                  │  │
│  │  ┌──────┐ ┌────────────┐ ┌───────────┐ ┌───────────┐  │  │
│  │  │ App  │ │  Views     │ │ Components│ │  Styles   │  │  │
│  │  │ TabNav│ │List/Detail │ │MultiSelect│ │ Catppuccin│  │  │
│  │  │ Shell│ │Marketplace │ │ Search    │ │ Mocha/    │  │  │
│  │  │      │ │Settings    │ │ StatusBar │ │ Latte     │  │  │
│  │  └──────┘ └────────────┘ └───────────┘ └───────────┘  │  │
│  └───────────────────────────────────────────────────────┘  │
│  ┌──────────┐ ┌───────────┐ ┌──────────┐ ┌──────────────┐  │
│  │marketplace│ │ collection│ │ discover │ │     ai       │  │
│  │  Client   │ │  Store    │ │ Scanner  │ │  Explainer   │  │
│  │  Cache    │ │  CRUD     │ │ Recursive│ │  Cache       │  │
│  └──────────┘ └───────────┘ └──────────┘ └──────────────┘  │
├─────────────────────────────────────────────────────────────┤
│  config/    │  配置层 (Viper + YAML)                           │
│  i18n/      │  国际化 (中/英, LANG 自动检测)                    │
├─────────────────────────────────────────────────────────────┤
│  configs/   │  platforms.yaml · registry.json                 │
└─────────────────────────────────────────────────────────────┘
```

## 模块职责

### cmd/ — CLI 命令层

Cobra 命令定义，负责参数解析、调用 internal 层、格式化输出。

| 文件 | 命令 | 职责 |
|------|------|------|
| `root.go` | `skill-tui` | 入口：无参数 → TUI，有参数 → 子命令 |
| `tui.go` | (内部) | 初始化 bubbletea Program 并启动 TUI |
| `list.go` | `list` | 表格输出 skills，支持 --verbose / --platform |
| `install.go` | `install` | 安装 skill 到平台，--all / --platform 选择 |
| `sync.go` | `sync` | 全量同步，--dry-run 预览，修复断裂链接 |
| `remove.go` | `remove` | 删除符号链接，--purge 删除源文件 |
| `marketplace.go` | `marketplace` | browse / search / install 子命令 |
| `import.go` | `import` | GitHub URL 导入，调用 github.Importer |
| `collection.go` | `collection` | CRUD + 批量安装 |
| `discover.go` | `discover` | 本地目录扫描 |
| `config.go` | `config` | get / set / platforms 配置管理 |

### internal/skill/ — Skill 核心模型

```
skill.go        Skill struct { Name, Path, Description, Version, Author, Tags, Content, Platforms }
                Registry struct — 中央技能库 CRUD (ListSkills, GetSkill, AddSkill, RemoveSkill)
metadata.go     parseMetadata() — 解析 SKILL.md YAML frontmatter
                extractFrontmatter() — 提取 --- 分隔的 frontmatter 块
```

**SKILL.md 格式：**
```markdown
---
description: "Skill 描述"
version: "1.0.0"
author: "作者"
tags: [tag1, tag2]
---

# Skill 标题

Skill 内容...
```

### internal/platform/ — 平台管理

```
platform.go     Platform struct { Name, Category, SkillsDir, CommandsDir, MarketplacesDir, Installed }
                ListPlatforms() — 加载所有平台定义
                DetectInstalled() — 自动检测目录存在性
                FindPlatform() — 按名称查找
                PluginInstallClass() — 返回平台的插件安装类型
                IsPluginInstalled() — 检测插件是否已安装 (平台感知)

platform_cli.go PlatformCLI() — 返回平台 CLI 命令名
                AddMarketplaceViaCLI() — 添加 marketplace 到平台
                InstallPluginViaCLI() — 通过 CLI 安装插件
                UninstallPluginViaCLI() — 通过 CLI 卸载插件
                InstallMarketplaceViaCLI() — 完整 marketplace 安装流程
                UninstallMarketplaceViaCLI() — 完整卸载流程
                hermesCreatePluginDir() — Hermes 适配器: 创建兼容目录结构

linker.go       Install() — 创建符号链接 (platform_dir/skill_name → agents_dir/skill_name)
                Uninstall() — 删除符号链接
                IsLinked() — 检查链接状态
                LinkedPlatforms() — 反查 skill 安装到了哪些平台
                BrokenLinks() — 检测断裂的符号链接
```

**符号链接策略 (Skills)：**

```
源 (中央库)                      目标 (平台目录)
~/.agents/skills/my-skill/   →  ~/.claude/skills/my-skill  (symlink)
                              →  ~/.cursor/skills/my-skill  (symlink)
                              →  ~/.copilot/skills/my-skill (symlink)
```

**插件安装策略 (Plugins)：**

```
插件安装类型:
  PluginInstallClaude  → CLI: claude plugin marketplace add + install
  PluginInstallCopilot → CLI: copilot plugin marketplace add + install
  PluginInstallHermes  → 适配器: plugin.yaml + __init__.py + hermes plugins enable
  PluginInstallSymlinkOnly → 仅 skills 目录 symlink (其他平台)
```

### internal/plugin/ — 插件管理

```
plugin.go       Store struct — 管理 plugins_path 下的 marketplace 克隆
                NewStore() — 初始化 Store
                ScanMarketplaces() — 扫描所有已 clone 的 marketplace
                AddByRepo() — 从 GitHub URL 克隆 marketplace
                RemoveMarketplace() — 删除 marketplace
                CloneRepo() — 带 timeout 的 git clone (300s)
                ParseMarketplace() — 解析 .claude-plugin/marketplace.json
```

### internal/config/ — 配置管理

```
config.go       Config struct — 所有可配置项
                Load() — 从 ~/.skill-tui/config.yaml 加载
                Save() — 保存配置
                ConfigPath() — 返回配置文件路径
defaults.go     loadDefaultPlatforms() — 从 configs/platforms.yaml 加载
                fallbackPlatforms() — 硬编码兜底
                expandHome() — ~/ 路径展开
```

**配置加载优先级：**
1. 环境变量 (`SKILL_CLI_*`)
2. 配置文件 (`~/.skill-tui/config.yaml`)
3. 默认值

### internal/tui/ — 交互式界面

```
app.go          AppModel — bubbletea 主模型
                4 Tab 页路由 (Skills/Marketplace/Plugin/Settings)
                7 视图状态 (List/Detail/PlatformSelect/PluginDetail/PluginInstall/PluginAdd/Settings)
                全局快捷键处理
                搜索过滤逻辑
                鼠标事件处理 (单击/双击)
                插件 install/uninstall 异步操作 + 状态消息

styles/theme.go Theme struct — Catppuccin Mocha/Latte 调色板
                NewTheme() — 按名称创建主题
                NewThemeWithAccent() — 创建带自定义强调色的主题
                AccentColors — 10 种可选强调色
                28 种颜色 + 16 种预定义样式

components/     multiselect.go — 通用多选组件 (Space/a/j/k)
                search.go — 搜索输入框 (bubbles/textinput)
                statusbar.go — 底部状态栏 (含 error/success 消息)
```

**TUI 状态机：**

```
                    ┌─────────────┐
          ┌────────►│  viewList   │◄────────┐
          │         │  (技能列表)   │          │
          │         └──┬────┬─────┘          │
          │   Esc      │    │ Enter/d        │ Esc
          │            │    ▼                │
          │            │  ┌──────────────┐   │
          │            │  │ viewDetail   │   │
          │            │  │  (详情页)     │   │
          │            │  └──────────────┘   │
          │            │                     │
          │            │ p (多选后)           │ Enter
          │            ▼                     │
          │  ┌────────────────────┐          │
          └──│ viewPlatformSelect │──────────┘
             │  (平台选择)          │
             └────────────────────┘

Settings Tab:
  ↑/↓ 移动光标 → Enter 切换主题/强调色 / 编辑路径
  主题: mocha ↔ latte (实时切换)
  强调色: 10 种 Catppuccin 色循环
  路径: 文本输入编辑, Enter 确认
```

### internal/github/ — GitHub 集成

```
client.go       Client — GitHub REST API v3 封装
                ParseRepoURL() — 解析 GitHub URL
                GetRepo() — 获取仓库信息
                GetTree() — 递归获取文件树
                GetFileContent() — 下载文件内容
                ListRepos() — 列出用户仓库

importer.go     Importer — 从 GitHub 仓库导入 Skills
                ImportFromURL() — URL → 解析 → 扫描 → 下载
                ImportFromRepo() — 仓库对象 → 扫描 → 下载
                findSkillFiles() — 在 Git tree 中查找 SKILL.md
                importSkillFile() — 下载并写入本地
```

**导入流程：**
```
GitHub URL → ParseRepoURL → GetTree(recursive)
  → findSkillFiles(SKILL.md) → for each:
    → GetFileContent → write to ~/.agents/skills/<name>/SKILL.md
```

### internal/marketplace/ — 技能市场

```
client.go       Client — JSON registry 客户端
                FetchRegistry() — 从远程获取 registry.json
                ListPublishers() — 列出发布者
                ListSkills() — 列出 Skills (可按发布者过滤)
                Search() — 按名称/描述/标签搜索

cache.go        Cache — 内存缓存 (10 min TTL, sync.RWMutex)
```

**Registry 格式 (configs/registry.json)：**
```json
{
  "publishers": [{ "id", "name", "description", "repo_url" }],
  "skills": [{ "name", "publisher_id", "description", "version", "tags", "path", "repo_url" }]
}
```

### internal/collection/ — 技能集合

```
collection.go   Store — JSON 文件存储 (~/.skill-tui/collections.json)
                Collection struct { Name, Description, Skills[], CreatedAt }
                CRUD: List, Get, Create, Delete, AddSkill, RemoveSkill
```

### internal/discover/ — 本地发现

```
scanner.go      Scan() — 扫描指定目录下的 15 种已知 skills 目录
                ScanRecursive() — 递归扫描 (可控深度)
                knownSkillDirs — 预定义的 {dir, platform} 映射
```

### internal/ai/ — AI 解释

```
explainer.go    Explainer — LLM API 客户端
                Explain() — 生成 Skill 解释 (含本地缓存)
                callOpenAI() — OpenAI Chat Completions API
                callAnthropic() — Anthropic Messages API
                loadCache() / saveCache() — ~/.skill-tui/cache/<name>.json
```

## 数据流

### 安装流程

```
skill-tui install my-skill --all
  │
  ├─ config.Load()
  │    └─ viper → Config{SkillsPath, Platforms}
  │
  ├─ skill.NewRegistry(cfg.SkillsPath)
  │    └─ Registry.GetSkill("my-skill")
  │         └─ os.Stat + LoadSkill (读 SKILL.md frontmatter)
  │
  ├─ platform.DetectInstalled(cfg)
  │    └─ for each Platform: os.Stat(SkillsDir) → Installed=true
  │
  └─ for each installed platform:
       └─ platform.Install(p.SkillsDir, skill.Path, skill.Name)
            ├─ os.MkdirAll(platformSkillsDir)
            ├─ os.Remove(existingLink) // if any
            ├─ filepath.Abs(skillPath) // resolve source
            └─ os.Symlink(absSkillPath, linkPath)
```

### 同步流程

```
skill-tui sync
  │
  ├─ registry.ListSkills() → [skill-a, skill-b, ...]
  │
  └─ for each platform:
       ├─ for each skill:
       │    └─ if !IsLinked → Install()
       │
       └─ BrokenLinks()
            └─ for each broken:
                 ├─ if still in registry → remove + recreate
                 └─ if gone → remove stale link
```

### TUI 事件循环

```
bubbletea Program
  │
  ├─ Init() → tea.EnterAltScreen
  │
  ├─ Update(msg)
  │    ├─ WindowSizeMsg → 记录 width/height
  │    ├─ MouseMsg → Tab 栏点击 / 技能行单击 / 双击选中
  │    ├─ KeyMsg (search active) → search.Update + filterSkills
  │    ├─ KeyMsg (viewPlatformSelect) → multiSel.Update
  │    ├─ KeyMsg (viewDetail) → handleDetail
  │    ├─ KeyMsg (settings) → handleSettings (主题/强调色切换, 路径编辑)
  │    └─ KeyMsg (viewList) → navigate/select/enter/search
  │
  └─ View() → renderTabBar + renderContent + renderStatusBar
```

## 项目结构

```
skill-tui/
├── main.go                          # 入口
├── go.mod / go.sum                  # Go 模块
├── Makefile                         # 构建脚本
├── configs/
│   ├── platforms.yaml               # 28+ 平台定义
│   └── registry.json                # Marketplace 技能注册表
├── cmd/                             # Cobra 命令 (11 个文件)
│   ├── root.go                      # 根命令 + 版本信息
│   ├── tui.go                       # TUI 启动入口
│   ├── list.go                      # 列表命令
│   ├── install.go                   # 安装命令
│   ├── sync.go                      # 同步命令
│   ├── remove.go                    # 移除命令
│   ├── marketplace.go               # 市场 (browse/search/install)
│   ├── import.go                    # GitHub 导入
│   ├── collection.go                # 集合 (list/create/delete/install/add/remove)
│   ├── discover.go                  # 本地发现
│   └── config.go                    # 配置 (get/set/platforms)
├── internal/
│   ├── config/                      # 配置管理
│   │   ├── config.go                # Config struct + Load/Save
│   │   └── defaults.go              # 平台加载 + 路径展开
│   ├── skill/                       # Skill 模型
│   │   ├── skill.go                 # Skill struct + Registry CRUD
│   │   └── metadata.go              # SKILL.md frontmatter 解析
│   ├── platform/                    # 平台管理
│   │   ├── platform.go              # Platform struct + 检测 + 插件类型
│   │   ├── platform_cli.go          # CLI 安装/卸载 + Hermes 适配器
│   │   ├── platform_test.go         # 平台安装格式测试
│   │   └── linker.go                # 符号链接操作
│   ├── github/                      # GitHub 集成
│   │   ├── client.go                # REST API v3 客户端
│   │   └── importer.go              # 仓库导入逻辑
│   ├── plugin/                      # 插件 (Marketplace) 管理
│   │   ├── plugin.go                # Store + clone + scan + parse
│   │   └── marketplace_test.go      # 解析测试
│   ├── marketplace/                 # Marketplace
│   │   ├── client.go                # Registry 客户端 + 搜索
│   │   └── cache.go                 # 内存缓存
│   ├── collection/                  # 技能集合
│   │   └── collection.go            # JSON 存储 + CRUD
│   ├── discover/                    # 本地发现
│   │   └── scanner.go               # 目录扫描器
│   ├── ai/                          # AI 解释
│   │   └── explainer.go             # LLM 客户端 + 缓存
│   ├── i18n/                        # 国际化
│   │   └── i18n.go                  # 中/英消息映射
│   └── tui/                         # 交互式界面
│       ├── app.go                   # 主 bubbletea 模型
│       ├── styles/
│       │   └── theme.go             # Catppuccin 调色板 + 样式
│       └── components/
│           ├── multiselect.go       # 多选列表
│           ├── search.go            # 搜索框
│           └── statusbar.go         # 状态栏
```

## 关键设计决策

### 为什么用符号链接？

- **单一事实来源** — 一份 skill 文件，多平台共享
- **即时生效** — 修改中央库的 skill 立即反映到所有平台
- **零拷贝** — 不复制文件，节省磁盘空间
- **可逆** — 删除符号链接即卸载，不影响源文件

### 为什么用 bubbletea？

- Go 生态最成熟的 TUI 框架
- Elm Architecture (Model-Update-View) 清晰易测
- 内置鼠标支持、焦点管理、终端兼容
- GitHub CLI、Hugo 等大型项目验证

### 为什么用 Cobra？

- Go CLI 事实标准
- 自动生成帮助文档、shell 补全、man pages
- 丰富的 flag 类型、子命令嵌套
- 与 Viper 配置管理无缝集成

## 依赖关系图

```
cmd/
 ├── config
 ├── skill
 │    └── metadata (internal)
 ├── platform
 │    └── skill (检测 linked platforms)
 ├── github
 │    └── (net/http)
 ├── marketplace
 │    └── cache (internal)
 ├── collection
 │    └── (encoding/json)
 ├── discover
 │    └── (os, filepath)
 ├── ai
 │    └── (net/http, encoding/json)
 ├── i18n
 │    └── (os, strings)
 └── tui
      ├── styles (Catppuccin, lipgloss)
      ├── components (bubbletea, bubbles, textinput)
      ├── config
      ├── skill
      ├── platform
      └── marketplace (Phase 5 扩展)
```
