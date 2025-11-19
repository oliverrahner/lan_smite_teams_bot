package config

import (
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

// GuildConfig represents configuration for a specific Discord guild
type GuildConfig struct {
	GuildID        string `yaml:"guild_id"`
	AllowedChannel string `yaml:"allowed_channel,omitempty"` // Optional: restrict to specific channel
}

// Config represents the overall bot configuration
type Config struct {
	Token      string                 `yaml:"token"`
	Guilds     map[string]GuildConfig `yaml:"guilds"`
	mutex      sync.RWMutex
	configPath string
}

var (
	instance *Config
	once     sync.Once
)

// GetConfig returns the singleton configuration instance
func GetConfig() *Config {
	once.Do(func() {
		instance = &Config{
			Guilds: make(map[string]GuildConfig),
		}
	})
	return instance
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configPath string) error {
	config := GetConfig()
	config.mutex.Lock()
	defer config.mutex.Unlock()

	config.configPath = configPath

	// Load from file if it exists
	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}

		if err := yaml.Unmarshal(data, config); err != nil {
			return fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Override with environment variables
	if token := os.Getenv("DISCORD_BOT_TOKEN"); token != "" {
		config.Token = token
	}

	// Validate required configuration
	if config.Token == "" {
		return fmt.Errorf("discord bot token is required (set DISCORD_BOT_TOKEN environment variable)")
	}

	return nil
}

// SaveConfig saves the current configuration to file
func SaveConfig() error {
	config := GetConfig()
	config.mutex.RLock()
	defer config.mutex.RUnlock()

	if config.configPath == "" {
		return fmt.Errorf("config path not set")
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(config.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetGuildConfig returns configuration for a specific guild
func GetGuildConfig(guildID string) GuildConfig {
	config := GetConfig()
	config.mutex.RLock()
	defer config.mutex.RUnlock()

	if guildConfig, exists := config.Guilds[guildID]; exists {
		return guildConfig
	}

	// Return default configuration for new guilds
	return GuildConfig{
		GuildID: guildID,
	}
}

// SetGuildConfig sets configuration for a specific guild
func SetGuildConfig(guildID string, guildConfig GuildConfig) error {
	config := GetConfig()
	config.mutex.Lock()
	defer config.mutex.Unlock()

	guildConfig.GuildID = guildID
	config.Guilds[guildID] = guildConfig

	return SaveConfig()
}

// GetToken returns the Discord bot token
func GetToken() string {
	config := GetConfig()
	config.mutex.RLock()
	defer config.mutex.RUnlock()

	return config.Token
}
