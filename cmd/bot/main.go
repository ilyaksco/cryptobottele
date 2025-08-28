package main

import (
	"log"

	"cryptowordgamebot/internal/bot"
	"cryptowordgamebot/internal/config"
	"cryptowordgamebot/internal/game"
	"cryptowordgamebot/internal/i18n"
	"cryptowordgamebot/internal/storage"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	log.Println("Starting bot application...")

	cfg, err := config.New()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	log.Println("Configuration loaded successfully.")

	gameCfg, err := game.LoadConfig("game.yaml")
	if err != nil {
		log.Fatalf("Failed to load game configuration: %v", err)
	}
	log.Println("Game configuration loaded successfully.")

	db, err := storage.New(cfg.SupabaseURL, cfg.SupabaseKey)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	log.Println("Storage initialized successfully.")

	translator, err := i18n.New("locales", cfg.DefaultLanguage)
	if err != nil {
		log.Fatalf("Failed to initialize translator: %v", err)
	}
	log.Println("Translator initialized successfully.")
	
	gameSvc := game.NewService(gameCfg)

	themeCfg, err := game.LoadThemes("themes.yaml")
	if err != nil {
		log.Fatalf("Failed to load themes configuration: %v", err)
	}
	log.Println("Themes configuration loaded successfully.")

	api, err := tgbotapi.NewBotAPI(cfg.TelegramBotToken)
	if err != nil {
		log.Fatalf("Failed to connect to Telegram: %v", err)
	}
	log.Printf("Authorized on account %s", api.Self.UserName)

	handler := bot.NewBotHandler(api, translator, cfg, db, gameSvc, themeCfg)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	u.AllowedUpdates = []string{"message", "callback_query"}
	updates := api.GetUpdatesChan(u)

	for update := range updates {
		handler.HandleUpdate(update)
	}
}