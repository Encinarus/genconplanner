package pubsub

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/Encinarus/genconplanner/internal/postgres"
	"github.com/lib/pq"
)

type PartyUpdateEvent struct {
	PartyId int64  `json:"party_id"`
	EventId string `json:"event_id"`
	Email   string `json:"email"`
	Tier    string `json:"tier"`
}

type Subscription struct {
	PartyId int64
	C       chan PartyUpdateEvent
	id      int64
}

type Hub struct {
	mu            sync.RWMutex
	subscriptions map[int64]map[int64]*Subscription
	nextId        int64
	listener      *pq.Listener
}

var GlobalHub *Hub
var once sync.Once

func Init() {
	once.Do(func() {
		GlobalHub = &Hub{
			subscriptions: make(map[int64]map[int64]*Subscription),
		}
		connStr := postgres.GetConnStr()
		if connStr == "" {
			log.Println("pubsub: no connection string available for pq.Listener")
			return
		}

		GlobalHub.listener = pq.NewListener(connStr, 10*time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
			if err != nil {
				log.Printf("pubsub listener error: %v\n", err)
			}
		})

		err := GlobalHub.listener.Listen("party_updates")
		if err != nil {
			log.Printf("pubsub failed to listen on party_updates: %v\n", err)
			return
		}

		go GlobalHub.run()
	})
}

func (h *Hub) run() {
	for n := range h.listener.Notify {
		if n == nil {
			continue
		}
		var ev PartyUpdateEvent
		if err := json.Unmarshal([]byte(n.Extra), &ev); err != nil {
			log.Printf("pubsub unmarshal error: %v\n", err)
			continue
		}

		h.mu.RLock()
		subs, ok := h.subscriptions[ev.PartyId]
		if ok {
			for _, sub := range subs {
				select {
				case sub.C <- ev:
				default:
					// channel buffer full, skip to avoid blocking hub
				}
			}
		}
		h.mu.RUnlock()
	}
}

func Subscribe(partyId int64) *Subscription {
	if GlobalHub == nil {
		Init()
	}
	GlobalHub.mu.Lock()
	defer GlobalHub.mu.Unlock()

	GlobalHub.nextId++
	sub := &Subscription{
		PartyId: partyId,
		C:       make(chan PartyUpdateEvent, 32),
		id:      GlobalHub.nextId,
	}

	if GlobalHub.subscriptions[partyId] == nil {
		GlobalHub.subscriptions[partyId] = make(map[int64]*Subscription)
	}
	GlobalHub.subscriptions[partyId][sub.id] = sub
	return sub
}

func (s *Subscription) Unsubscribe() {
	if GlobalHub == nil {
		return
	}
	GlobalHub.mu.Lock()
	defer GlobalHub.mu.Unlock()

	subs, ok := GlobalHub.subscriptions[s.PartyId]
	if ok {
		delete(subs, s.id)
		if len(subs) == 0 {
			delete(GlobalHub.subscriptions, s.PartyId)
		}
	}
	close(s.C)
}

// PublishTestEvent allows unit tests to push events directly to active subscriptions
func PublishTestEvent(partyId int64, ev PartyUpdateEvent) {
	if GlobalHub == nil {
		Init()
	}
	GlobalHub.mu.RLock()
	defer GlobalHub.mu.RUnlock()

	subs, ok := GlobalHub.subscriptions[partyId]
	if ok {
		for _, sub := range subs {
			select {
			case sub.C <- ev:
			default:
			}
		}
	}
}
