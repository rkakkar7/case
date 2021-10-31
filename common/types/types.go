package types

import (
	"net"
	"sync"

	uuid "github.com/satori/go.uuid"
)

type RouterHeader string

type PayloadURI string

type Message struct {
	RouterHeader RouterHeader `json:"RH"`
	PayloadURI   PayloadURI   `json:"PU"`
	UserID       string       `json:"ID,omitempty"`
	Payload      []byte       `json:"PY"`
	ChannelKey   string       `json:"CK"`
	Error        int          `json:"ER,omitempty"`
}

type ChannelMsg struct {
	Msg   Message
	Error error
}

type ServiceDefinition struct {
	ID      string
	Service string
	Port    int
	Host    string
}

type GameState struct {
	UID         uuid.UUID `json:"id"`
	GamesPlayed int64     `json:"gamesPlayed,omitempty"`
	Highscore   int64     `json:"highscore,omitempty"`
}

type TcpListener struct {
	Listener net.Listener
	Closed   bool
	Mutex    sync.Mutex
}
