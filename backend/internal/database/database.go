package database

import (
	"github.com/supabase-community/supabase-go"
)

// DB wraps the Supabase client.
type DB struct {
	Client *supabase.Client
}

// Connect initializes the Supabase client using the project URL and service/anon key.
func Connect(supabaseURL, supabaseKey string) (*DB, error) {
	client, err := supabase.NewClient(supabaseURL, supabaseKey, &supabase.ClientOptions{})
	if err != nil {
		return nil, err
	}

	return &DB{Client: client}, nil
}
