package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/jredh-dev/pylon/internal/cal"
	"github.com/jredh-dev/pylon/internal/config"
	"github.com/jredh-dev/pylon/internal/discord"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "version":
		fmt.Println("pylon", version)
	case "cal":
		if len(os.Args) < 3 {
			calUsage()
			os.Exit(1)
		}
		runCal(os.Args[2:])
	case "discord":
		if len(os.Args) < 3 {
			discordUsage()
			os.Exit(1)
		}
		runDiscord(os.Args[2:])
	case "help", "--help", "-h":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func runCal(args []string) {
	cfg, err := config.Load()
	if err != nil {
		fatal("config: %v", err)
	}

	// Allow --url flag to override
	url := cfg.CalURL
	rest := args
	for i, a := range args {
		if a == "--url" && i+1 < len(args) {
			url = args[i+1]
			rest = append(args[:i], args[i+2:]...)
			break
		}
		if strings.HasPrefix(a, "--url=") {
			url = strings.TrimPrefix(a, "--url=")
			rest = append(args[:i], args[i+1:]...)
			break
		}
	}

	client := cal.NewClient(url)

	if len(rest) < 1 {
		calUsage()
		os.Exit(1)
	}

	switch rest[0] {
	case "feed":
		if len(rest) < 2 {
			calFeedUsage()
			os.Exit(1)
		}
		runCalFeed(client, rest[1:])
	case "event":
		if len(rest) < 2 {
			calEventUsage()
			os.Exit(1)
		}
		runCalEvent(client, rest[1:])
	case "subscribe":
		runCalSubscribe(client, rest[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown cal command: %s\n\n", rest[0])
		calUsage()
		os.Exit(1)
	}
}

func runCalFeed(client *cal.Client, args []string) {
	switch args[0] {
	case "create":
		if len(args) < 2 {
			fatal("usage: pylon cal feed create <name> [slug]")
		}
		// Last arg is the slug if there are 3+ args, otherwise no slug.
		// Name can be multiple words, slug is always the final single token.
		var name, slug string
		if len(args) >= 3 {
			slug = args[len(args)-1]
			name = strings.Join(args[1:len(args)-1], " ")
		} else {
			name = strings.Join(args[1:], " ")
		}
		feed, err := client.CreateFeed(name, slug)
		if err != nil {
			fatal("create feed: %v", err)
		}
		fmt.Printf("Created feed:\n")
		fmt.Printf("  ID:    %s\n", feed.ID)
		fmt.Printf("  Name:  %s\n", feed.Name)
		fmt.Printf("  Token: %s\n", feed.Token)
		fmt.Printf("  URL:   %s\n", feed.URL)

	case "list", "ls":
		feeds, err := client.ListFeeds()
		if err != nil {
			fatal("list feeds: %v", err)
		}
		if len(feeds) == 0 {
			fmt.Println("No feeds.")
			return
		}
		tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintf(tw, "ID\tNAME\tTOKEN\tCREATED\n")
		for _, f := range feeds {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
				f.ID, f.Name, f.Token, f.CreatedAt.Format(time.DateOnly))
		}
		_ = tw.Flush()

	case "delete", "rm":
		if len(args) < 2 {
			fatal("usage: pylon cal feed delete <id>")
		}
		if err := client.DeleteFeed(args[1]); err != nil {
			fatal("delete feed: %v", err)
		}
		fmt.Println("Feed deleted.")

	default:
		fmt.Fprintf(os.Stderr, "unknown feed command: %s\n\n", args[0])
		calFeedUsage()
		os.Exit(1)
	}
}

func runCalEvent(client *cal.Client, args []string) {
	switch args[0] {
	case "add", "create":
		req := parseEventFlags(args[1:])
		event, err := client.CreateEvent(req)
		if err != nil {
			fatal("create event: %v", err)
		}
		fmt.Printf("Created event:\n")
		fmt.Printf("  ID:      %s\n", event.ID)
		fmt.Printf("  Summary: %s\n", event.Summary)
		fmt.Printf("  Start:   %s\n", event.Start.Format(time.RFC3339))
		if event.End != nil {
			fmt.Printf("  End:     %s\n", event.End.Format(time.RFC3339))
		}
		if event.Location != "" {
			fmt.Printf("  Location: %s\n", event.Location)
		}

	case "list", "ls":
		feedID := parseFeedIDFlag(args[1:])
		if feedID == "" {
			fatal("usage: pylon cal event list --feed <feed-id>")
		}
		events, err := client.ListEvents(feedID)
		if err != nil {
			fatal("list events: %v", err)
		}
		if len(events) == 0 {
			fmt.Println("No events.")
			return
		}
		tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintf(tw, "ID\tSUMMARY\tSTART\tEND\tSTATUS\n")
		for _, e := range events {
			end := ""
			if e.End != nil {
				end = e.End.Format(time.RFC3339)
			}
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
				e.ID, e.Summary, e.Start.Format(time.RFC3339), end, e.Status)
		}
		_ = tw.Flush()

	case "delete", "rm":
		if len(args) < 2 {
			fatal("usage: pylon cal event delete <id>")
		}
		if err := client.DeleteEvent(args[1]); err != nil {
			fatal("delete event: %v", err)
		}
		fmt.Println("Event deleted.")

	default:
		fmt.Fprintf(os.Stderr, "unknown event command: %s\n\n", args[0])
		calEventUsage()
		os.Exit(1)
	}
}

