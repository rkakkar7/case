package types

import (
	"net"
	"sync"
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
	UID         string `json:"id"`
	GamesPlayed int64  `json:"gamesPlayed"`
	Highscore   int64  `json:"highscore"`
	Name        string `json:"name"`
}

type TcpListener struct {
	Listener net.Listener
	Closed   bool
	Mutex    sync.Mutex
}
