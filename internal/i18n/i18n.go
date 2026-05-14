package i18n

import (
	"os"
	"strings"
)

type Lang string

const (
	Auto Lang = "auto"
	ZH   Lang = "zh"
	EN   Lang = "en"
)

var detected Lang

var messages = map[string]map[Lang]string{
	// CLI
	"cli.short":        {EN: "Manage AI coding agent skills across multiple platforms", ZH: "统一管理多平台 AI 编程 Agent 的 Skills"},
	"cli.long":         {EN: "Manages skills in ~/.agents/skills/ (central registry) and installs\nthem to 28+ AI coding platforms via symlinks.\n\nRun without arguments to enter interactive TUI mode.", ZH: "使用 ~/.agents/skills/ 作为中央技能库，通过符号链接安装到 28+ AI 编程平台。\n\n不带参数运行将进入交互式 TUI 界面。"},

	// Commands
	"cmd.list":         {EN: "List installed skills", ZH: "列出已安装的技能"},
	"cmd.install":      {EN: "Install a skill to platform(s)", ZH: "安装技能到指定平台"},
	"cmd.sync":         {EN: "Sync skills between central registry and platforms", ZH: "同步中央技能库与各平台"},
	"cmd.remove":       {EN: "Remove a skill from platform(s)", ZH: "从平台移除技能"},
	"cmd.marketplace":  {EN: "Browse and install skills from the marketplace", ZH: "浏览并安装市场中的技能"},
	"cmd.import":       {EN: "Import skills from a GitHub repository", ZH: "从 GitHub 仓库导入技能"},
	"cmd.collection":   {EN: "Manage skill collections", ZH: "管理技能集合"},
	"cmd.discover":     {EN: "Discover local project-level skills", ZH: "发现本地项目级技能"},
	"cmd.config":       {EN: "View and manage configuration", ZH: "查看和管理配置"},

	// List
	"list.empty":       {EN: "No skills found in", ZH: "未找到技能，路径："},
	"list.header":      {EN: "NAME\tVERSION\tDESCRIPTION", ZH: "名称\t版本\t描述"},

	// Install
	"install.not_found": {EN: "skill not found in registry", ZH: "在技能库中未找到"},
	"install.success":   {EN: "Installed %s → %s", ZH: "已安装 %s → %s"},
	"install.select":    {EN: "Use --platform <name> or --all to select target platform(s).", ZH: "使用 --platform <名称> 或 --all 选择目标平台。"},

	// Sync
	"sync.complete":    {EN: "Sync complete.", ZH: "同步完成。"},
	"sync.linked":      {EN: "Linked %s → %s", ZH: "已链接 %s → %s"},
	"sync.fixed":       {EN: "Fixed broken link: %s", ZH: "修复断裂链接: %s"},
	"sync.stale":       {EN: "Removed stale link: %s", ZH: "移除过期链接: %s"},

	// Remove
	"remove.confirm":   {EN: "Remove %s from all platforms? [y/N] ", ZH: "确认从所有平台移除 %s？[y/N] "},
	"remove.cancelled": {EN: "Cancelled.", ZH: "已取消。"},
	"removed":          {EN: "Removed %s from %s", ZH: "已从 %s 移除 %s"},
	"purged":           {EN: "Purged %s from central registry", ZH: "已从中央库清除 %s"},

	// TUI
	"tui.tab.skills":       {EN: "Skills", ZH: "技能"},
	"tui.tab.marketplace":  {EN: "Marketplace", ZH: "市场"},
	"tui.tab.collections":  {EN: "Collections", ZH: "集合"},
	"tui.tab.discover":     {EN: "Discover", ZH: "发现"},
	"tui.tab.settings":     {EN: "Settings", ZH: "设置"},
	"tui.search.placeholder": {EN: "Search skills...", ZH: "搜索技能..."},
	"tui.no_skills":        {EN: "No skills found", ZH: "未找到技能"},
	"tui.add_skills":       {EN: "Add skills to ~/.agents/skills/ to get started", ZH: "将技能添加到 ~/.agents/skills/ 即可开始"},
	"tui.keys.navigate":    {EN: "↑/k↓/j: navigate  Space: select  a: all  Enter/d: detail  p: install  x: remove  /: search", ZH: "↑/k↓/j: 导航  Space: 选择  a: 全选  Enter/d: 详情  p: 安装  x: 移除  /: 搜索"},
	"tui.not_installed":    {EN: "Not installed to any platform", ZH: "尚未安装到任何平台"},
	"tui.detail.keys":      {EN: "Esc: back  i: install  u: uninstall  o: open full content", ZH: "Esc: 返回  i: 安装  u: 卸载  o: 查看完整内容"},

	// Settings
	"settings.skills_path": {EN: "Skills Path: ", ZH: "技能路径: "},
	"settings.theme":       {EN: "Theme: ", ZH: "主题: "},
	"settings.language":    {EN: "Language: ", ZH: "语言: "},
	"settings.platforms":   {EN: "Detected Platforms", ZH: "已检测平台"},

	// Config
	"config.not_set":       {EN: "(not set)", ZH: "(未设置)"},
	"config.unknown_key":   {EN: "unknown config key: %s", ZH: "未知配置项: %s"},
	"config.saved":         {EN: "Set %s = %s", ZH: "已设置 %s = %s"},

	// Common
	"version":          {EN: "version", ZH: "版本"},
	"author":           {EN: "Author: ", ZH: "作者: "},
	"installed_platforms": {EN: "Installed Platforms", ZH: "已安装平台"},
	"platforms":        {EN: "%d platforms", ZH: "%d 个平台"},
	"skills_count":     {EN: "%d skills", ZH: "%d 个技能"},
}

var currentLang Lang

func Init(lang string) {
	currentLang = Lang(lang)
	if currentLang == Auto || currentLang == "" {
		currentLang = detectLang()
	}
	detected = currentLang
}

func T(key string) string {
	msgs, ok := messages[key]
	if !ok {
		return key
	}
	text, ok := msgs[currentLang]
	if !ok {
		text = msgs[EN]
	}
	return text
}

func Tf(key string, args ...interface{}) string {
	return fmtSprintf(T(key), args...)
}

func fmtSprintf(format string, args ...interface{}) string {
	return strings.ReplaceAll(format, "%s", strings.Repeat("%s", len(args)))
}

func detectLang() Lang {
	lang := os.Getenv("LANG")
	if strings.HasPrefix(lang, "zh") {
		return ZH
	}
	lang = os.Getenv("LC_ALL")
	if strings.HasPrefix(lang, "zh") {
		return ZH
	}
	return EN
}

func Current() Lang {
	return currentLang
}
