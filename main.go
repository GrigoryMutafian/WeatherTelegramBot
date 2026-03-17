package main

import (
	"log"
	"os"
	"weatherbot/clients/openweather"
	"weatherbot/handler"
	"weatherbot/storage"

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

	db, err := storage.New()
	if err != nil {
		log.Panic(err)
	}

	defer db.Close()

	err = db.CreateUserTable()
	if err != nil {
		log.Fatal("failed to create table: ", err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	owClient := openweather.New(os.Getenv("OPENWEATHERAPI"))

	botHandler := handler.New(bot, owClient)

	botHandler.Start()
}
