package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const (
	AppName       = "skill-tui"
	ConfigDir     = ".skill-tui"
	DefaultSkills = ".agents/skills"
)

type Config struct {
	SkillsPath      string     `mapstructure:"skills_path"`
	Platforms       []Platform `mapstructure:"platforms"`
	Theme           string     `mapstructure:"theme"`
	AccentColor     string     `mapstructure:"accent_color"`
	Language        string     `mapstructure:"language"`
	GitHubToken     string     `mapstructure:"github_token"`
	AIProvider      string     `mapstructure:"ai_provider"`
	AIKey           string     `mapstructure:"ai_key"`
	AIEndpoint      string     `mapstructure:"ai_endpoint"`
	CustomPlatforms []Platform `mapstructure:"custom_platforms"`
}

type Platform struct {
	Name      string `mapstructure:"name" yaml:"name"`
	Category  string `mapstructure:"category" yaml:"category"`
	SkillsDir string `mapstructure:"skills_dir" yaml:"skills_dir"`
}

var cfg *Config
var vInstance *viper.Viper

func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(filepath.Join(home, ConfigDir))
	v.AddConfigPath(".")

	v.SetDefault("skills_path", filepath.Join(home, DefaultSkills))
	v.SetDefault("theme", "mocha")
	v.SetDefault("accent_color", "mauve")
	v.SetDefault("language", "auto")

	v.SetEnvPrefix("SKILL_CLI")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}
	vInstance = v

	platforms, err := loadDefaultPlatforms(filepath.Join(home, ConfigDir))
	if err != nil {
		return nil, err
	}

	skillsPath := v.GetString("skills_path")

	cfg = &Config{
		SkillsPath:   skillsPath,
		Theme:        v.GetString("theme"),
		AccentColor:  v.GetString("accent_color"),
		Language:     v.GetString("language"),
		GitHubToken:  v.GetString("github_token"),
		AIProvider:   v.GetString("ai_provider"),
		AIKey:        v.GetString("ai_key"),
		AIEndpoint:   v.GetString("ai_endpoint"),
		Platforms:    platforms,
	}

	var customs []Platform
	if err := v.UnmarshalKey("custom_platforms", &customs); err == nil {
		cfg.CustomPlatforms = customs
		cfg.Platforms = append(cfg.Platforms, customs...)
	}

	syncCentralPlatform(cfg)

	return cfg, nil
}

func Get() *Config {
	return cfg
}

func ConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ConfigDir, "config.yaml")
}

func Save(c *Config) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	dir := filepath.Join(home, ConfigDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	configFile := filepath.Join(dir, "config.yaml")

	v := viper.New()
	v.SetConfigFile(configFile)
	_ = v.ReadInConfig()

	v.Set("skills_path", c.SkillsPath)
	v.Set("theme", c.Theme)
	v.Set("accent_color", c.AccentColor)
	v.Set("language", c.Language)
	v.Set("github_token", c.GitHubToken)
	v.Set("ai_provider", c.AIProvider)
	v.Set("ai_key", c.AIKey)
	v.Set("ai_endpoint", c.AIEndpoint)

	if err := v.WriteConfig(); err != nil {
		return v.SafeWriteConfigAs(configFile)
	}
	return nil
}

func Reload() (*Config, error) {
	return Load()
}

func syncCentralPlatform(c *Config) {
	for i := range c.Platforms {
		if c.Platforms[i].Category == "central" {
			c.Platforms[i].SkillsDir = c.SkillsPath
			return
		}
	}
	c.Platforms = append(c.Platforms, Platform{
		Name:      "central",
		Category:  "central",
		SkillsDir: c.SkillsPath,
	})
}
