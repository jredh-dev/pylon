package cal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client talks to the cal service API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a cal API client.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Feed represents a calendar feed.
type Feed struct {
	ID        string    `json:"ID"`
	Name      string    `json:"Name"`
	Token     string    `json:"Token"`
	CreatedAt time.Time `json:"CreatedAt"`
	UpdatedAt time.Time `json:"UpdatedAt"`
}

// CreateFeedResponse is the response from creating a feed.
type CreateFeedResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Token string `json:"token"`
	URL   string `json:"url"`
}

// Event represents a calendar event.
type Event struct {
	ID          string     `json:"ID"`
	FeedID      string     `json:"FeedID"`
	Summary     string     `json:"Summary"`
	Description string     `json:"Description"`
	Location    string     `json:"Location"`
	URL         string     `json:"URL"`
	Start       time.Time  `json:"Start"`
	End         *time.Time `json:"End"`
	AllDay      bool       `json:"AllDay"`
	Deadline    *time.Time `json:"Deadline"`
	Status      string     `json:"Status"`
	Categories  string     `json:"Categories"`
	CreatedAt   time.Time  `json:"CreatedAt"`
	UpdatedAt   time.Time  `json:"UpdatedAt"`
}

// CreateEventRequest is the payload for creating an event.
type CreateEventRequest struct {
	FeedID      string `json:"feed_id"`
	Summary     string `json:"summary"`
	Description string `json:"description,omitempty"`
	Location    string `json:"location,omitempty"`
	URL         string `json:"url,omitempty"`
	Start       string `json:"start"`
	End         string `json:"end,omitempty"`
	AllDay      bool   `json:"all_day,omitempty"`
	Deadline    string `json:"deadline,omitempty"`
	Status      string `json:"status,omitempty"`
	Categories  string `json:"categories,omitempty"`
}

// APIError is returned when the API responds with an error.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("cal api: %d %s", e.StatusCode, e.Message)
}

// CreateFeed creates a new calendar feed. If slug is non-empty, it is used as
// a readable token for the subscription URL (e.g. "my-calendar" ->
// /cal/my-calendar.ics). Otherwise the server generates a UUID token.
func (c *Client) CreateFeed(name, slug string) (*CreateFeedResponse, error) {
	payload := map[string]string{"name": name}
	if slug != "" {
		payload["slug"] = slug
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.post("/api/feeds", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, parseError(resp)
	}

	var feed CreateFeedResponse
	if err := json.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &feed, nil
}

// ListFeeds returns all feeds.
func (c *Client) ListFeeds() ([]Feed, error) {
	resp, err := c.get("/api/feeds")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}

	var feeds []Feed
	if err := json.NewDecoder(resp.Body).Decode(&feeds); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return feeds, nil
}

// DeleteFeed deletes a feed by ID.
func (c *Client) DeleteFeed(id string) error {
	resp, err := c.delete("/api/feeds/" + id)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return parseError(resp)
	}
	return nil
}

// CreateEvent creates a new event.
func (c *Client) CreateEvent(req *CreateEventRequest) (*Event, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.post("/api/events", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, parseError(resp)
	}

	var event Event
	if err := json.NewDecoder(resp.Body).Decode(&event); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &event, nil
}

// ListEvents returns all events for a feed.
func (c *Client) ListEvents(feedID string) ([]Event, error) {
	resp, err := c.get("/api/feeds/" + feedID + "/events")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}

	var events []Event
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return events, nil
}

// DeleteEvent deletes an event by ID.
func (c *Client) DeleteEvent(id string) error {
	resp, err := c.delete("/api/events/" + id)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return parseError(resp)
	}
	return nil
}

// SubscribeURL returns the webcal subscription URL for a feed token.
func (c *Client) SubscribeURL(token string) string {
	return c.baseURL + "/cal/" + token + ".ics"
}

// --- HTTP helpers ---

func (c *Client) get(path string) (*http.Response, error) {
	return c.httpClient.Get(c.baseURL + path)
}

func (c *Client) post(path string, body []byte) (*http.Response, error) {
	return c.httpClient.Post(c.baseURL+path, "application/json", bytes.NewReader(body))
}

func (c *Client) delete(path string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	return c.httpClient.Do(req)
}

func parseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var errResp struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
		return &APIError{StatusCode: resp.StatusCode, Message: errResp.Error}
	}
	return &APIError{StatusCode: resp.StatusCode, Message: string(body)}
}