func runCalSubscribe(client *cal.Client, args []string) {
	if len(args) < 1 {
		fatal("usage: pylon cal subscribe <token>")
	}
	token := args[0]
	url := client.SubscribeURL(token)
	webcal := strings.Replace(url, "http://", "webcal://", 1)
	webcal = strings.Replace(webcal, "https://", "webcal://", 1)

	fmt.Printf("Subscribe URL:  %s\n", url)
	fmt.Printf("Webcal URL:     %s\n", webcal)
	fmt.Println()
	fmt.Println("To subscribe in your calendar app, use the webcal URL.")
	fmt.Println("For Google Calendar, use the https URL in 'Other calendars > From URL'.")
}

// --- Discord commands ---

func runDiscord(args []string) {
	cfg, err := config.Load()
	if err != nil {
		fatal("config: %v", err)
	}
	client := discord.NewClient(cfg.DiscordBotToken, cfg.DiscordWebhook)

	switch args[0] {
	case "msg", "send":
		if len(args) < 2 {
			fatal("usage: pylon discord msg <message>")
		}
		message := strings.Join(args[1:], " ")
		if err := client.SendMessage(message); err != nil {
			fatal("discord msg: %v", err)
		}
		fmt.Println("Message sent.")

	case "read":
		channelID := cfg.DiscordChannelID
		count := 20
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--channel":
				if i+1 < len(args) {
					i++
					channelID = args[i]
				}
			case "--count":
				if i+1 < len(args) {
					i++
					n, err := strconv.Atoi(args[i])
					if err == nil && n > 0 {
						count = n
					}
				}
			default:
				if strings.HasPrefix(args[i], "--channel=") {
					channelID = strings.TrimPrefix(args[i], "--channel=")
				} else if strings.HasPrefix(args[i], "--count=") {
					n, err := strconv.Atoi(strings.TrimPrefix(args[i], "--count="))
					if err == nil && n > 0 {
						count = n
					}
				}
			}
		}
		if channelID == "" {
			fatal("channel ID required\nUsage: pylon discord read [--channel <id>] [--count N]\nOr set channel_id in ~/.pylonrc [discord] or PYLON_DISCORD_CHANNEL_ID")
		}
		msgs, err := client.ReadMessages(channelID, count)
		if err != nil {
			fatal("discord read: %v", err)
		}
		if len(msgs) == 0 {
			fmt.Println("No messages found.")
			return
		}
		fmt.Print(discord.FormatMessages(msgs))

	case "channels":
		guildID := cfg.DiscordGuildID
		for i := 1; i < len(args); i++ {
			if args[i] == "--guild" && i+1 < len(args) {
				i++
				guildID = args[i]
			} else if strings.HasPrefix(args[i], "--guild=") {
				guildID = strings.TrimPrefix(args[i], "--guild=")
			}
		}
		if guildID == "" {
			fatal("guild ID required\nUsage: pylon discord channels --guild <id>\nOr set guild_id in ~/.pylonrc [discord] or PYLON_DISCORD_GUILD_ID")
		}
		channels, err := client.ListChannels(guildID)
		if err != nil {
			fatal("discord channels: %v", err)
		}
		tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintf(tw, "ID\tNAME\n")
		for _, ch := range channels {
			fmt.Fprintf(tw, "%s\t#%s\n", ch.ID, ch.Name)
		}
		_ = tw.Flush()

	default:
		fmt.Fprintf(os.Stderr, "unknown discord command: %s\n\n", args[0])
		discordUsage()
		os.Exit(1)
	}
}

