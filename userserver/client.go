package userserver

import (
	"case/common"
	"case/common/constants"
	"case/common/packets"
	"case/common/types"
	"case/db/redis"
	"sync/atomic"
	"time"

	"github.com/segmentio/encoding/json"
	log "github.com/sirupsen/logrus"
)

func (client *Client) Listen() {
	log.Infof("router.listen")
	hbMsg := types.Message{
		RouterHeader: constants.RouterHeader.Heartbeat,
	}

	hbMsgByte, _ := json.Marshal(hbMsg)

	for {
		client.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		msgBytes, err := client.buf.ReadBytes(MSG_DELIM)
		client.conn.SetReadDeadline(time.Time{})
		if err != nil {
			if client.disconnected {
				log.Warnf("error: read: %v", err)
			} else {
				log.Errorf("error: read: %v", err)
			}
			client.conn.Close()
			return
		}

		msg := types.Message{}
		err = json.Unmarshal(msgBytes[:len(msgBytes)-1], &msg)
		if err != nil {
			log.Warnf("error: unmarshall: %v %s", err, string(msgBytes))
			continue
		}

		if msg.RouterHeader == constants.RouterHeader.Heartbeat {
			client.conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
			_, err = client.conn.Write(append(hbMsgByte, MSG_DELIM))
			client.conn.SetWriteDeadline(time.Time{})
			if err != nil {
				if client.disconnected {
					log.Warnf("send-reply: error: write: %v", err)
				} else {
					log.Errorf("send-reply: error: write: %v", err)
				}
				client.conn.Close()
				return
			}
			continue
		} else if msg.RouterHeader == constants.RouterHeader.Disconnect {
			client.disconnected = true
			log.Infof("disconnect")
			continue
		}

		if atomic.LoadInt32(mongoConn) < 600 {
			go common.WithRecover(func() {
				client.processMessage(msg)
			}, "listenRead processMessage")
		}

		if client.disconnected {
			log.Warn("Listen: disconnected")
			continue
		}
	}
}

func (client *Client) processMessage(msg types.Message) {
	log.Infof("processMessage: %+v", msg)
	switch msg.PayloadURI {
	case constants.UserPayloadURI.LoadGameState:
		log.Infof("Got user load game state request")
		client.loadGameState(msg.UserID, msg.ChannelKey)
	case constants.UserPayloadURI.UserCreate:
		log.Infof("Create user request")
		var createUser packets.CreateUser
		err := json.Unmarshal(msg.Payload, &createUser)
		if err != nil {
			log.Errorf("UserPayloadURI.UserCreate json.Unmarshal %v", err)
			client.SendReply("", msg.ChannelKey, packets.CreateUserAck{
				Error: 1,
			})
			return
		}
		client.createUser(msg.ChannelKey, createUser)
	case constants.UserPayloadURI.GetFriends:
		log.Infof("Get friends request")
		client.getFriends(msg.UserID, msg.ChannelKey)
	}
}

func (client *Client) createUser(channelKey string, createUser packets.CreateUser) {
	log.Infof("client.createUser %s")
	user, err := userStore.CreateUser(createUser)
	if err != nil {
		log.Errorf("client.createUser: userStore.LoadUserUUIDString %v", err)
		client.SendReply("", channelKey, packets.CreateUserAck{
			Error: 1,
		})
		return
	}
	log.Infof("user %+v", user)
	client.SendReply("", channelKey, packets.CreateUserAck{
		Name:   user.Name,
		UserID: user.UID.String(),
	})
}

func (client *Client) loadGameState(userID, channelKey string) {
	log.Infof("client.loadGameState %s", userID)
	gameState, err := redis.GetReceipt(userID)
	if err == nil {
		client.SendReply(userID, channelKey, packets.LoadGameState{
			Score:       gameState.Highscore,
			GamesPlayed: gameState.GamesPlayed,
		})
		return
	}
	user, err := userStore.LoadUserUUIDString(userID)
	if err != nil {
		log.Errorf("client.loadGameState userStore.LoadUserUUIDString %s %v", userID, err)
		client.SendReply(userID, channelKey, packets.LoadGameState{
			Error: 1,
		})
		return
	}

	client.SendReply(userID, channelKey, packets.LoadGameState{
		Score:       user.Highscore,
		GamesPlayed: user.GamesPlayed,
	})
}

func (client *Client) getFriends(userID, channelKey string) {
	log.Infof("client.getFriends %s", userID)
	user, err := userStore.LoadUserUUIDString(userID)
	if err != nil {
		log.Errorf("client.getFriends:userStore.LoadUserUUIDString %s %v", userID, err)
		client.SendReply(userID, channelKey, packets.GetFriends{
			Error: 1,
		})
		return
	}

	var friends []packets.FriendData
	if len(user.Friends) == 0 {
		client.SendReply(userID, channelKey, packets.GetFriends{
			Friends: friends,
		})
	}

	gameStates := redis.GetGameStates(user.Friends)
	for _, friendID := range user.Friends {
		if val, ok := gameStates[friendID]; ok {
			friend := packets.FriendData{}
			err := json.Unmarshal([]byte(val), &friend)
			if err != nil {
				log.Errorf("getFriends json.Unmarshal %s %v", userID, err)
				continue
			}
			friends = append(friends, friend)
		} else {
			friend, err := userStore.LoadUserUUIDString(friendID)
			if err != nil {
				log.Errorf("getFriends userStore.LoadUserUUIDString: %s %v", userID, err)
				continue
			}
			friends = append(friends, packets.FriendData{
				Score:  friend.Score,
				Name:   friend.Name,
				UserID: friendID,
			})
		}
	}

	client.SendReply(userID, channelKey, packets.GetFriends{
		Friends: friends,
	})
}

func (client *Client) SendReply(userID, channelKey string, payload interface{}) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Errorf("send-reply: error: marshal: %v", err)
		return
	}

	msg := types.Message{
		ChannelKey:   channelKey,
		UserID:       userID,
		RouterHeader: constants.RouterHeader.User,
		Payload:      payloadBytes,
	}
	log.Debugf("send-reply: payload %v", payload)
	client.SendMessage(msg)
}

func (client *Client) SendMessage(msg types.Message) {
	log.Debugf("send-msg: msg %v", msg)
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Errorf("send-reply: error: marshall: %v", err)
		return
	}

	client.conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
	_, err = client.conn.Write(append(msgBytes, MSG_DELIM))
	client.conn.SetWriteDeadline(time.Time{})
	if err != nil {
		if client.disconnected {
			log.Warnf("send-reply: error: write: %v", err)
		} else {
			log.Errorf("send-reply: error: write: %v", err)
		}
		client.conn.Close()
	}
}
