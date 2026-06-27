package database

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/supabase-community/supabase-go"
)

// apiClient is shared across all DB calls. The 10 s timeout prevents a slow
// Supabase response from leaking a goroutine indefinitely.
var apiClient = &http.Client{Timeout: 10 * time.Second}

// DB wraps the Supabase client and stores credentials for direct RPC calls.
type DB struct {
	Client     *supabase.Client
	url        string
	serviceKey string
}

// Connect initialises the Supabase client using the project URL and service key.
func Connect(supabaseURL, supabaseKey string) (*DB, error) {
	client, err := supabase.NewClient(supabaseURL, supabaseKey, &supabase.ClientOptions{})
	if err != nil {
		return nil, err
	}
	return &DB{Client: client, url: supabaseURL, serviceKey: supabaseKey}, nil
}

// CallRPC invokes a Postgres function via the PostgREST /rpc/ endpoint using the
// service-role key. This bypasses RLS — only call with a user_id that has already
// been verified by the JWT middleware.
func (db *DB) CallRPC(ctx context.Context, fn string, params []byte) ([]byte, error) {
	url := db.url + "/rest/v1/rpc/" + fn

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(params))
	if err != nil {
		return nil, fmt.Errorf("callrpc build request: %w", err)
	}
	req.Header.Set("apikey", db.serviceKey)
	req.Header.Set("Authorization", "Bearer "+db.serviceKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := apiClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("callrpc %s: %w", fn, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("callrpc %s read body: %w", fn, err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("callrpc %s failed (HTTP %d): %s", fn, resp.StatusCode, body)
	}
	return body, nil
}

// ActiveAuctionID returns the id of the currently active auction for the given
// league. Returns an error if no active auction exists.
func (db *DB) ActiveAuctionID(ctx context.Context, leagueID string) (string, error) {
	apiURL := db.url + "/rest/v1/auctions?league_id=eq." + leagueID + "&status=eq.active&select=id&limit=1"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("active auction request: %w", err)
	}
	req.Header.Set("apikey", db.serviceKey)
	req.Header.Set("Authorization", "Bearer "+db.serviceKey)

	resp, err := apiClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("active auction fetch: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("active auction read: %w", err)
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("active auction query failed (HTTP %d): %s", resp.StatusCode, body)
	}

	var rows []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &rows); err != nil || len(rows) == 0 {
		return "", fmt.Errorf("no active auction for league %s", leagueID)
	}
	return rows[0].ID, nil
}
