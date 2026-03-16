package handler

import (
	"fmt"
	"log"
	"math"
	"weatherbot/clients/openweather"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler struct {
	bot      *tgbotapi.BotAPI
	owClient *openweather.OpenWeatherClient
}

func New(bot *tgbotapi.BotAPI, owClient *openweather.OpenWeatherClient) *Handler {
	return &Handler{
		bot:      bot,
		owClient: owClient,
	}
}

func (h *Handler) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := h.bot.GetUpdatesChan(u)
	for update := range updates {
		h.HandlerUpdate(update)
	}
}

func (h *Handler) HandlerUpdate(update tgbotapi.Update) {
	if update.Message != nil {
		if update.Message.Text == "/start" {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Введите город для получения прогноза погоды")
			msg.ReplyToMessageID = update.Message.MessageID
			h.bot.Send(msg)
			return
		}
		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
		coordinates, err := h.owClient.Coordinates(update.Message.Text)
		if err != nil {
			log.Println(err)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Не смогли получить верные координаты города, убедитесь что написали название города верно")
			msg.ReplyToMessageID = update.Message.MessageID
			h.bot.Send(msg)
			return
		}
		weather, err := h.owClient.Weather(coordinates.Lat, coordinates.Lon)
		if err != nil {
			log.Println(err)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Не смогли получить показатель температуры в Вашем городе, убедитесь что написали название города верно")
			msg.ReplyToMessageID = update.Message.MessageID
			h.bot.Send(msg)
			return
		}

		msg := tgbotapi.NewMessage(
			update.Message.Chat.ID,
			fmt.Sprintf("Температура в городе %s: %d °C", update.Message.Text, int(math.Round(weather.Temp-273.15))))
		msg.ReplyToMessageID = update.Message.MessageID
		h.bot.Send(msg)
	}
}
