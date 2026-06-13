// Package barchart is the library behind the bc command: the HTTP client,
// request shaping, and the typed data models for Barchart market data.
//
// Data is sourced from Alpha Vantage's public TOP_GAINERS_LOSERS endpoint,
// which returns top gainers, top losers, and most actively traded US tickers
// without requiring authentication.
package barchart

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// BaseURL is the Alpha Vantage API base used by DefaultConfig.
const BaseURL = "https://www.alphavantage.co"

// DefaultUserAgent identifies the client to the upstream API.
const DefaultUserAgent = "bc/dev (+https://github.com/tamnd/barchart-cli)"

// Config holds constructor parameters for a Client.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Retries   int
	Timeout   time.Duration
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:   BaseURL,
		UserAgent: DefaultUserAgent,
		Rate:      200 * time.Millisecond,
		Retries:   3,
		Timeout:   30 * time.Second,
	}
}

// Client talks to the Alpha Vantage API over HTTP.
type Client struct {
	cfg        Config
	httpClient *http.Client
	last       time.Time
}

// NewClient returns a Client built from cfg.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: cfg.Timeout},
	}
}

// Quote holds a single market mover record.
type Quote struct {
	Symbol          string `json:"ticker"`
	Price           string `json:"price"`
	ChangeAmount    string `json:"change_amount"`
	ChangePercent   string `json:"change_percentage"`
	Volume          string `json:"volume"`
}

// avTopResp is the wire format returned by the TOP_GAINERS_LOSERS function.
type avTopResp struct {
	Metadata         string  `json:"metadata"`
	LastUpdated      string  `json:"last_updated"`
	TopGainers       []Quote `json:"top_gainers"`
	TopLosers        []Quote `json:"top_losers"`
	MostActivelyTraded []Quote `json:"most_actively_traded"`
}

// Gainers returns the top gaining US tickers for the latest trading day.
func (c *Client) Gainers(ctx context.Context, limit int) ([]Quote, error) {
	resp, err := c.fetch(ctx)
	if err != nil {
		return nil, err
	}
	return apply(resp.TopGainers, limit), nil
}

// Losers returns the top losing US tickers for the latest trading day.
func (c *Client) Losers(ctx context.Context, limit int) ([]Quote, error) {
	resp, err := c.fetch(ctx)
	if err != nil {
		return nil, err
	}
	return apply(resp.TopLosers, limit), nil
}

// Active returns the most actively traded US tickers (by volume) for the latest
// trading day.
func (c *Client) Active(ctx context.Context, limit int) ([]Quote, error) {
	resp, err := c.fetch(ctx)
	if err != nil {
		return nil, err
	}
	return apply(resp.MostActivelyTraded, limit), nil
}

func apply(qs []Quote, limit int) []Quote {
	if limit > 0 && limit < len(qs) {
		return qs[:limit]
	}
	return qs
}

func (c *Client) fetch(ctx context.Context) (*avTopResp, error) {
	url := c.cfg.BaseURL + "/query?function=TOP_GAINERS_LOSERS&apikey=demo"
	body, err := c.get(ctx, url)
	if err != nil {
		return nil, err
	}
	// Detect the "demo key" information message that means data is absent.
	if strings.Contains(string(body), `"Information"`) {
		return nil, fmt.Errorf("API returned an information message instead of data; check API key")
	}
	var resp avTopResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if len(resp.TopGainers) == 0 && len(resp.TopLosers) == 0 {
		return nil, fmt.Errorf("empty response from API")
	}
	return &resp, nil
}

func (c *Client) get(ctx context.Context, url string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, url)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", url, lastErr)
}

func (c *Client) do(ctx context.Context, url string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	if c.cfg.Rate <= 0 {
		return
	}
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}