// --- flag parsing helpers ---

func parseEventFlags(args []string) *cal.CreateEventRequest {
	req := &cal.CreateEventRequest{}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--feed":
			i++
			req.FeedID = args[i]
		case "--summary":
			i++
			req.Summary = args[i]
		case "--start":
			i++
			req.Start = args[i]
		case "--end":
			i++
			req.End = args[i]
		case "--description":
			i++
			req.Description = args[i]
		case "--location":
			i++
			req.Location = args[i]
		case "--url":
			i++
			req.URL = args[i]
		case "--all-day":
			req.AllDay = true
		case "--deadline":
			i++
			req.Deadline = args[i]
		case "--status":
			i++
			req.Status = args[i]
		case "--categories":
			i++
			req.Categories = args[i]
		default:
			if strings.HasPrefix(args[i], "--") {
				fatal("unknown flag: %s", args[i])
			}
			// Positional: treat as summary if not set
			if req.Summary == "" {
				req.Summary = args[i]
			}
		}
	}

	if req.FeedID == "" {
		fatal("--feed is required")
	}
	if req.Summary == "" {
		fatal("--summary is required")
	}
	if req.Start == "" {
		fatal("--start is required")
	}

	return req
}

func parseFeedIDFlag(args []string) string {
	for i := 0; i < len(args); i++ {
		if args[i] == "--feed" && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(args[i], "--feed=") {
			return strings.TrimPrefix(args[i], "--feed=")
		}
	}
	return ""
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "pylon: "+format+"\n", args...)
	os.Exit(1)
}

func usage() {
	fmt.Fprintf(os.Stderr, `pylon - interact with deployed infrastructure

Usage:
  pylon <service> <command> [flags]

Services:
  cal         Calendar subscription service
  discord     Discord messaging and channel access

Other:
  version     Show version
  help        Show this help

Configuration:
  ~/.pylonrc            INI-style config file (optional)
  PYLON_* env vars      Override config file values

Run 'pylon <service> --help' for service-specific commands.
`)
}

func calUsage() {
	fmt.Fprintf(os.Stderr, `pylon cal - calendar service commands

Usage:
  pylon cal [--url <base-url>] <resource> <action> [flags]

Resources:
  feed        Manage calendar feeds
  event       Manage calendar events
  subscribe   Get subscription URLs for a feed

Configuration:
  ~/.pylonrc [cal] url = ...     Base URL for the cal service
  PYLON_CAL_URL                  Env var override (default: http://localhost:8085)
`)
}

func calFeedUsage() {
	fmt.Fprintf(os.Stderr, `pylon cal feed - manage calendar feeds

Commands:
  create <name> [slug]  Create a new feed (slug sets a readable URL token)
  list                  List all feeds
  delete <id>           Delete a feed and all its events
`)
}

func calEventUsage() {
	fmt.Fprintf(os.Stderr, `pylon cal event - manage calendar events

Commands:
  add [flags]         Create a new event
  list --feed <id>    List events for a feed
  delete <id>         Delete an event

Flags for 'add':
  --feed <id>         Feed ID (required)
  --summary <text>    Event title (required)
  --start <datetime>  Start time in RFC 3339 format (required)
  --end <datetime>    End time in RFC 3339 format
  --description <text>
  --location <text>
  --url <url>
  --all-day           Mark as all-day event
  --deadline <datetime>  Deadline with alarm
  --status <status>   TENTATIVE, CONFIRMED, or CANCELLED
  --categories <list> Comma-separated categories
`)
}

func discordUsage() {
	fmt.Fprintf(os.Stderr, `pylon discord - Discord messaging and channel access

Usage:
  pylon discord <command> [flags]

Commands:
  msg <message>                     Send a message via webhook
  read [--channel <id>] [--count N] Read recent messages from a channel
  channels [--guild <id>]           List text channels in a guild

Configuration (~/.pylonrc [discord] section or env vars):
  webhook      / PYLON_DISCORD_WEBHOOK      Webhook URL for sending messages
  bot_token    / PYLON_DISCORD_BOT_TOKEN    Bot token for reading messages/channels
  guild_id     / PYLON_DISCORD_GUILD_ID     Default guild (server) ID
  channel_id   / PYLON_DISCORD_CHANNEL_ID   Default channel ID for reading
`)
}
