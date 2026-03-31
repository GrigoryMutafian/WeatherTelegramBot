package storage

import (
	"database/sql"
	"fmt"
	"os"
	"time"

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
    city VARCHAR(100),
    request_count INT DEFAULT 0,
    last_reset_date DATE DEFAULT CURRENT_DATE
);

	CREATE TABLE IF NOT EXISTS notifications (
		id SERIAL PRIMARY KEY,
		user_id BIGINT UNIQUE,
		city VARCHAR(100),
		notify_hour INT DEFAULT 8,
		enabled BOOLEAN DEFAULT true,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);`

	_, err := s.db.Exec(query)
	return err
}

func (s *Storage) SaveSender(id int64, city string) error {
	query := `INSERT INTO users(id, city) 
	VALUES ($1, $2) 
	ON CONFLICT (id) DO UPDATE SET city = $2`

	_, err := s.db.Exec(query, id, city)
	return err
}

func (s *Storage) GetUserData(id int64) (string, bool, error) {
	query := `SELECT city FROM users WHERE id = $1`

	var city string
	row := s.db.QueryRow(query, id)
	err := row.Scan(&city)

	if err == sql.ErrNoRows {
		return "", false, nil
	}

	if err != nil {
		return "", false, err
	}

	return city, true, nil
}

func (s *Storage) IncreamentRequestCount(id int64) error {
	query := `UPDATE users 
	SET request_count = request_count + 1
	WHERE id = $1`

	_, err := s.db.Exec(query, id)
	if err != nil {
		return err
	}
	return nil
}

func (s *Storage) GetRequestCount(id int64) (int64, error) {
	query := `SELECT request_count, last_reset_date FROM users WHERE id = $1`

	var count int64
	var resetDate time.Time

	row := s.db.QueryRow(query, id)
	err := row.Scan(&count, &resetDate)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	if resetDate.Format("2006-01-02") != time.Now().Format("2006-01-02") {
		return 0, nil
	}

	return count, nil
}

func (s *Storage) ResetRequestCount(id int64) error {
	query := `UPDATE users
		SET request_count = 0, last_reset_date = CURRENT_DATE
		WHERE id = $1`

	_, err := s.db.Exec(query, id)
	if err != nil {
		return err
	}
	return nil
}

func (s *Storage) SaveNotification(userID int64, city string, hour int) error {
	query := `
	INSERT INTO notifications (user_id, city, notify_hour, enabled)
	VALUES ($1, $2, $3, true)
	ON CONFLICT (user_id) DO UPDATE SET city = $2, notify_hour = $3, enabled = true`

	_, err := s.db.Exec(query, userID, city, hour)
	return err
}

func (s *Storage) RemoveNotification(userID int64) error {
	query := `UPDATE notifications SET enabled = false WHERE user_id = $1`
	_, err := s.db.Exec(query, userID)
	return err
}
