package notifications

import (
	"fmt"
	"log"
	"time"
)

func New(sendMessage func(int64, string) error, getWeather func(string) (string, error)) *Manager {
	return &Manager{
		subscriptions: make(map[int64]*Subscription),
		subscribeCh:   make(chan *Subscription, 100),
		unsubscribeCh: make(chan int64, 100),
		stopCh:        make(chan struct{}),
		sendMessage:   sendMessage,
		getWeather:    getWeather,
	}
}

func (m *Manager) Subscribe(userID int64, city string, hour int) {
	sub := &Subscription{
		UserID:   userID,
		City:     city,
		NotifyAt: time.Duration(hour) * time.Hour,
		Timezone: time.FixedZone("MSK", 3*60*60),
	}

	m.subscribeCh <- sub
}

func (m *Manager) Unsubscribe(userID int64) {
	m.unsubscribeCh <- userID
}

func (m *Manager) Start() {
	go m.run()
}

func (m *Manager) Stop() {
	close(m.stopCh)
}

func (m *Manager) run() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	log.Println("Notification Manager запущен")

	for {
		select {
		case sub := <-m.subscribeCh:
			m.mu.Lock()
			m.subscriptions[sub.UserID] = sub
			m.mu.Unlock()
			log.Printf("Пользователь %d подписался на уведомления (город: %s)", sub.UserID, sub.City)

		case userID := <-m.unsubscribeCh:
			m.mu.Lock()
			delete(m.subscriptions, userID)
			m.mu.Unlock()
			log.Printf("Пользователь %d отписался от уведомлений", userID)

		case <-ticker.C:
			m.checkAndSendNotifications()

		case <-m.stopCh:
			log.Println("Notification Manager остановлен")
			return
		}
	}
}

func (m *Manager) checkAndSendNotifications() {
	now := time.Now().In(time.FixedZone("MSK", 3*60*60))
	currentHour := now.Hour()
	currentMinute := now.Minute()

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, sub := range m.subscriptions {
		targetHour := int(sub.NotifyAt / time.Hour)

		// Отправляем только если текущее время совпадает с временем подписки
		if currentHour == targetHour && currentMinute == 0 {
			go m.sendWeatherNotification(sub) // Запускаем в отдельной горутине!
		}
	}
}

func (m *Manager) sendWeatherNotification(sub *Subscription) {
	weather, err := m.getWeather(sub.City)
	if err != nil {
		log.Printf("Ошибка получения погоды для %s: %v", sub.City, err)
		return
	}

	message := fmt.Sprintf("🌤️ Ежедневное уведомление о погоде!\n\n%s", weather)

	if err := m.sendMessage(sub.UserID, message); err != nil {
		log.Printf("Ошибка отправки уведомления пользователю %d: %v", sub.UserID, err)
	}
}
