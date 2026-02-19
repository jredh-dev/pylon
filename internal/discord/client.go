package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const apiBase = "https://discord.com/api/v10"

// Client talks to the Discord API.
type Client struct {
	botToken   string
	webhookURL string
	httpClient *http.Client
}

// NewClient creates a Discord client. botToken is used for reading
// messages/channels (Bot API), webhookURL is used for sending messages.
func NewClient(botToken, webhookURL string) *Client {
	return &Client{
		botToken:   botToken,
		webhookURL: webhookURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Message is a Discord message.
type Message struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
	Author    Author `json:"author"`
	Reference *struct {
		Content string `json:"content"`
		Author  Author `json:"author"`
	} `json:"referenced_message"`
}

// Author is a Discord message author.
type Author struct {
	Username   string `json:"username"`
	GlobalName string `json:"global_name"`
}

// DisplayName returns the best display name for an author.
func (a Author) DisplayName() string {
	if a.GlobalName != "" {
		return a.GlobalName
	}
	return a.Username
}

// Channel is a Discord guild channel.
type Channel struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     int    `json:"type"`
	Position int    `json:"position"`
}

// SendMessage posts a plain text message to the configured webhook.
func (c *Client) SendMessage(message string) error {
	if c.webhookURL == "" {
		return fmt.Errorf("webhook URL not configured (set PYLON_DISCORD_WEBHOOK)")
	}

	payload, err := json.Marshal(map[string]string{"content": message})
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	resp, err := c.httpClient.Post(c.webhookURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// ReadMessages fetches the latest messages from a channel. Limit is capped at
// 100 by the Discord API; defaults to 20 if out of range.
func (c *Client) ReadMessages(channelID string, limit int) ([]Message, error) {
	if c.botToken == "" {
		return nil, fmt.Errorf("bot token not configured (set PYLON_DISCORD_BOT_TOKEN)")
	}
	if channelID == "" {
		return nil, fmt.Errorf("channel ID required")
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	url := fmt.Sprintf("%s/channels/%s/messages?limit=%d", apiBase, channelID, limit)
	body, err := c.botGet(url)
	if err != nil {
		return nil, err
	}

	var msgs []Message
	if err := json.Unmarshal(body, &msgs); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	// API returns newest-first; reverse to chronological order.
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}

	return msgs, nil
}

// ListChannels returns text channels visible to the bot in a guild.
func (c *Client) ListChannels(guildID string) ([]Channel, error) {
	if c.botToken == "" {
		return nil, fmt.Errorf("bot token not configured (set PYLON_DISCORD_BOT_TOKEN)")
	}
	if guildID == "" {
		return nil, fmt.Errorf("guild ID required")
	}

	url := fmt.Sprintf("%s/guilds/%s/channels", apiBase, guildID)
	body, err := c.botGet(url)
	if err != nil {
		return nil, err
	}

	var all []Channel
	if err := json.Unmarshal(body, &all); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	// Filter to text channels (type 0) only.
	var text []Channel
	for _, ch := range all {
		if ch.Type == 0 {
			text = append(text, ch)
		}
	}
	return text, nil
}

// FormatMessages renders messages for terminal output.
func FormatMessages(msgs []Message) string {
	var sb strings.Builder
	for _, m := range msgs {
		ts := m.Timestamp
		if len(ts) >= 19 {
			ts = ts[:19]
		}
		author := m.Author.DisplayName()
		content := m.Content
		if content == "" {
			content = "(no text)"
		}
		if m.Reference != nil {
			ref := m.Reference
			refAuthor := ref.Author.DisplayName()
			refContent := ref.Content
			if refContent == "" {
				refContent = "(no text)"
			}
			fmt.Fprintf(&sb, "[%s] %s (reply to %s: %q): %s\n", ts, author, refAuthor, refContent, content)
		} else {
			fmt.Fprintf(&sb, "[%s] %s: %s\n", ts, author, content)
		}
	}
	return sb.String()
}

// botGet performs an authenticated GET request against the Discord Bot API.
func (c *Client) botGet(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bot "+c.botToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Discord API error (status %d): %s", resp.StatusCode, string(body))
	}
	return body, nil
}
