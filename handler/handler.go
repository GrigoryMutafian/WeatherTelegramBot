package handler

import (
	"fmt"
	"log"
	"math"
	"sync"
	"weatherbot/clients/glm"
	"weatherbot/clients/notifications"
	"weatherbot/clients/openweather"
	"weatherbot/storage"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler struct {
	bot             *tgbotapi.BotAPI
	owClient        *openweather.OpenWeatherClient
	storage         *storage.Storage
	GLM             *glm.Client
	notificationMgr *notifications.Manager
}

func New(bot *tgbotapi.BotAPI, owClient *openweather.OpenWeatherClient, storage *storage.Storage, glmClient *glm.Client) *Handler {
	h := &Handler{
		bot:      bot,
		owClient: owClient,
		storage:  storage,
		GLM:      glmClient,
	}

	notifMgr := notifications.New(
		h.sendTelegramMessage, // функция отправки сообщения
		h.getWeatherForCity,   // функция получения погоды
	)
	notifMgr.Start()
	h.notificationMgr = notifMgr

	return h
}

func (h *Handler) sendTelegramMessage(userID int64, message string) error {
	msg := tgbotapi.NewMessage(userID, message)
	_, err := h.bot.Send(msg)
	return err
}

func (h *Handler) getWeatherForCity(city string) (string, error) {
	coordinates, country, err := h.owClient.Coordinates(city)
	if err != nil {
		return "", err
	}

	weather, err := h.owClient.Weather(coordinates.Lat, coordinates.Lon)
	if err != nil {
		return "", err
	}

	country = h.owClient.CountryName(country)
	return fmt.Sprintf("Температура в %s, %s: %d °C", city, country, int(math.Round(weather.Temp-273.15))), nil
}

func (h *Handler) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := h.bot.GetUpdatesChan(u)
	for update := range updates {
		h.HandlerUpdate(update)
	}
}

var (
	mu sync.Mutex
)

func (h *Handler) HandlerUpdate(update tgbotapi.Update) {
	if update.Message != nil {
		mu.Lock()
		defer mu.Unlock()

		var sender openweather.Sender
		sender.ID = update.Message.From.ID
		sender.City = update.Message.Text

		count, err := h.storage.GetRequestCount(sender.ID)
		if err != nil {
			log.Println("error while getting request count:", err)
			return
		}

		if count == 0 {
			h.storage.ResetRequestCount(sender.ID)
		}

		if count > 4 {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Вы привысили количество запросов за день. Возвращайтесь завтра.")
			h.bot.Send(msg)
			return
		}

		if update.Message.Text == "/start" {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Введите город или населённый пункт для получения прогноза погоды")
			h.bot.Send(msg)
			return
		}

		if update.Message.Text == "/subscribe" {
			h.handleSubscribe(update)
			return
		}

		if update.Message.Text == "/unsubscribe" {
			h.handleUnsubscribe(update)
			return
		}

		if update.Message.Text == "/weather" || update.Message.Text == "Погода" {
			city, check, err := h.storage.GetUserData(sender.ID)
			sender.City = city
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
		coordinates, country, err := h.owClient.Coordinates(sender.City)
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
		response := fmt.Sprintf("Температура в населённом пункте %s, %s: %d °C", sender.City, country, int(math.Round(weather.Temp-273.15)))

		err = h.storage.SaveSender(sender.ID, sender.City)
		if err != nil {
			log.Println("failed to save sender:", err)
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Один момент...")
		sentMsg, _ := h.bot.Send(msg)
		systemPrompt := glm.SystemPrompt
		userMessage := response

		answer, err := h.GLM.ChatWithPrompt(systemPrompt, userMessage)
		if err != nil {
			log.Println("failed getting AI answer", err)
			return
		}
		editMsg := tgbotapi.NewEditMessageText(update.Message.Chat.ID, sentMsg.MessageID, answer)
		h.bot.Send(editMsg)
		err = h.storage.SaveSender(sender.ID, sender.City)
		if err != nil {
			log.Println("failed to save sender:", err)
		}

		err = h.storage.IncreamentRequestCount(sender.ID)
		if err != nil {
			log.Println("failed to increament requests count:", err)

		}
	}
}

func (h *Handler) handleSubscribe(update tgbotapi.Update) {
	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	city, hasCity, err := h.storage.GetUserData(userID)
	if err != nil {
		log.Println("error getting user data:", err)
		msg := tgbotapi.NewMessage(chatID, "Произошла ошибка при получении данных пользователя")
		h.bot.Send(msg)
		return
	}

	if !hasCity {
		msg := tgbotapi.NewMessage(chatID, "Сначала укажите свой город, отправив его название в чат")
		h.bot.Send(msg)
		return
	}

	// Подписываем на уведомления в 8:00 утра (по Москве)
	h.notificationMgr.Subscribe(userID, city, 8)

	msg := tgbotapi.NewMessage(chatID,
		fmt.Sprintf("✅ Вы подписались на ежедневные уведомления о погоде в городе %s!\n"+
			"Уведомление будет приходить каждый день в 8:00 утра (по Москве).\n"+
			"Чтобы отписаться, используйте команду /unsubscribe", city))
	h.bot.Send(msg)
}

func (h *Handler) handleUnsubscribe(update tgbotapi.Update) {
	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID

	h.notificationMgr.Unsubscribe(userID)

	msg := tgbotapi.NewMessage(chatID, "❌ Вы отписались от ежедневных уведомлений о погоде")
	h.bot.Send(msg)
}
