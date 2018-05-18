package event

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/raintank/worldping-api/pkg/log"
)

type Event interface {
	Type() string
	Timestamp() time.Time
	Body() ([]byte, error)
}

type RawEvent struct {
	Type      string          `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	Body      json.RawMessage `json:"payload"`
	Source    string
	Attempts  int `json:"attempts"`
}

type Handlers struct {
	sync.Mutex
	Listeners map[string][]chan<- RawEvent
}

func (h *Handlers) Add(key string, ch chan<- RawEvent) {
	h.Lock()
	if _, ok := h.Listeners[key]; !ok {
		l := make([]chan<- RawEvent, 0)
		h.Listeners[key] = l
	}
	h.Listeners[key] = append(h.Listeners[key], ch)
	h.Unlock()
}

func (h *Handlers) GetListeners(key string) []chan<- RawEvent {
	listeners := make([]chan<- RawEvent, 0)
	h.Lock()
	for rk, l := range h.Listeners {
		if rk == "*" || rk == key {
			listeners = append(listeners, l...)
		}
	}
	h.Unlock()
	return listeners
}

var (
	handlers *Handlers
	pubChan  chan Message
	subChan  chan Message
	enabled  bool
)

func Init(rabbitmqUrl, exchange string) error {
	enabled = true
	handlers = &Handlers{
		Listeners: make(map[string][]chan<- RawEvent),
	}
	pubChan = make(chan Message, 100)
	if rabbitmqUrl == "" || exchange == "" {
		log.Info("using internal event channels")
		go handleMessages(pubChan)
	} else {
		log.Info("using rabbitmq for event channels")
		subChan = make(chan Message, 10)
		go Run(rabbitmqUrl, exchange, pubChan, subChan)
		go handleMessages(subChan)
	}
	return nil
}

func Subscribe(t string, channel chan<- RawEvent) {
	handlers.Add(t, channel)
}

func Publish(e Event, attempts int) error {
	if !enabled {
		return nil
	}
	payload, err := e.Body()
	if err != nil {
		return err
	}
	hostname, _ := os.Hostname()
	raw := &RawEvent{
		Type:      e.Type(),
		Timestamp: e.Timestamp(),
		Source:    hostname,
		Body:      payload,
		Attempts:  attempts + 1,
	}
	body, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	msg := Message{
		RoutingKey: e.Type(),
		Payload:    body,
	}
	pubChan <- msg
	return nil
}

func handleMessages(c chan Message) {
	for m := range c {
		go func(msg Message) {
			e := RawEvent{}
			err := json.Unmarshal(msg.Payload, &e)
			if err != nil {
				log.Error(3, "unable to unmarshal event Message. %s", err)
				return
			}
			log.Debug("processing event of type %s", e.Type)
			//broadcast the event to listeners.
			for _, ch := range handlers.GetListeners(e.Type) {
				ch <- e
			}
		}(m)
	}
}
