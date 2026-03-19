package handler

import (
	"fmt"
	"log"
	"math"
	"weatherbot/clients/openweather"
	"weatherbot/storage"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler struct {
	bot      *tgbotapi.BotAPI
	owClient *openweather.OpenWeatherClient
	storage  *storage.Storage
}

func New(bot *tgbotapi.BotAPI, owClient *openweather.OpenWeatherClient, storage *storage.Storage) *Handler {
	return &Handler{
		bot:      bot,
		owClient: owClient,
		storage:  storage,
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

		var Sender openweather.Sender

		Sender.ID = update.Message.From.ID
		Sender.City = update.Message.Text

		if update.Message.Text == "/start" {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Введите город или населённый пункт для получения прогноза погоды")
			h.bot.Send(msg)
			return
		}

		if update.Message.Text == "/weather" || update.Message.Text == "Погода" {
			city, check, err := h.storage.GetUserData(Sender.ID)
			Sender.City = city
			if err != nil {
				log.Println(err)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Не смогли вспомнить Ваш город, попробуйте ввести его ещё в чате")
				msg.ReplyToMessageID = update.Message.MessageID
				h.bot.Send(msg)
				return
			}
			if check == false {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Введите город или населённый пункт для получения прогноза погоды")
				msg.ReplyToMessageID = update.Message.MessageID
				h.bot.Send(msg)
				return
			}
		}
		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
		coordinates, country, err := h.owClient.Coordinates(Sender.City)
		country = h.owClient.CountryName(country)
		if err != nil {
			log.Println(err)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Не смогли получить верные координаты населённого пункта, убедитесь, что написали название города верно")
			msg.ReplyToMessageID = update.Message.MessageID
			h.bot.Send(msg)
			return
		}
		weather, err := h.owClient.Weather(coordinates.Lat, coordinates.Lon)
		if err != nil {
			log.Println(err)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Не смогли получить показатель температуры Вашего населённого пункта, убедитесь, что написали название города верно")
			msg.ReplyToMessageID = update.Message.MessageID
			h.bot.Send(msg)
			return
		}

		msg := tgbotapi.NewMessage(
			update.Message.Chat.ID,
			fmt.Sprintf("Температура в населённом пункте %s, %s: %d °C", Sender.City, country, int(math.Round(weather.Temp-273.15))))
		msg.ReplyToMessageID = update.Message.MessageID
		h.bot.Send(msg)

		err = h.storage.SaveSender(Sender.ID, Sender.City)
		if err != nil {
			log.Println("failed to save sender:", err)
		}
	}
}
