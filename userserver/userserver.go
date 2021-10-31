package userserver

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/segmentio/encoding/json"

	"case/common"
	"case/common/constants"

	"case/common/types"
	"case/common/usermodel"

	nats "github.com/nats-io/nats.go"
	stan "github.com/nats-io/stan.go"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var userStore usermodel.UserStore
var natsCon stan.Conn
var mongoConn = new(int32)

var (
	ClusterID   = "test-cluster"
	ClientID    = "client-us1"
	clientList  []*Client
	tcpListener types.TcpListener
)

type Client struct {
	conn         net.Conn
	buf          *bufio.Reader
	disconnected bool
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
	log.Infof("nats: connected")
}

const (
	MSG_DELIM = 96 // msgs sent and received are separated by this char
)

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
	log.Info("mongo init")
}

func gracefulExit() {
	defer common.Recover("gracefulExit")

	exitSignal := make(chan os.Signal)
	signal.Notify(exitSignal, syscall.SIGTERM)
	signal.Notify(exitSignal, os.Interrupt)
	sig := <-exitSignal
	if sig == syscall.SIGTERM || sig == os.Interrupt {
		log.Info("received signal to exit")

		// for _, client := range clientList {
		// 	msg := types.Message{
		// 		RouterHeader: constants.RouterHeader.Disconnect,
		// 	}
		// 	client.disconnected = true
		// 	client.SendMessage(msg)
		// }
		// tcpListener.Closed = true
		// tcpListener.Listener.Close()
		// tcpListener.Mutex.Unlock()
		os.Exit(0)
	}
}

//ListenAndServe Start the server on the given address
func ListenAndServe(port int) {
	go gracefulExit()

	initMongo()
	initNats()

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	tcpListener = types.TcpListener{
		Listener: ln,
		Closed:   false,
	}
	if err != nil {
		log.Fatalf("error: starting server: %v", err)
	}

	// privateIP := "localhost"
	for {
		conn, err := ln.Accept()
		log.Infof("Got a new connection request")
		tcpListener.Mutex.Lock()
		if err != nil {
			if tcpListener.Closed {
				log.Warnf("stopping server: tcpListener closed: %v", err)
				return
			}
			log.Errorf("error: new connection: %v", err)
			tcpListener.Mutex.Unlock()
			continue
		}
		remoteAddr, ok := conn.RemoteAddr().(*net.TCPAddr)
		if !ok {
			log.Infof("ln.Accept: rejected %s", remoteAddr.IP.String())
		}
		log.Infof("ln.Accept: connected %s", remoteAddr.IP.String())
		router := &Client{
			conn: conn,
			buf:  bufio.NewReader(conn),
		}
		clientList = append(clientList, router)
		tcpListener.Mutex.Unlock()
		go router.Listen()
	}
}

func PublishMsg(subj string, msg interface{}) {
	payloadbytes, err := json.Marshal(msg)
	if err != nil {
		log.Errorf("error: publishMsg marshal %v", msg)
		return
	}

	_, err = natsCon.PublishAsync(subj, payloadbytes, nil)
	if err != nil {
		log.Errorf("error: publish %s %v", subj, err)
	}
}
