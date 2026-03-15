package main

import (
	"log"
	"os"
	"weatherbot/clients/openweather"
	"weatherbot/handler"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	bot, err := tgbotapi.NewBotAPI(os.Getenv("BOT_TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	owClient := openweather.New(os.Getenv("OPENWEATHERAPI"))

	botHandler := handler.New(bot, owClient)

	for update := range updates {
		botHandler.HandlerUpdate(update)
	}
}
