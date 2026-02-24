package chatapps

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

type PlatformConfig struct {
	Platform         string         `yaml:"platform"`
	SystemPrompt     string         `yaml:"system_prompt"`
	TaskInstructions string         `yaml:"task_instructions"`
	Options          map[string]any `yaml:"options,omitempty"`
}

type Logger = slog.Logger

type ConfigLoader struct {
	configs map[string]*PlatformConfig
	mu      sync.RWMutex
	logger  *slog.Logger
}

func NewConfigLoader(configDir string, logger *slog.Logger) (*ConfigLoader, error) {
	loader := &ConfigLoader{
		configs: make(map[string]*PlatformConfig),
		logger:  logger,
	}

	if err := loader.Load(configDir); err != nil {
		return nil, fmt.Errorf("load configs: %w", err)
	}

	return loader, nil
}

func (c *ConfigLoader) Load(configDir string) error {
	entries, err := os.ReadDir(configDir)
	if err != nil {
		return fmt.Errorf("read config dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}

		filename := filepath.Join(configDir, entry.Name())
		data, err := os.ReadFile(filename)
		if err != nil {
			c.logger.Debug("Failed to read config file", "file", filename, "error", err)
			continue
		}

		var cfg PlatformConfig
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			c.logger.Error("Failed to parse config file", "file", filename, "error", err)
			continue
		}

		if cfg.Platform == "" {
			c.logger.Warn("Config missing platform field", "file", filename)
			continue
		}

		c.mu.Lock()
		c.configs[cfg.Platform] = &cfg
		c.mu.Unlock()
		c.logger.Info("Loaded platform config", "platform", cfg.Platform)
	}
	return nil
}

func (c *ConfigLoader) GetConfig(platform string) *PlatformConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if cfg, ok := c.configs[platform]; ok {
		return &PlatformConfig{
			Platform:         cfg.Platform,
			SystemPrompt:     cfg.SystemPrompt,
			TaskInstructions: cfg.TaskInstructions,
		}
	}
	return nil
}

func (c *ConfigLoader) GetSystemPrompt(platform string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if cfg, ok := c.configs[platform]; ok {
		return cfg.SystemPrompt
	}
	return ""
}

func (c *ConfigLoader) GetTaskInstructions(platform string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if cfg, ok := c.configs[platform]; ok {
		return cfg.TaskInstructions
	}
	return ""
}

func (c *ConfigLoader) HasPlatform(platform string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, ok := c.configs[platform]
	return ok
}

func (c *ConfigLoader) Platforms() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	platforms := make([]string, 0, len(c.configs))
	for p := range c.configs {
		platforms = append(platforms, p)
	}
	return platforms
}

func (c *ConfigLoader) GetOptions(platform string) map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if cfg, ok := c.configs[platform]; ok {
		return cfg.Options
	}
	return nil
}
