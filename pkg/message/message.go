package message

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/gorilla/websocket"
)

type Version uint8

// identifier of message format
const (
	EventV1 Version = iota
)

type Message struct {
	MessageType int
	Body        []byte
}

func (msg *Message) ToEvent() (*Event, error) {
	switch msg.MessageType {
	case websocket.TextMessage:
		return nil, errors.New("Events must use BinaryMessage type")
	case websocket.BinaryMessage:
		if len(msg.Body) < 9 {
			return nil, errors.New("Message Payload too small")
		}
		ver := msg.Body[0]
		if Version(ver) != EventV1 {
			return nil, errors.New("Invalid Message Body")
		}
		eventLength := uint8(msg.Body[1])
		payloadLength := len(msg.Body) - int(eventLength) - 2

		// eventLength must be at least 1 char, and less then the total length of the payload.
		if eventLength < 1 || payloadLength < 0 {
			return nil, errors.New("Invalid Message Body")
		}

		payload := make([]byte, payloadLength)
		if payloadLength > 0 {
			copy(payload, msg.Body[2+eventLength:])
		}

		return &Event{Event: string(msg.Body[2 : eventLength+2]), Payload: payload}, nil
	}
	return nil, errors.New("unknown mesageType")
}

type Event struct {
	Event   string
	Payload []byte
}

func (e *Event) ToMessage() (*Message, error) {
	msg := &Message{MessageType: websocket.BinaryMessage}
	body := new(bytes.Buffer)
	err := binary.Write(body, binary.LittleEndian, uint8(EventV1))
	if err != nil {
		return nil, fmt.Errorf("binary.Write failed: %s", err.Error())
	}
	eventLength := len(e.Event)
	if eventLength > 255 {
		return nil, errors.New("Event can not be more then 255 chars")
	}
	err = binary.Write(body, binary.LittleEndian, uint8(eventLength))
	if err != nil {
		return nil, fmt.Errorf("binary.Write failed: %s", err.Error())
	}

	_, err = body.Write([]byte(e.Event))
	if err != nil {
		return nil, fmt.Errorf("body.Write failed: %s", err.Error())
	}
	_, err = body.Write(e.Payload)
	if err != nil {
		return nil, fmt.Errorf("body.Write failed: %s", err.Error())
	}

	msg.Body = body.Bytes()
	return msg, nil
}
