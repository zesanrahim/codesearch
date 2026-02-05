package database

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/joho/godotenv"
	supabase "github.com/lengzuo/supa"
)

var (
	once   sync.Once
	client *supabase.Client
)

func initSupabase() *supabase.Client {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	conf := supabase.Config{
		ApiKey:     os.Getenv("SUPABASE_API_KEY"),
		ProjectRef: os.Getenv("SUPABASE_URL"),
	}

	client, err = supabase.New(conf)
	if err != nil {
		fmt.Println("Failed to connect to  client")
	}

	fmt.Println("Supabase client initialized successfully!")
	return client
}

// in go, if a function starts with lower case its private
func GetClient() *supabase.Client {

	once.Do(func() {
		initSupabase()
	})
	return client
}
