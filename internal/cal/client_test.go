package cal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCreateFeed(t *testing.T) {
	tests := []struct {
		name       string
		feedName   string
		status     int
		response   string
		wantErr    bool
		wantFeedID string
	}{
		{
			name:       "success",
			feedName:   "Work",
			status:     http.StatusCreated,
			response:   `{"id":"feed-1","name":"Work","token":"abc123","url":"/cal/abc123.ics"}`,
			wantErr:    false,
			wantFeedID: "feed-1",
		},
		{
			name:     "server error",
			feedName: "Bad",
			status:   http.StatusInternalServerError,
			response: `{"error":"database error"}`,
			wantErr:  true,
		},
		{
			name:     "conflict",
			feedName: "Duplicate",
			status:   http.StatusConflict,
			response: `{"error":"feed already exists"}`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if r.URL.Path != "/api/feeds" {
					t.Errorf("expected /api/feeds, got %s", r.URL.Path)
				}

				var body map[string]string
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatalf("decode request body: %v", err)
				}
				if body["name"] != tt.feedName {
					t.Errorf("expected name %q, got %q", tt.feedName, body["name"])
				}

				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.response))
			}))
			defer srv.Close()

			client := NewClient(srv.URL)
			feed, err := client.CreateFeed(tt.feedName)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if feed.ID != tt.wantFeedID {
				t.Errorf("expected feed ID %q, got %q", tt.wantFeedID, feed.ID)
			}
			if feed.Name != tt.feedName {
				t.Errorf("expected feed name %q, got %q", tt.feedName, feed.Name)
			}
		})
	}
}

func TestListFeeds(t *testing.T) {
	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		status    int
		response  string
		wantErr   bool
		wantCount int
	}{
		{
			name:   "success with feeds",
			status: http.StatusOK,
			response: mustJSON(t, []Feed{
				{ID: "f1", Name: "Work", Token: "tok1", CreatedAt: now, UpdatedAt: now},
				{ID: "f2", Name: "Personal", Token: "tok2", CreatedAt: now, UpdatedAt: now},
			}),
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:      "success empty",
			status:    http.StatusOK,
			response:  `[]`,
			wantErr:   false,
			wantCount: 0,
		},
		{
			name:     "server error",
			status:   http.StatusInternalServerError,
			response: `{"error":"internal"}`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("expected GET, got %s", r.Method)
				}
				if r.URL.Path != "/api/feeds" {
					t.Errorf("expected /api/feeds, got %s", r.URL.Path)
				}
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.response))
			}))
			defer srv.Close()

			client := NewClient(srv.URL)
			feeds, err := client.ListFeeds()

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(feeds) != tt.wantCount {
				t.Errorf("expected %d feeds, got %d", tt.wantCount, len(feeds))
			}
		})
	}
}

func TestDeleteFeed(t *testing.T) {
	tests := []struct {
		name    string
		feedID  string
		status  int
		wantErr bool
	}{
		{
			name:    "success",
			feedID:  "feed-1",
			status:  http.StatusNoContent,
			wantErr: false,
		},
		{
			name:    "not found",
			feedID:  "nonexistent",
			status:  http.StatusNotFound,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete {
					t.Errorf("expected DELETE, got %s", r.Method)
				}
				expectedPath := "/api/feeds/" + tt.feedID
				if r.URL.Path != expectedPath {
					t.Errorf("expected %s, got %s", expectedPath, r.URL.Path)
				}
				w.WriteHeader(tt.status)
				if tt.status != http.StatusNoContent {
					_, _ = w.Write([]byte(`{"error":"not found"}`))
				}
			}))
			defer srv.Close()

			client := NewClient(srv.URL)
			err := client.DeleteFeed(tt.feedID)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestCreateEvent(t *testing.T) {
	now := time.Date(2026, 2, 1, 14, 0, 0, 0, time.UTC)
	end := now.Add(time.Hour)

	tests := []struct {
		name        string
		req         *CreateEventRequest
		status      int
		response    string
		wantErr     bool
		wantEventID string
	}{
		{
			name: "success",
			req: &CreateEventRequest{
				FeedID:  "feed-1",
				Summary: "Meeting",
				Start:   now.Format(time.RFC3339),
				End:     end.Format(time.RFC3339),
			},
			status: http.StatusCreated,
			response: mustJSON(t, Event{
				ID: "evt-1", FeedID: "feed-1", Summary: "Meeting",
				Start: now, End: &end, Status: "CONFIRMED",
				CreatedAt: now, UpdatedAt: now,
			}),
			wantErr:     false,
			wantEventID: "evt-1",
		},
		{
			name: "bad request",
			req: &CreateEventRequest{
				FeedID:  "",
				Summary: "No Feed",
				Start:   now.Format(time.RFC3339),
			},
			status:   http.StatusBadRequest,
			response: `{"error":"feed_id is required"}`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if r.URL.Path != "/api/events" {
					t.Errorf("expected /api/events, got %s", r.URL.Path)
				}

				var body CreateEventRequest
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatalf("decode request body: %v", err)
				}
				if body.Summary != tt.req.Summary {
					t.Errorf("expected summary %q, got %q", tt.req.Summary, body.Summary)
				}

				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.response))
			}))
			defer srv.Close()

			client := NewClient(srv.URL)
			event, err := client.CreateEvent(tt.req)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if event.ID != tt.wantEventID {
				t.Errorf("expected event ID %q, got %q", tt.wantEventID, event.ID)
			}
		})
	}
}

