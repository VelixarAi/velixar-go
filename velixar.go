// Package velixar provides a Go client for the Velixar AI memory platform.
//
// Usage:
//
//	client := velixar.New("your-api-key")
//	id, err := client.Store(ctx, "important fact", velixar.WithTags("project", "go"))
//	results, err := client.Search(ctx, "important", velixar.WithLimit(5))
package velixar

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

const (
	DefaultBaseURL = "https://api.velixarai.com"
	DefaultTimeout = 30 * time.Second
	Version        = "1.0.0"
)

// Client is the Velixar API client.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// Option configures the client.
type Option func(*Client)

// WithBaseURL sets a custom API base URL.
func WithBaseURL(u string) Option { return func(c *Client) { c.baseURL = u } }

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.httpClient = h } }

// New creates a new Velixar client. If apiKey is empty, reads VELIXAR_API_KEY.
func New(apiKey string, opts ...Option) *Client {
	if apiKey == "" {
		apiKey = os.Getenv("VELIXAR_API_KEY")
	}
	c := &Client{
		apiKey:     apiKey,
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: DefaultTimeout},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Memory represents a stored memory.
type Memory struct {
	ID         string   `json:"id"`
	Content    string   `json:"content"`
	Tags       []string `json:"tags"`
	Tier       int      `json:"tier"`
	Score      float64  `json:"score,omitempty"`
	MemoryType string   `json:"memory_type,omitempty"`
	CreatedAt  string   `json:"created_at,omitempty"`
	UpdatedAt  string   `json:"updated_at,omitempty"`
}

// SearchResult holds search results.
type SearchResult struct {
	Memories []Memory `json:"memories"`
	Count    int      `json:"count"`
}

// GraphEntity represents a knowledge graph entity.
type GraphEntity struct {
	ID         string                 `json:"id"`
	EntityType string                 `json:"entity_type"`
	Label      string                 `json:"label"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// GraphEdge represents a knowledge graph relationship.
type GraphEdge struct {
	Source       string  `json:"source"`
	Target       string  `json:"target"`
	RelationType string  `json:"relation_type"`
	Weight       float64 `json:"weight,omitempty"`
}

// TraverseResult holds graph traversal results.
type TraverseResult struct {
	Nodes []GraphEntity `json:"nodes"`
	Edges []GraphEdge   `json:"edges"`
}

// HealthStatus holds API health info.
type HealthStatus struct {
	Status  string `json:"status"`
	Qdrant  bool   `json:"qdrant"`
	Redis   bool   `json:"redis"`
	Search  bool   `json:"search"`
	Version int    `json:"version"`
}

// StoreOption configures a Store call.
type StoreOption func(map[string]interface{})

// WithTags adds tags to a store call.
func WithTags(tags ...string) StoreOption {
	return func(m map[string]interface{}) { m["tags"] = tags }
}

// WithTier sets the memory tier (0=pinned, 1=session, 2=semantic, 3=org).
func WithTier(tier int) StoreOption {
	return func(m map[string]interface{}) { m["tier"] = tier }
}

// WithUserID sets the user ID.
func WithUserID(id string) StoreOption {
	return func(m map[string]interface{}) { m["user_id"] = id }
}

// SearchOption configures a Search call.
type SearchOption func(url.Values)

// WithLimit sets the max results.
func WithLimit(n int) SearchOption {
	return func(v url.Values) { v.Set("limit", strconv.Itoa(n)) }
}

// Error types.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("velixar: HTTP %d: %s", e.StatusCode, e.Message)
}

// do executes an HTTP request.
func (c *Client) do(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("velixar: marshal: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("velixar: request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "velixar-go/"+Version)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("velixar: http: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("velixar: read: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, &APIError{StatusCode: resp.StatusCode, Message: string(data)}
	}
	return data, nil
}

// Store saves a memory. Returns the memory ID.
func (c *Client) Store(ctx context.Context, content string, opts ...StoreOption) (string, error) {
	body := map[string]interface{}{"content": content, "tier": 2}
	for _, o := range opts {
		o(body)
	}
	data, err := c.do(ctx, "POST", "/memory", body)
	if err != nil {
		return "", err
	}
	var res struct{ ID string `json:"id"` }
	if err := json.Unmarshal(data, &res); err != nil {
		return "", err
	}
	return res.ID, nil
}

// Search finds memories by semantic similarity.
func (c *Client) Search(ctx context.Context, query string, opts ...SearchOption) (*SearchResult, error) {
	v := url.Values{"q": {query}}
	for _, o := range opts {
		o(v)
	}
	data, err := c.do(ctx, "GET", "/memory/search?"+v.Encode(), nil)
	if err != nil {
		return nil, err
	}
	var res SearchResult
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// Get retrieves a single memory by ID.
func (c *Client) Get(ctx context.Context, id string) (*Memory, error) {
	data, err := c.do(ctx, "GET", "/memory/"+id, nil)
	if err != nil {
		return nil, err
	}
	var m Memory
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// Update modifies a memory's content or tags.
func (c *Client) Update(ctx context.Context, id string, content *string, tags []string) error {
	body := map[string]interface{}{}
	if content != nil {
		body["content"] = *content
	}
	if tags != nil {
		body["tags"] = tags
	}
	_, err := c.do(ctx, "PATCH", "/memory/"+id, body)
	return err
}

// Delete removes a memory.
func (c *Client) Delete(ctx context.Context, id string) error {
	_, err := c.do(ctx, "DELETE", "/memory/"+id, nil)
	return err
}

// List returns recent memories.
func (c *Client) List(ctx context.Context, limit int) (*SearchResult, error) {
	v := url.Values{"limit": {strconv.Itoa(limit)}}
	data, err := c.do(ctx, "GET", "/memory/list?"+v.Encode(), nil)
	if err != nil {
		return nil, err
	}
	var res SearchResult
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// GraphTraverse walks relationships from an entity.
func (c *Client) GraphTraverse(ctx context.Context, entity string, depth int) (*TraverseResult, error) {
	v := url.Values{"entity": {entity}, "depth": {strconv.Itoa(depth)}}
	data, err := c.do(ctx, "GET", "/v1/graph/traverse?"+v.Encode(), nil)
	if err != nil {
		return nil, err
	}
	var res TraverseResult
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// Health checks API status.
func (c *Client) Health(ctx context.Context) (*HealthStatus, error) {
	data, err := c.do(ctx, "GET", "/health", nil)
	if err != nil {
		return nil, err
	}
	var h HealthStatus
	if err := json.Unmarshal(data, &h); err != nil {
		return nil, err
	}
	return &h, nil
}
