package config

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramBotToken string
	SupabaseURL      string
	SupabaseKey      string
	DefaultLanguage  string
}

func New() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, errors.New("error loading .env file")
	}

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		return nil, errors.New("TELEGRAM_BOT_TOKEN is not set in .env file")
	}

	return &Config{
		TelegramBotToken: token,
		SupabaseURL:      os.Getenv("SUPABASE_URL"),
		SupabaseKey:      os.Getenv("SUPABASE_KEY"),
		DefaultLanguage:  os.Getenv("DEFAULT_LANGUAGE"),
	}, nil
}