func TestListEvents(t *testing.T) {
	now := time.Date(2026, 2, 1, 14, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		feedID    string
		status    int
		response  string
		wantErr   bool
		wantCount int
	}{
		{
			name:   "success",
			feedID: "feed-1",
			status: http.StatusOK,
			response: mustJSON(t, []Event{
				{ID: "e1", FeedID: "feed-1", Summary: "Meeting", Start: now, Status: "CONFIRMED", CreatedAt: now, UpdatedAt: now},
			}),
			wantErr:   false,
			wantCount: 1,
		},
		{
			name:      "empty",
			feedID:    "feed-2",
			status:    http.StatusOK,
			response:  `[]`,
			wantErr:   false,
			wantCount: 0,
		},
		{
			name:     "not found",
			feedID:   "nonexistent",
			status:   http.StatusNotFound,
			response: `{"error":"feed not found"}`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("expected GET, got %s", r.Method)
				}
				expectedPath := "/api/feeds/" + tt.feedID + "/events"
				if r.URL.Path != expectedPath {
					t.Errorf("expected %s, got %s", expectedPath, r.URL.Path)
				}
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.response))
			}))
			defer srv.Close()

			client := NewClient(srv.URL)
			events, err := client.ListEvents(tt.feedID)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(events) != tt.wantCount {
				t.Errorf("expected %d events, got %d", tt.wantCount, len(events))
			}
		})
	}
}

func TestDeleteEvent(t *testing.T) {
	tests := []struct {
		name    string
		eventID string
		status  int
		wantErr bool
	}{
		{
			name:    "success",
			eventID: "evt-1",
			status:  http.StatusNoContent,
			wantErr: false,
		},
		{
			name:    "not found",
			eventID: "nonexistent",
			status:  http.StatusNotFound,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete {
					t.Errorf("expected DELETE, got %s", r.Method)
				}
				expectedPath := "/api/events/" + tt.eventID
				if r.URL.Path != expectedPath {
					t.Errorf("expected %s, got %s", expectedPath, r.URL.Path)
				}
				w.WriteHeader(tt.status)
				if tt.status != http.StatusNoContent {
					_, _ = w.Write([]byte(`{"error":"not found"}`))
				}
			}))
			defer srv.Close()

			client := NewClient(srv.URL)
			err := client.DeleteEvent(tt.eventID)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSubscribeURL(t *testing.T) {
	client := NewClient("https://cal.example.com")
	got := client.SubscribeURL("my-token")
	want := "https://cal.example.com/cal/my-token.ics"
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestAPIError(t *testing.T) {
	err := &APIError{StatusCode: 404, Message: "not found"}
	want := "cal api: 404 not found"
	if err.Error() != want {
		t.Errorf("expected %q, got %q", want, err.Error())
	}
}

func TestParseError(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		status  int
		wantMsg string
	}{
		{
			name:    "json error response",
			body:    `{"error":"bad request"}`,
			status:  400,
			wantMsg: "bad request",
		},
		{
			name:    "plain text response",
			body:    "something went wrong",
			status:  500,
			wantMsg: "something went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			recorder.WriteHeader(tt.status)
			_, _ = recorder.Write([]byte(tt.body))
			resp := recorder.Result()

			apiErr := parseError(resp)
			if apiErr == nil {
				t.Fatal("expected error, got nil")
			}
			ae, ok := apiErr.(*APIError)
			if !ok {
				t.Fatalf("expected *APIError, got %T", apiErr)
			}
			if ae.StatusCode != tt.status {
				t.Errorf("expected status %d, got %d", tt.status, ae.StatusCode)
			}
			if ae.Message != tt.wantMsg {
				t.Errorf("expected message %q, got %q", tt.wantMsg, ae.Message)
			}
		})
	}
}

// mustJSON marshals v to JSON for use in test table data.
func mustJSON(t *testing.T, v interface{}) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal test data: %v", err)
	}
	return string(b)
}
