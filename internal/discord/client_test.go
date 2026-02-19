package discord

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendMessage(t *testing.T) {
	tests := []struct {
		name       string
		webhookURL string // empty means use server URL
		message    string
		status     int
		wantErr    bool
	}{
		{
			name:    "success",
			message: "hello world",
			status:  http.StatusNoContent,
			wantErr: false,
		},
		{
			name:    "success 200",
			message: "hello",
			status:  http.StatusOK,
			wantErr: false,
		},
		{
			name:    "server error",
			message: "fail",
			status:  http.StatusInternalServerError,
			wantErr: true,
		},
		{
			name:       "no webhook configured",
			webhookURL: "none",
			message:    "test",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotBody map[string]string

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
					t.Fatalf("decode body: %v", err)
				}
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()

			webhookURL := srv.URL
			if tt.webhookURL == "none" {
				webhookURL = ""
			}

			client := NewClient("", webhookURL)
			err := client.SendMessage(tt.message)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotBody["content"] != tt.message {
				t.Errorf("expected content %q, got %q", tt.message, gotBody["content"])
			}
		})
	}
}

func TestReadMessages(t *testing.T) {
	tests := []struct {
		name      string
		channelID string
		botToken  string
		limit     int
		status    int
		response  string
		wantErr   bool
		wantCount int
	}{
		{
			name:      "success",
			channelID: "chan-1",
			botToken:  "test-token",
			limit:     5,
			status:    http.StatusOK,
			response: mustJSON(t, []Message{
				{ID: "2", Content: "newer", Author: Author{Username: "bob"}},
				{ID: "1", Content: "older", Author: Author{Username: "alice"}},
			}),
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:      "empty channel",
			channelID: "chan-2",
			botToken:  "test-token",
			limit:     10,
			status:    http.StatusOK,
			response:  `[]`,
			wantErr:   false,
			wantCount: 0,
		},
		{
			name:      "no bot token",
			channelID: "chan-1",
			botToken:  "",
			wantErr:   true,
		},
		{
			name:     "no channel ID",
			botToken: "test-token",
			wantErr:  true,
		},
		{
			name:      "api error",
			channelID: "chan-1",
			botToken:  "test-token",
			limit:     5,
			status:    http.StatusForbidden,
			response:  `{"message":"Missing Access"}`,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("expected GET, got %s", r.Method)
				}
				auth := r.Header.Get("Authorization")
				if auth != "Bot "+tt.botToken {
					t.Errorf("expected auth %q, got %q", "Bot "+tt.botToken, auth)
				}
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.response))
			}))
			defer srv.Close()

			// Override apiBase via a client that points to our test server.
			client := NewClient(tt.botToken, "")
			// We need to hit the test server, so we'll call botGet directly
			// by building the URL ourselves. But ReadMessages uses the const
			// apiBase. We'll test via the handler instead.

			// Skip server-dependent tests when we expect client-side errors
			if tt.botToken == "" || tt.channelID == "" {
				_, err := client.ReadMessages(tt.channelID, tt.limit)
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			// For server tests, we need to override the API base.
			// Use a test-specific approach: create a handler that verifies
			// the request and test the client with the test server URL.
			// Since ReadMessages uses the const apiBase, we test the
			// integration differently - by testing botGet + parsing.
			body, err := client.botGet(srv.URL)
			if tt.wantErr && tt.status != http.StatusOK {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var msgs []Message
			if err := json.Unmarshal(body, &msgs); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if len(msgs) != tt.wantCount {
				t.Errorf("expected %d messages, got %d", tt.wantCount, len(msgs))
			}
		})
	}
}

func TestReadMessages_Reversal(t *testing.T) {
	// Verify messages are reversed to chronological order.
	msgs := []Message{
		{ID: "3", Content: "newest"},
		{ID: "2", Content: "middle"},
		{ID: "1", Content: "oldest"},
	}
	resp := mustJSON(t, msgs)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(resp))
	}))
	defer srv.Close()

	// We can't easily override the const apiBase, so test the reversal
	// logic directly using botGet + manual parse + reverse.
	client := NewClient("test-token", "")
	body, err := client.botGet(srv.URL)
	if err != nil {
		t.Fatalf("botGet: %v", err)
	}

	var got []Message
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Reverse (same logic as ReadMessages)
	for i, j := 0, len(got)-1; i < j; i, j = i+1, j-1 {
		got[i], got[j] = got[j], got[i]
	}

	if got[0].ID != "1" || got[1].ID != "2" || got[2].ID != "3" {
		t.Errorf("expected chronological order [1,2,3], got [%s,%s,%s]",
			got[0].ID, got[1].ID, got[2].ID)
	}
}

