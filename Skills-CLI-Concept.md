
## 文档概述
这是一份原始需求
## 项目概述
这个项目是一个用于管理多智能体skills的command line 工具。命令为skill-tui 
## 交互方式
支持两种交互方式
- skill-tui [sync|list|install|remove] 用于同步skill到多智能体删除等
- skill-tui 直接执行则进入交互界面，交互界面 REPL
交互界面支持原生的终端如ghostty item2等
![[Pasted image 20260513161330.png]] UI可以参考图示，可多选

## 默认配置

默认管理~/.agent/ 下的skills文件夹，该目录可以通过配置更改

## 功能
支持以下功能
- install
- sync
- remove 
- list 
## CLI 设计原则

| 原则             | 说明                                            |
| -------------- | --------------------------------------------- |
| **大声失败**       | Agent 需要明确的错误消息来自我纠正                          |
| **尽可能幂等**      | 运行相同命令两次应该安全                                  |
| **提供内省**       | `info`, `list`, `status` 命令对 Agent 了解当前状态至关重要 |
| **JSON 输出模式**  | 每个命令必须支持 `--json` 用于机器解析                      |
| **统一 REPL 外观** | 使用 `repl_skin.py` 提供一致的交互体验                   |
| **REPL 为默认行为** | 无参数运行时自动进入 REPL                               |
