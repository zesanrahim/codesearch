package database

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/joho/godotenv"
	supabase "github.com/lengzuo/supa"
)

var (
	once    sync.Once
	client  *supabase.Client
	initErr error
)

func initSupabase() (*supabase.Client, error) {
	// A missing .env is not fatal: the credentials may already be exported.
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading .env: %w", err)
	}

	conf := supabase.Config{
		ApiKey:     os.Getenv("SUPABASE_API_KEY"),
		ProjectRef: os.Getenv("SUPABASE_URL"),
	}
	if conf.ApiKey == "" || conf.ProjectRef == "" {
		return nil, errors.New("SUPABASE_API_KEY and SUPABASE_URL must be set")
	}

	c, err := supabase.New(conf)
	if err != nil {
		return nil, fmt.Errorf("connecting to supabase: %w", err)
	}
	return c, nil
}

// GetClient lazily builds the shared Supabase client, returning the same error
// to every caller if initialisation failed.
func GetClient() (*supabase.Client, error) {
	once.Do(func() {
		client, initErr = initSupabase()
	})
	return client, initErr
}
