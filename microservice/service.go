package microservice

import (
	"bufio"
	"fmt"
	"net"
	"time"

	"github.com/segmentio/encoding/json"
	"golang.org/x/sync/syncmap"

	"case/common"
	"case/common/constants"
	"case/common/types"

	log "github.com/sirupsen/logrus"
)

const (
	MsgDelim = 96 // msgs sent and received are separated by this char
)

type service struct {
	ID           string
	Name         string
	Caller       string
	Host         string
	Port         int
	RouterID     string
	lastMsgWrite time.Time
	lastMsgRead  time.Time
	logger       *log.Entry
	chanMap      syncmap.Map // user request to service map, e.g. key - userID+req+timestamp, value - chan
	Send         func(types.Message)
	destroyed    bool
	restarted    bool
	conn         net.Conn
	Connecting   bool
}

func NewService(name, caller, host string, port int, routerID string) *service {
	s := &service{
		ID:       fmt.Sprintf("%s_%d", host, port),
		Name:     name,
		Caller:   caller,
		Host:     host,
		Port:     port,
		RouterID: routerID + name,
	}

	s.logger = log.WithFields(log.Fields{
		"service": name,
		"host":    host,
		"port":    port,
		"package": "service",
		"router":  routerID,
	})
	return s
}

func (s *service) Start() {
	go common.WithRecover(s.connect, "start")
}

func (s *service) Stop() {
	s.logger.Infof("stop %s %s:%d", s.Name, s.Host, s.Port)
	if s.conn != nil {
		msg := types.Message{
			RouterHeader: constants.RouterHeader.Disconnect,
			UserID:       s.RouterID,
			PayloadURI:   types.PayloadURI(s.Caller),
		}
		s.Send(msg)
		s.destroyed = true
	}
}

// dials on the service address until it connects
func (s *service) connect() {
	if s.destroyed {
		s.logger.Warnf("connect already destroyed")
		return
	}

	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)

	s.logger.Infof("start %s %s", s.Name, addr)

	s.Connecting = true
	conn, err := net.Dial("tcp", addr)
	for err != nil {
		if s.destroyed {
			s.logger.Warnf("dial already destroyed")
			return
		}
		if s.restarted {
			s.logger.WithError(err).Warnf("dial %v", err)
		} else {
			s.logger.WithError(err).Errorf("dial %v", err)
		}
		time.Sleep(time.Duration(1) * time.Second)
		conn, err = net.Dial("tcp", addr)
	}
	s.conn = conn
	s.restarted = false
	s.Connecting = false
	s.logger.Infof("connected %s %s", s.Name, addr)

	quit := make(chan struct{})
	s.Send = s.makeSendFun(conn)
	msg := types.Message{
		RouterHeader: constants.RouterHeader.Connect,
		UserID:       s.RouterID,
		PayloadURI:   types.PayloadURI(s.Caller),
	}
	s.Send(msg)
	go common.WithRecover(func() { s.readPump(conn, quit) }, "read-pump")
	s.startHB(conn, quit)
}

// make send function to write to the service, outgoing data
func (s *service) makeSendFun(conn net.Conn) func(types.Message) {
	return func(msg types.Message) {
		if s.Connecting && (msg.RouterHeader != constants.RouterHeader.Connect) {
			s.logger.Warn("send: not connected")
			return
		}

		if s.destroyed {
			s.logger.Warn("send: destroyed")
			return
		}

		if s.restarted {
			s.logger.Warn("send: restarting")
			return
		}

		msgBytes, err := json.Marshal(msg)
		if err != nil {
			s.logger.WithError(err).Errorf("marshal")
			return
		}
		s.logger.Infof("message sent")
		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		_, err = conn.Write(append(msgBytes, MsgDelim))
		conn.SetWriteDeadline(time.Time{})
		s.lastMsgWrite = time.Now()
		if err != nil {
			if s.destroyed || s.restarted {
				s.logger.WithError(err).Warnf("send")
			} else {
				s.logger.WithError(err).Errorf("send")
				conn.Close()
			}
		}
	}
}

// incoming data from services
func (s *service) readPump(conn net.Conn, quit chan struct{}) {
	var buf = bufio.NewReader(conn)
	for {
		conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		msgBytes, err := buf.ReadBytes(MsgDelim)
		conn.SetReadDeadline(time.Time{})
		if err != nil {
			if s.destroyed || s.restarted {
				s.logger.WithError(err).Warnf("read")
			} else {
				s.logger.WithError(err).Errorf("read")
			}
			close(quit)
			s.connect()
			return
		}
		s.lastMsgRead = time.Now()
		msg := types.Message{}
		err = json.Unmarshal(msgBytes[:len(msgBytes)-1], &msg)
		if err != nil {
			s.logger.WithError(err).Errorf("unmarshall %s", string(msgBytes[:len(msgBytes)-1]))
			continue
		}

		if msg.RouterHeader == constants.RouterHeader.Heartbeat {
			continue
		} else if msg.RouterHeader == constants.RouterHeader.Disconnect {
			s.logger.Info("got disconnect request")
			s.restarted = true
			close(quit)
			s.connect()
			return
		} else {
			s.msgHandler(msg, s.ID, s.Name)
		}
	}
}

func (s *service) startHB(conn net.Conn, quit chan struct{}) {
	hb := time.NewTicker(time.Duration(5) * time.Second)
	defer hb.Stop()

	hbMsg := types.Message{
		RouterHeader: constants.RouterHeader.Heartbeat,
	}

	msgByte, _ := json.Marshal(hbMsg)
	s.lastMsgRead = time.Now()

	for {
		select {
		case <-quit:
			s.logger.Info("quit heartbeat")
			return
		case <-hb.C:
			if s.destroyed || s.restarted {
				s.logger.Warn("quit heartbeat destroyed")
				return
			}

			if time.Since(s.lastMsgRead) >= time.Duration(30)*time.Second {
				s.logger.Errorf("heartbeat time out")
				conn.Close()
				return
			}

			conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
			_, err := conn.Write(append(msgByte, MsgDelim))
			conn.SetWriteDeadline(time.Time{})
			s.lastMsgWrite = time.Now()
			if err != nil {
				if s.destroyed || s.restarted {
					s.logger.WithError(err).Warnf("send heartbeat")
				} else {
					s.logger.WithError(err).Errorf("send heartbeat")
					conn.Close()
				}
				return
			}
		}
	}
}

func (s *service) msgHandler(msg types.Message, serviceID, serviceName string) {
	switch msg.RouterHeader {
	case constants.RouterHeader.HeartbeatAck:
		return
	case constants.RouterHeader.User:
		s.logger.Info("msgHandler: msg received from user service")
		if len(msg.ChannelKey) == 0 {
			s.logger.Errorf("msgHandler: empty channel key %+v", msg)
			return
		}
		chInterface, ok := s.chanMap.LoadAndDelete(msg.ChannelKey)
		if !ok {
			s.logger.Errorf("msgHandler: channel key not present in chanmap")
			return
		}
		ch := chInterface.(chan types.Message)
		ch <- msg
		log.Infof("msgHandler: msg sent")
		close(ch)
		s.chanMap.Delete(msg.ChannelKey)
	}
}
