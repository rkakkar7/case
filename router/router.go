package router

import (
	"case/common/constants"
	"case/common/packets"
	"case/common/types"
	"case/common/usermodel"
	"case/consumer"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"case/microservice"

	"github.com/gorilla/mux"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var userStore usermodel.UserStore
var MAX_CLIENT int = 20
var natsCon stan.Conn
var natsAdress string = "nats://127.0.0.1:4222"

const ClusterID = "test-cluster"
const ClientID = "router1"

type Router struct {
	srv               *http.Server
	serviceController *microservice.Controller
}

func init() {
	initMongo()
	initNats()
}

func (router *Router) handleUserGetAll(w http.ResponseWriter, r *http.Request) {
	log.Infof("handleUserGetAll start")
	defer log.Infof("handleUserGetAll end")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
}

func (router *Router) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	log.Infof("handleCreateUser start")
	defer log.Infof("handleCreateUser end")
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Errorf("handleCreateUser: json Decoder err %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	msg := types.Message{
		RouterHeader: constants.RouterHeader.User,
		PayloadURI:   constants.UserPayloadURI.UserCreate,
		Payload:      body,
	}
	ch, err := router.serviceController.SendMessage(msg)
	if err != nil {
		log.Errorf("handleCreateUser: router.serviceController.SendMessage err %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	resp := <-ch
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(resp.Payload)
}

func (router *Router) handleSaveGameState(w http.ResponseWriter, r *http.Request) {
	log.Infof("handleSaveGameState start")
	defer log.Infof("handleSaveGameState end")
	params := mux.Vars(r)
	userID := params["userID"]
	var req packets.SaveGameState
	json.NewDecoder(r.Body).Decode(&req)
	msg := consumer.SaveStateDefinition{
		UserID:        userID,
		SaveGameState: req,
	}
	natsMsgBytes, _ := json.Marshal(msg)
	_, err := natsCon.PublishAsync("saveGameState", natsMsgBytes, nil)
	if err != nil {
		log.Errorf("msgHandler: natsCon.Publish %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
}

func (router *Router) handleLoadGameState(w http.ResponseWriter, r *http.Request) {
	log.Infof("handleLoadGameState start")
	defer log.Infof("handleLoadGameState end")
	params := mux.Vars(r)
	userID := params["userID"]
	msg := types.Message{
		RouterHeader: constants.RouterHeader.User,
		PayloadURI:   constants.UserPayloadURI.LoadGameState,
		UserID:       userID,
	}
	ch, err := router.serviceController.SendMessage(msg)
	if err != nil {
		log.Errorf("handleLoadGameState: router.serviceController.SendMessage err %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	resp := <-ch
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(resp.Payload)
}

func (router *Router) handleUpdateFriends(w http.ResponseWriter, r *http.Request) {
	log.Infof("handleUpdateFriends start")
	defer log.Infof("handleUpdateFriends end")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
}

func (router *Router) handleGetFriends(w http.ResponseWriter, r *http.Request) {
	log.Infof("handleGetFriends start")
	defer log.Infof("handleGetFriends end")
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
}

func (router *Router) HandleRouting(muxRouter *mux.Router) {
	muxRouter.HandleFunc("/user", limitClients(router.handleUserGetAll, MAX_CLIENT)).Methods("GET")
	muxRouter.HandleFunc("/user", limitClients(router.handleCreateUser, MAX_CLIENT)).Methods("POST")
	muxRouter.HandleFunc("/user/{userID}/state", router.handleSaveGameState).Methods("PUT")
	muxRouter.HandleFunc("/user/{userID}/state", router.handleLoadGameState).Methods("GET")
	muxRouter.HandleFunc("/user/{userID}/friends", router.handleUpdateFriends).Methods("PUT")
	muxRouter.HandleFunc("/user/{userID}/friends", router.handleGetFriends).Methods("GET")
}

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
	log.Infof("mongo connected")
}

func initNats() {
	conn1, err := stan.Connect(ClusterID, ClientID)
	for err != nil {
		log.Errorf("error: connecting to nats server: %v", err)
		time.Sleep(time.Duration(5) * time.Second)
		conn1, err = stan.Connect(ClusterID, ClientID)
	}
	conn := conn1.NatsConn()
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
	natsCon = conn1
}

func NewRouter(muxRouter *mux.Router, serverAddress string) *Router {
	srv := &http.Server{
		Addr:              serverAddress,
		ReadHeaderTimeout: 15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       120 * time.Second,
		Handler:           muxRouter,
	}
	router := &Router{
		srv:               srv,
		serviceController: microservice.NewController(),
	}
	router.serviceController.AddService(types.ServiceDefinition{
		Host:    "localhost",
		Port:    constants.UserserverPort,
		Service: constants.UserService,
		ID:      "localhost_8081",
	}, serverAddress, "router")
	return router
}

func (router *Router) Serve() error {
	return router.srv.ListenAndServe()
}

func (router *Router) RemoveAllService() {
	router.serviceController.RemoveAllService()
}

func limitClients(next http.HandlerFunc, maxClients int) http.HandlerFunc {
	sema := make(chan int, maxClients)
	return func(w http.ResponseWriter, req *http.Request) {
		sema <- 1
		defer func() { <-sema }()
		next(w, req)
	}
}