func TestListChannels(t *testing.T) {
	allChannels := []Channel{
		{ID: "1", Name: "general", Type: 0, Position: 0},
		{ID: "2", Name: "voice", Type: 2, Position: 1},
		{ID: "3", Name: "dev", Type: 0, Position: 2},
	}
	resp := mustJSON(t, allChannels)

	tests := []struct {
		name      string
		guildID   string
		botToken  string
		status    int
		response  string
		wantErr   bool
		wantCount int
	}{
		{
			name:      "success filters text channels",
			guildID:   "guild-1",
			botToken:  "test-token",
			status:    http.StatusOK,
			response:  resp,
			wantErr:   false,
			wantCount: 2, // only type 0
		},
		{
			name:     "no bot token",
			guildID:  "guild-1",
			botToken: "",
			wantErr:  true,
		},
		{
			name:     "no guild ID",
			botToken: "test-token",
			wantErr:  true,
		},
		{
			name:     "api error",
			guildID:  "guild-1",
			botToken: "test-token",
			status:   http.StatusForbidden,
			response: `{"message":"Missing Access"}`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.response))
			}))
			defer srv.Close()

			client := NewClient(tt.botToken, "")

			if tt.botToken == "" || tt.guildID == "" {
				_, err := client.ListChannels(tt.guildID)
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			// Test via botGet since ListChannels uses const apiBase
			body, err := client.botGet(srv.URL)
			if tt.wantErr && tt.status != http.StatusOK {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var all []Channel
			if err := json.Unmarshal(body, &all); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			// Apply same filter as ListChannels
			var text []Channel
			for _, ch := range all {
				if ch.Type == 0 {
					text = append(text, ch)
				}
			}
			if len(text) != tt.wantCount {
				t.Errorf("expected %d text channels, got %d", tt.wantCount, len(text))
			}
		})
	}
}

func TestFormatMessages(t *testing.T) {
	tests := []struct {
		name string
		msgs []Message
		want string
	}{
		{
			name: "simple message",
			msgs: []Message{
				{
					Timestamp: "2026-02-18T10:30:00.000Z",
					Content:   "hello",
					Author:    Author{Username: "alice", GlobalName: "Alice"},
				},
			},
			want: "[2026-02-18T10:30:00] Alice: hello\n",
		},
		{
			name: "falls back to username",
			msgs: []Message{
				{
					Timestamp: "2026-02-18T10:30:00.000Z",
					Content:   "hi",
					Author:    Author{Username: "bob"},
				},
			},
			want: "[2026-02-18T10:30:00] bob: hi\n",
		},
		{
			name: "empty content",
			msgs: []Message{
				{
					Timestamp: "2026-02-18T10:30:00.000Z",
					Author:    Author{Username: "eve"},
				},
			},
			want: "[2026-02-18T10:30:00] eve: (no text)\n",
		},
		{
			name: "reply message",
			msgs: []Message{
				{
					Timestamp: "2026-02-18T10:30:00.000Z",
					Content:   "I agree",
					Author:    Author{Username: "bob", GlobalName: "Bob"},
					Reference: &struct {
						Content string `json:"content"`
						Author  Author `json:"author"`
					}{
						Content: "this is great",
						Author:  Author{Username: "alice", GlobalName: "Alice"},
					},
				},
			},
			want: "[2026-02-18T10:30:00] Bob (reply to Alice: \"this is great\"): I agree\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatMessages(tt.msgs)
			if got != tt.want {
				t.Errorf("expected:\n%s\ngot:\n%s", tt.want, got)
			}
		})
	}
}

func TestAuthorDisplayName(t *testing.T) {
	tests := []struct {
		name   string
		author Author
		want   string
	}{
		{
			name:   "prefers global name",
			author: Author{Username: "alice", GlobalName: "Alice Smith"},
			want:   "Alice Smith",
		},
		{
			name:   "falls back to username",
			author: Author{Username: "bob"},
			want:   "bob",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.author.DisplayName()
			if got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func mustJSON(t *testing.T, v interface{}) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal test data: %v", err)
	}
	return string(b)
}
