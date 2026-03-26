package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type App struct {
	Slug             string          `json:"slug"`
	Name             string          `json:"name"`
	Description      string          `json:"description"`
	Readme           string          `json:"readme,omitempty"`
	Version          string          `json:"version"`
	Author           string          `json:"author"`
	IconURL          string          `json:"icon_url"`
	Homepage         string          `json:"homepage"`
	WebhookURL       string          `json:"webhook_url,omitempty"`
	OAuthSetupURL    string          `json:"oauth_setup_url,omitempty"`
	OAuthRedirectURL string          `json:"oauth_redirect_url,omitempty"`
	Tools            json.RawMessage `json:"tools"`
	Events           json.RawMessage `json:"events"`
	Scopes           json.RawMessage `json:"scopes"`
}

type Manifest struct {
	Version   int    `json:"version"`
	UpdatedAt string `json:"updated_at"`
	Apps      []App  `json:"apps"`
}

// Source represents a registry source with its URL and cached data
type Source struct {
	URL       string
	Name      string
	mu        sync.Mutex
	cache     *Manifest
	cacheTime time.Time
}

type Client struct {
	sources []*Source
	mu      sync.RWMutex
	ttl     time.Duration
	client  *http.Client
}

func NewClient(ttl time.Duration) *Client {
	if ttl == 0 {
		ttl = 1 * time.Hour
	}
	return &Client{
		ttl:    ttl,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) AddSource(name, url string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sources = append(c.sources, &Source{URL: url, Name: name})
}

func (c *Client) SetSources(sources []struct{ Name, URL string }) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sources = make([]*Source, len(sources))
	for i, s := range sources {
		c.sources[i] = &Source{URL: s.URL, Name: s.Name}
	}
}

// ListApps returns all apps from all sources, with source URL attached.
// Returns an error only if ALL sources fail. Partial failures are skipped.
func (c *Client) ListApps() ([]AppWithSource, error) {
	c.mu.RLock()
	sources := make([]*Source, len(c.sources))
	copy(sources, c.sources)
	c.mu.RUnlock()

	if len(sources) == 0 {
		return nil, nil
	}

	var result []AppWithSource
	var lastErr error
	successCount := 0
	for _, src := range sources {
		apps, err := c.fetchSource(src)
		if err != nil {
			lastErr = err
			continue
		}
		successCount++
		for _, app := range apps {
			result = append(result, AppWithSource{App: app, RegistryURL: src.URL, RegistryName: src.Name})
		}
	}
	if successCount == 0 && lastErr != nil {
		return nil, fmt.Errorf("all registry sources failed: %w", lastErr)
	}
	return result, nil
}

type AppWithSource struct {
	App
	RegistryURL  string `json:"registry_url"`
	RegistryName string `json:"registry_name"`
}

func (c *Client) GetApp(slug string) (*AppWithSource, error) {
	apps, err := c.ListApps()
	if err != nil {
		return nil, err
	}
	for i := range apps {
		if apps[i].Slug == slug {
			return &apps[i], nil
		}
	}
	return nil, nil
}

func (c *Client) fetchSource(src *Source) ([]App, error) {
	src.mu.Lock()
	defer src.mu.Unlock()

	if src.cache != nil && time.Since(src.cacheTime) < c.ttl {
		return src.cache.Apps, nil
	}

	url := src.URL + "/api/registry/v1/apps.json"
	resp, err := c.client.Get(url)
	if err != nil {
		if src.cache != nil {
			return src.cache.Apps, nil
		}
		return nil, fmt.Errorf("registry fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if src.cache != nil {
			return src.cache.Apps, nil
		}
		return nil, fmt.Errorf("registry returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		if src.cache != nil {
			return src.cache.Apps, nil
		}
		return nil, fmt.Errorf("registry read failed: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		if src.cache != nil {
			return src.cache.Apps, nil
		}
		return nil, fmt.Errorf("registry parse failed: %w", err)
	}

	src.cache = &manifest
	src.cacheTime = time.Now()
	return manifest.Apps, nil
}
