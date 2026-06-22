package database

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/supabase-community/supabase-go"
)

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

	resp, err := http.DefaultClient.Do(req)
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
