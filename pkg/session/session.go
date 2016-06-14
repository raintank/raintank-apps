package session

import (
	"fmt"
	"sync"
	"time"

	"github.com/codeskyblue/go-uuid"
	"github.com/gorilla/websocket"
	"github.com/raintank/raintank-apps/pkg/message"
	"github.com/raintank/worldping-api/pkg/log"
)

type Handler interface {
	HandleMessage(message *message.Event)
}

type Session struct {
	sync.Mutex
	Id               string
	EventHandlers    map[string]*message.Handler
	Conn             *websocket.Conn
	writeMessageChan chan *message.Message
	closing          bool
	rDone            chan struct{}
	wDone            chan struct{}
}

func NewSession(conn *websocket.Conn, writeQueueSize int) *Session {
	s := &Session{
		Id:               uuid.NewUUID().String(),
		EventHandlers:    make(map[string]*message.Handler),
		Conn:             conn,
		writeMessageChan: make(chan *message.Message, writeQueueSize),
	}
	return s
}

func (s *Session) On(event string, f interface{}) error {
	s.Lock()
	defer s.Unlock()
	if _, ok := s.EventHandlers[event]; ok {
		return fmt.Errorf("Handler for event %s already defined", event)
	}
	h, err := message.NewHandler(f)
	if err != nil {
		return err
	}
	s.EventHandlers[event] = h
	return nil
}

func (s *Session) Emit(event *message.Event) error {
	msg, err := event.ToMessage()
	if err != nil {
		return err
	}
	s.writeMessageChan <- msg
	return nil
}

func (s *Session) Start() {
	s.rDone = make(chan struct{})
	s.wDone = make(chan struct{})
	go s.socketReader(s.rDone)
	go s.socketWriter(s.wDone)

	select {
	case <-s.wDone:
		log.Debug("writer closed.")
		s.disconnected()
		return
	case <-s.rDone:
		log.Debug("reader closed.")
		s.disconnected()
		return
	}
}

func (s *Session) Close() {
	s.Lock()
	s.closing = true
	s.Unlock()
	s.writeMessageChan <- &message.Message{MessageType: websocket.CloseMessage, Body: websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")}
	close(s.writeMessageChan)
	log.Info("waiting for socketWriter to finish sending all messages.")
	select {
	case <-s.wDone:
	case <-time.After(time.Second * 2):
		log.Warn("socketWriter taking too long. Closing connectio now. %d messages in queue will be lost.", len(s.writeMessageChan)+1)
	}
	s.Conn.Close()
}

func (s *Session) disconnected() {
	//dont emit a disconnect event if Close() was called.
	closing := false
	s.Lock()
	if !s.closing {
		closing = true
	}
	s.Unlock()
	if closing {
		if h, ok := s.EventHandlers["disconnect"]; ok {
			h.Call([]byte{})
		}
	}
}

func (s *Session) socketReader(done chan struct{}) {
	defer s.Conn.Close()
	defer close(done)
	for {
		mtype, body, err := s.Conn.ReadMessage()
		if err != nil {
			log.Error(3, "read: %s", err)
			return
		}
		msg := &message.Message{MessageType: mtype, Body: body}
		e, err := msg.ToEvent()
		if err != nil {
			log.Error(3, "Error: failed to decode message to Event.")
		}
		s.Lock()
		h, ok := s.EventHandlers[e.Event]
		s.Unlock()
		if ok {
			h.Call(e.Payload)
		} else {
			log.Warn("no handler for event: %s", e.Event)
		}
	}
}

func (s *Session) socketWriter(done chan struct{}) {
	defer s.Conn.Close()
	defer close(done)

	for msg := range s.writeMessageChan {
		log.Debug("socket %s sending message", s.Id)
		err := s.Conn.WriteMessage(msg.MessageType, msg.Body)
		retryDelay := time.Millisecond * 25
		for err != nil {
			s.Lock()
			closing := s.closing
			s.Unlock()
			if closing {
				return
			}
			log.Error(3, "unable to write to websocket: %s", err)
			if retryDelay < time.Second {
				retryDelay = retryDelay * 2
			}
			time.Sleep(retryDelay)
			err = s.Conn.WriteMessage(msg.MessageType, msg.Body)
		}
	}
	log.Debug("writeMessageChan closed.")
}
