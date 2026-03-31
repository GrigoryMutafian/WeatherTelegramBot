package notifications

import (
	"sync"
	"time"
)

type Subscription struct {
	UserID   int64
	City     string
	NotifyAt time.Duration
	Timezone *time.Location
}

type Manager struct {
	subscriptions map[int64]*Subscription
	mu            sync.RWMutex

	subscribeCh   chan *Subscription
	unsubscribeCh chan int64
	stopCh        chan struct{}

	sendMessage func(userID int64, message string) error
	getWeather  func(city string) (string, error)
}
