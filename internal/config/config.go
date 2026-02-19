package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Config holds pylon configuration.
type Config struct {
	CalURL string // base URL for the cal service API

	DiscordWebhook   string // Discord webhook URL for sending messages
	DiscordBotToken  string // Discord bot token for reading messages/channels
	DiscordGuildID   string // Default Discord guild (server) ID
	DiscordChannelID string // Default Discord channel ID for reading
}

// Load reads configuration from ~/.pylonrc (INI-style sections), then applies
// environment variable overrides. Env vars always take precedence over the
// config file. If ~/.pylonrc does not exist, only env vars are used.
func Load() (*Config, error) {
	cfg := &Config{
		CalURL: "http://localhost:8085",
	}

	// Load from file first.
	if err := cfg.loadFile(); err != nil {
		return nil, err
	}

	// Env vars override file values.
	cfg.applyEnv()

	return cfg, nil
}

// loadFile reads ~/.pylonrc if it exists. The file uses INI-style sections:
//
//	[cal]
//	url = http://localhost:8085
//
//	[discord]
//	webhook = https://discord.com/api/webhooks/...
//	bot_token = ...
//	guild_id = ...
//	channel_id = ...
func (c *Config) loadFile() error {
	path, err := rcPath()
	if err != nil {
		return nil // can't determine home dir, skip file
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no config file is fine
		}
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	return c.parse(f)
}

// parse reads an INI-style config from the given reader.
func (c *Config) parse(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	section := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments.
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Section header.
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}

		// Key = value.
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		c.set(section, key, value)
	}

	return scanner.Err()
}

// set applies a single config value from the given section and key.
func (c *Config) set(section, key, value string) {
	switch section {
	case "cal":
		switch key {
		case "url":
			c.CalURL = value
		}
	case "discord":
		switch key {
		case "webhook":
			c.DiscordWebhook = value
		case "bot_token":
			c.DiscordBotToken = value
		case "guild_id":
			c.DiscordGuildID = value
		case "channel_id":
			c.DiscordChannelID = value
		}
	}
}

// applyEnv overrides config values with environment variables when set.
func (c *Config) applyEnv() {
	if v := os.Getenv("PYLON_CAL_URL"); v != "" {
		c.CalURL = v
	}
	if v := os.Getenv("PYLON_DISCORD_WEBHOOK"); v != "" {
		c.DiscordWebhook = v
	}
	if v := os.Getenv("PYLON_DISCORD_BOT_TOKEN"); v != "" {
		c.DiscordBotToken = v
	}
	if v := os.Getenv("PYLON_DISCORD_GUILD_ID"); v != "" {
		c.DiscordGuildID = v
	}
	if v := os.Getenv("PYLON_DISCORD_CHANNEL_ID"); v != "" {
		c.DiscordChannelID = v
	}
}

// rcPath returns the path to ~/.pylonrc.
func rcPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".pylonrc"), nil
}
