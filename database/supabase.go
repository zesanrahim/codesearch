package database

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	supa "github.com/lengzuo/supa"
)

func InitSupabase() *supa.Client {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	conf := supa.Config{
		ApiKey:     os.Getenv("SUPABASE_API_KEY"),
		ProjectRef: os.Getenv("SUPABASE_URL"),
	}

	supaClient, err := supa.New(conf)
	if err != nil {
		fmt.Println("Failed to connect to  client")
	}

	fmt.Println("Supabase client initialized successfully!")
	return supaClient
}
