# Contributing

感谢你对 skill-tui 的关注！以下是贡献指南。

## 开发环境

### 前置依赖

- Go 1.22+
- make

### 快速开始

```bash
git clone https://github.com/ying-sun1/skill-tui.git
cd skill-tui
make build
```

### 常用命令

```bash
make build       # 构建 skill-tui 二进制
make test        # 运行所有测试
make lint        # go vet + gofmt
make run         # 构建并运行
make clean       # 清理
```

## 项目结构

```
cmd/             # Cobra 命令定义 — 参数解析 + 输出格式化
internal/        # 核心业务逻辑（不对外暴露）
  config/        #   配置加载/保存
  skill/         #   Skill 模型 + Registry CRUD
  platform/      #   平台检测 + 符号链接管理
  github/        #   GitHub API 客户端 + 导入器
  marketplace/   #   Marketplace 客户端 + 缓存
  collection/    #   技能集合存储
  discover/      #   本地技能发现扫描器
  ai/            #   LLM API 客户端
  i18n/          #   国际化
  tui/           #   Bubbletea TUI
    styles/      #     主题 (Catppuccin)
    components/  #     可复用 TUI 组件
configs/         # 静态配置 (platforms.yaml, registry.json)
```

## 开发约定

### 命令模式

新命令放在 `cmd/` 下，遵循 cobra 模式：

```go
var fooCmd = &cobra.Command{
    Use:   "foo <args>",
    Short: "Short description",
    RunE: func(cmd *cobra.Command, args []string) error {
        cfg, err := config.Load()
        if err != nil {
            return err
        }
        // 业务逻辑调用 internal/ 层
        return nil
    },
}

func init() {
    rootCmd.AddCommand(fooCmd)
    fooCmd.Flags().StringP("option", "o", "", "Description")
}
```

### 业务逻辑

业务逻辑放在 `internal/` 对应包中，命令层只做参数解析和输出格式化。

### 错误处理

```go
// 使用 fmt.Errorf + %w 包装上下文
if err != nil {
    return fmt.Errorf("loading skill %s: %w", name, err)
}
```

### 测试

```bash
# 运行测试
go test ./... -v

# 带覆盖率
go test ./... -cover

# 单个包
go test ./internal/skill/ -v
```

测试使用标准 `testing` + table-driven 模式：

```go
func TestParseMetadata(t *testing.T) {
    tests := []struct{
        name    string
        input   string
        want    Metadata
    }{
        {"empty", "", Metadata{}},
        {"basic", "---\ndescription: test\n---\n", Metadata{Description: "test"}},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := parseMetadata(tt.input)
            if got.Description != tt.want.Description {
                t.Errorf("got %q, want %q", got.Description, tt.want.Description)
            }
        })
    }
}
```

## 添加新平台

编辑 `configs/platforms.yaml`：

```yaml
  - name: my-platform
    category: coding
    skills_dir: ~/.my-platform/skills/
```

无需改代码，程序会自动加载。

## 添加 Marketplace Skills

编辑 `configs/registry.json`：

```json
{
  "name": "my-skill",
  "publisher_id": "community",
  "description": "My awesome skill",
  "version": "1.0.0",
  "tags": ["tag1"],
  "path": "skills/my-skill",
  "repo_url": "https://github.com/user/my-skill-repo"
}
```

## 提交 PR

1. Fork 本仓库
2. 创建功能分支 (`git checkout -b feature/my-feature`)
3. 提交更改 (`git commit -m 'feat: add my feature'`)
4. 推送分支 (`git push origin feature/my-feature`)
5. 创建 Pull Request

### Commit 格式

```
<type>: <description>

类型: feat, fix, refactor, docs, test, chore, perf, ci
```

## License

Apache License 2.0
