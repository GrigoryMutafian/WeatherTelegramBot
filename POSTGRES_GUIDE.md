# Гайд: Подключение PostgreSQL и работа с User ID в Telegram боте

## Содержание
1. [Установка зависимостей](#1-установка-зависимостей)
2. [Настройка подключения к PostgreSQL](#2-настройка-подключения-к-postgresql)
3. [Получение User ID из сообщения](#3-получение-user-id-из-сообщения)
4. [Пример интеграции в проект](#4-пример-интеграции-в-проект)

---

## 1. Установка зависимостей

### Установка драйвера PostgreSQL для Go

```bash
go get github.com/lib/pq
```

Это установит драйвер `pq` - популярный чистый Go драйвер для PostgreSQL.

---

## 2. Настройка подключения к PostgreSQL

### 2.1 Добавление переменных окружения

Добавьте в файл `.env`:

```env
# Существующие переменные
BOT_TOKEN=your_bot_token
OPENWEATHERAPI=your_api_key

# Новые переменные для PostgreSQL
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=weatherbot
```

### 2.2 Создание структуры для БД

Создайте файл `storage/storage.go`:

```go
package storage

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

type Storage struct {
	db *sql.DB
}

func New() (*Storage, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) CreateUserTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		telegram_id BIGINT UNIQUE NOT NULL,
		username VARCHAR(255),
		first_name VARCHAR(255),
		last_name VARCHAR(255),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := s.db.Exec(query)
	return err
} 

// SaveUser сохраняет или обновляет пользователя
func (s *Storage) SaveUser(telegramID int64, username, firstName, lastName string) error {
	query := `
	INSERT INTO users (telegram_id, username, first_name, last_name)
	VALUES ($1, $2, $3, $4)
	ON CONFLICT (telegram_id) 
	DO UPDATE SET username = $2, first_name = $3, last_name = $4;
	`

	_, err := s.db.Exec(query, telegramID, username, firstName, lastName)
	return err
}

// GetUser получает пользователя по telegram_id
func (s *Storage) GetUser(telegramID int64) (*User, error) {
	query := `SELECT telegram_id, username, first_name, last_name, created_at FROM users WHERE telegram_id = $1`
	
	user := &User{}
	err := s.db.QueryRow(query, telegramID).Scan(
		&user.TelegramID,
		&user.Username,
		&user.FirstName,
		&user.LastName,
		&user.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

type User struct {
	TelegramID int64
	Username   string
	FirstName  string
	LastName   string
	CreatedAt  string
}
```

---

## 3. Получение User ID из сообщения

### 3.1 Структура сообщения Telegram

Когда пользователь отправляет сообщение, Telegram API возвращает структуру `Update`, которая содержит:

```go
type Update struct {
    Message *Message
    // ... другие поля
}

type Message struct {
    MessageID int
    From      *User    // Информация об отправителе
    Chat      *Chat    // Информация о чате
    Text      string   // Текст сообщения
    // ... другие поля
}

type User struct {
    ID        int64  // User ID - уникальный идентификатор пользователя
    FirstName string
    LastName  string
    UserName  string // @username (может быть пустым)
    // ... другие поля
}
```

### 3.2 Способы получения User ID

```go
func (h *Handler) HandlerUpdate(update tgbotapi.Update) {
    if update.Message != nil {
        // Способ 1: Получение ID напрямую
        userID := update.Message.From.ID
        userName := update.Message.From.UserName
        firstName := update.Message.From.FirstName
        lastName := update.Message.From.LastName
        
        // Способ 2: Chat ID (для групповых чатов может отличаться)
        chatID := update.Message.Chat.ID
        
        log.Printf("User ID: %d", userID)
        log.Printf("Username: @%s", userName)
        log.Printf("First Name: %s", firstName)
        log.Printf("Last Name: %s", lastName)
        log.Printf("Chat ID: %d", chatID)
    }
}
```

### 3.3 Все доступные данные пользователя

| Поле | Тип | Описание |
|------|-----|----------|
| `From.ID` | int64 | Уникальный ID пользователя в Telegram |
| `From.UserName` | string | Username пользователя (без @) |
| `From.FirstName` | string | Имя пользователя |
| `From.LastName` | string | Фамилия пользователя |
| `From.LanguageCode` | string | Код языка (ru, en, и т.д.) |
| `From.IsBot` | bool | Является ли пользователь ботом |
| `Chat.ID` | int64 | ID чата (для личных сообщений = User ID) |

---

## 4. Пример интеграции в проект

### 4.1 Обновленный handler/handler.go

```go
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

func (h *Handler) HandlerUpdate(update tgbotapi.Update) {
	if update.Message != nil {
		// Получаем информацию о пользователе
		userID := update.Message.From.ID
		username := update.Message.From.UserName
		firstName := update.Message.From.FirstName
		lastName := update.Message.From.LastName
		chatID := update.Message.Chat.ID

		// Сохраняем пользователя в БД
		if err := h.storage.SaveUser(userID, username, firstName, lastName); err != nil {
			log.Printf("Failed to save user: %v", err)
		}

		log.Printf("[User ID: %d] [@%s] %s %s: %s", 
			userID, username, firstName, lastName, update.Message.Text)

		if update.Message.Text == "/start" {
			msg := tgbotapi.NewMessage(chatID, 
				fmt.Sprintf("Привет, %s! Введите город для получения прогноза погоды.", firstName))
			msg.ReplyToMessageID = update.Message.MessageID
			h.bot.Send(msg)
			return
		}

		if update.Message.Text == "/myid" {
			msg := tgbotapi.NewMessage(chatID, 
				fmt.Sprintf("Ваш User ID: %d\nВаш Chat ID: %d", userID, chatID))
			h.bot.Send(msg)
			return
		}

		// Остальная логика обработки погоды...
		coordinates, err := h.owClient.Coordinates(update.Message.Text)
		if err != nil {
			log.Println(err)
			msg := tgbotapi.NewMessage(chatID, "Не смогли получить верные координаты города")
			msg.ReplyToMessageID = update.Message.MessageID
			h.bot.Send(msg)
			return
		}

		weather, err := h.owClient.Weather(coordinates.Lat, coordinates.Lon)
		if err != nil {
			log.Println(err)
			msg := tgbotapi.NewMessage(chatID, "Не смогли получить показатель температуры")
			msg.ReplyToMessageID = update.Message.MessageID
			h.bot.Send(msg)
			return
		}

		msg := tgbotapi.NewMessage(
			chatID,
			fmt.Sprintf("Температура в городе %s: %d °C", 
				update.Message.Text, int(math.Round(weather.Temp-273.15))))
		msg.ReplyToMessageID = update.Message.MessageID
		h.bot.Send(msg)
	}
}
```

### 4.2 Обновленный main.go

```go
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

	// Инициализация БД
	db, err := storage.New()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Создание таблицы пользователей
	if err := db.CreateUserTable(); err != nil {
		log.Fatalf("Failed to create users table: %v", err)
	}
	log.Println("Database connected and tables created")

	// Инициализация бота
	bot, err := tgbotapi.NewBotAPI(os.Getenv("BOT_TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	// Инициализация клиентов
	owClient := openweather.New(os.Getenv("OPENWEATHERAPI"))
	botHandler := handler.New(bot, owClient, db)

	botHandler.Start()
}
```

---

## Быстрая шпаргалка

### Получить User ID:
```go
userID := update.Message.From.ID
```

### Получить Chat ID:
```go
chatID := update.Message.Chat.ID
```

### Получить Username:
```go
username := update.Message.From.UserName
```

### Получить имя и фамилию:
```go
firstName := update.Message.From.FirstName
lastName := update.Message.From.LastName
```

---

## SQL команды для работы с PostgreSQL

### Создание БД:
```sql
CREATE DATABASE weatherbot;
```

### Просмотр пользователей:
```sql
SELECT * FROM users;
```

### Поиск пользователя по telegram_id:
```sql
SELECT * FROM users WHERE telegram_id = 123456789;
```

---

## Запуск PostgreSQL через Docker (опционально)

```bash
docker run --name weatherbot-postgres \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=your_password \
  -e POSTGRES_DB=weatherbot \
  -p 5432:5432 \
  -d postgres:15