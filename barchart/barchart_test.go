package barchart_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tamnd/barchart-cli/barchart"
)

// avResponse is a minimal Alpha Vantage response used in tests.
var avResponse = map[string]any{
	"metadata":    "Test data",
	"last_updated": "2024-01-10 16:00:00 US/Eastern",
	"top_gainers": []map[string]string{
		{"ticker": "AAA", "price": "10.00", "change_amount": "2.00", "change_percentage": "25.00%", "volume": "1000000"},
		{"ticker": "BBB", "price": "5.00", "change_amount": "1.00", "change_percentage": "20.00%", "volume": "500000"},
		{"ticker": "CCC", "price": "3.00", "change_amount": "0.50", "change_percentage": "20.00%", "volume": "250000"},
	},
	"top_losers": []map[string]string{
		{"ticker": "DDD", "price": "8.00", "change_amount": "-2.00", "change_percentage": "-20.00%", "volume": "2000000"},
		{"ticker": "EEE", "price": "4.00", "change_amount": "-1.00", "change_percentage": "-20.00%", "volume": "750000"},
	},
	"most_actively_traded": []map[string]string{
		{"ticker": "FFF", "price": "100.00", "change_amount": "1.00", "change_percentage": "1.00%", "volume": "50000000"},
		{"ticker": "GGG", "price": "50.00", "change_amount": "-0.50", "change_percentage": "-1.00%", "volume": "30000000"},
	},
}

func newTestServer(t *testing.T) (*httptest.Server, barchart.Config) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request carried no User-Agent")
		}
		w.Header().Set("Content-Type", "application/json")
		b, _ := json.Marshal(avResponse)
		_, _ = w.Write(b)
	}))
	cfg := barchart.DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	return srv, cfg
}

func TestGainers(t *testing.T) {
	srv, cfg := newTestServer(t)
	defer srv.Close()

	c := barchart.NewClient(cfg)
	quotes, err := c.Gainers(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(quotes) != 3 {
		t.Errorf("got %d gainers, want 3", len(quotes))
	}
	if quotes[0].Symbol != "AAA" {
		t.Errorf("first gainer = %q, want AAA", quotes[0].Symbol)
	}
}

func TestGainersLimit(t *testing.T) {
	srv, cfg := newTestServer(t)
	defer srv.Close()

	c := barchart.NewClient(cfg)
	quotes, err := c.Gainers(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(quotes) != 2 {
		t.Errorf("got %d gainers with limit=2, want 2", len(quotes))
	}
}

func TestLosers(t *testing.T) {
	srv, cfg := newTestServer(t)
	defer srv.Close()

	c := barchart.NewClient(cfg)
	quotes, err := c.Losers(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(quotes) != 2 {
		t.Errorf("got %d losers, want 2", len(quotes))
	}
	if quotes[0].Symbol != "DDD" {
		t.Errorf("first loser = %q, want DDD", quotes[0].Symbol)
	}
}

func TestActive(t *testing.T) {
	srv, cfg := newTestServer(t)
	defer srv.Close()

	c := barchart.NewClient(cfg)
	quotes, err := c.Active(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(quotes) != 2 {
		t.Errorf("got %d active, want 2", len(quotes))
	}
	if quotes[0].Symbol != "FFF" {
		t.Errorf("first active = %q, want FFF", quotes[0].Symbol)
	}
}

func TestRetryOn503(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		b, _ := json.Marshal(avResponse)
		_, _ = w.Write(b)
	}))
	defer srv.Close()

	cfg := barchart.DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	cfg.Retries = 5

	c := barchart.NewClient(cfg)
	start := time.Now()
	quotes, err := c.Gainers(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(quotes) == 0 {
		t.Error("expected quotes after retries")
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
	if time.Since(start) < 500*time.Millisecond {
		t.Error("retries did not back off")
	}
}
