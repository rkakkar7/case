package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"case/common"
	"case/common/constants"
	"case/common/packets"
	"case/common/usermodel"

	nats "github.com/nats-io/nats.go"
	stan "github.com/nats-io/stan.go"
	uuid "github.com/satori/go.uuid"

	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const (
	ClusterID = "test-cluster"
)

var ClientID = "consumer1"

var natsCon stan.Conn
var userStore usermodel.UserStore

type SaveStateDefinition struct {
	UserID string `json:"UID"`
	Error  int    `json:"error,omitempty"`
	packets.SaveGameState
}

func initNats() {
	var err error
	conn1, err := stan.Connect(ClusterID, ClientID)
	for err != nil {
		log.Errorf("error: connecting to nats server: %v", err)
		time.Sleep(time.Duration(5) * time.Second)
		conn1, err = stan.Connect(ClusterID, ClientID)
	}
	natsCon = conn1
	conn := natsCon.NatsConn()
	conn.SetClosedHandler(func(connection *nats.Conn) {
		log.Errorf("nats: connection closed")
		go initNats()
	})

	conn.SetErrorHandler(func(connection *nats.Conn, sub *nats.Subscription, err error) {
		log.Errorf("nats: error: %v %v", sub, err)
	})

	conn.SetReconnectHandler(func(connection *nats.Conn) {
		log.Errorf("nats: reconnected %v", connection.Statistics)
	})

	conn.SetDisconnectHandler(func(connection *nats.Conn) {
		log.Errorf("nats: disconnected")
	})
	start()
}

func start() {
	startSaveGameState()
}

func Start() {
	initMongo()
	initNats()
}

var mongoConn = new(int32)
var MAX_MONGO_LIMIT int32 = 100

func initMongo() {
	ctx := context.TODO()
	MongoSession, err := mongo.Connect(ctx, options.Client().ApplyURI(constants.MongoDBURI).SetMaxPoolSize(100))
	if err != nil {
		log.Fatal(err)
	}
	err = MongoSession.Ping(ctx, readpref.Primary())
	for err != nil {
		log.Errorf("Error while starting mongo session %+v", err)
		time.Sleep(time.Duration(5) * time.Second)
		err = MongoSession.Ping(ctx, readpref.Primary())
	}
	userStore.MongoSession = MongoSession
	userStore.DB = "case"
	userStore.Collections = "users"
}

func startSaveGameState() {
	_, err := natsCon.QueueSubscribe("saveGameState", "saveGameStateQ", func(m *stan.Msg) {
		go common.WithRecover(func() {
			var state SaveStateDefinition
			err := json.Unmarshal(m.Data, &state)
			log.Infof("state %+v", state)
			if err != nil {
				log.Errorf("startSaveGameState error: unmarshal %v", err)
				m.Ack()
				return
			}
			if len(state.UserID) == 0 {
				log.Errorf("startSaveGameState error: userID not present")
				m.Ack()
				return
			}
			if atomic.LoadInt32(mongoConn) > MAX_MONGO_LIMIT {
				return
			}
			var update = make(map[string]map[string]interface{})
			if update["$set"] == nil {
				update["$set"] = make(map[string]interface{})
			}
			if update["$inc"] == nil {
				update["$inc"] = make(map[string]interface{})
			}
			update["$set"]["gamesPlayed"] = state.GamesPlayed
			update["$inc"]["score"] = state.Score
			var user usermodel.User
			if user, err = updateUser(state.UserID, update); err != nil {
				log.Errorf("error: updating user 1 %s %v %v", state.UserID, err, update)
				if err == mongo.ErrNoDocuments {
					m.Ack()
				}
				return
			}
			if state.Score > user.Highscore {
				delete(update, "$inc")
				update["$set"] = make(map[string]interface{})
				update["$set"]["highScore"] = state.Score
				if user, err = updateUser(state.UserID, update); err != nil {
					log.Errorf("error: updating user 2 %s %v %v", state.UserID, err, update)
					if err == mongo.ErrNoDocuments {
						m.Ack()
					}
					return
				}
			}
			log.Infof("user %+v", user)
			m.Ack()
		}, "consumer.startSaveGameState")
	})

	if err != nil {
		log.Fatalf("error: subscribing to saveGameState queue: %v", err)
	} else {
		log.Info("subscribed to SaveGameState queue")
	}
}

func updateUser(uid string, updateQuery map[string]map[string]interface{}) (usermodel.User, error) {
	var (
		user usermodel.User
		err  error
	)

	if updateQuery == nil {
		return user, fmt.Errorf("updateUser: update is nil")
	}

	if updateQuery["$set"] == nil {
		return user, fmt.Errorf("updateUser: update does not contain set")
	}

	u, err := uuid.FromString(uid)
	if err != nil {
		return user, err
	}

	for i := 1; i < 4; i++ {
		atomic.AddInt32(mongoConn, 1)
		ctx := context.TODO()
		collection := userStore.MongoSession.Database(userStore.DB).Collection(userStore.Collections)
		findOneAndUpdateOpts := options.FindOneAndUpdate().SetReturnDocument(options.After)
		err = collection.FindOneAndUpdate(ctx, bson.M{"_id": u}, updateQuery, findOneAndUpdateOpts).Decode(&user)
		atomic.AddInt32(mongoConn, -1)
		if err != nil && err != mongo.ErrNoDocuments {
			sleepDuration := 20
			log.Warnf("error in try # %d; will retry after %d secs", i, sleepDuration)
			time.Sleep(time.Duration(sleepDuration) * time.Second)
		} else {
			break
		}
	}
	return user, err
}
