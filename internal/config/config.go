package config

import "os"

// Config holds pylon configuration.
type Config struct {
	CalURL string // base URL for the cal service API

	DiscordWebhook   string // Discord webhook URL for sending messages
	DiscordBotToken  string // Discord bot token for reading messages/channels
	DiscordGuildID   string // Default Discord guild (server) ID
	DiscordChannelID string // Default Discord channel ID for reading
}

// Load reads configuration from environment variables.
func Load() *Config {
	calURL := os.Getenv("PYLON_CAL_URL")
	if calURL == "" {
		calURL = "http://localhost:8085"
	}
	return &Config{
		CalURL:           calURL,
		DiscordWebhook:   os.Getenv("PYLON_DISCORD_WEBHOOK"),
		DiscordBotToken:  os.Getenv("PYLON_DISCORD_BOT_TOKEN"),
		DiscordGuildID:   os.Getenv("PYLON_DISCORD_GUILD_ID"),
		DiscordChannelID: os.Getenv("PYLON_DISCORD_CHANNEL_ID"),
	}
}
