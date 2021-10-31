package usermodel

import (
	"case/common/packets"
	"context"
	"math/rand"
	"strconv"
	"time"

	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserStore struct {
	MongoSession *mongo.Client
	DB           string
	Collections  string
}

func (store UserStore) CreateUser(packet packets.CreateUser) (*User, error) {
	uniqueID := uuid.NewV4()
	log.Infof("UUIDv4 created: %s", uniqueID)

	if packet.Name == "" {
		packet.Name = "Guest" + strconv.Itoa(rand.Intn(10)) + strconv.Itoa(rand.Intn(10)) + strconv.Itoa(rand.Intn(10)) + strconv.Itoa(rand.Intn(10))
	}

	user := User{
		CreatedAt:   time.Now(),
		UID:         uniqueID,
		Name:        packet.Name,
		Score:       0,
		GamesPlayed: 0,
		Highscore:   0,
	}

	ctx := context.TODO()
	collection := store.MongoSession.Database(store.DB).Collection(store.Collections)
	_, err := collection.InsertOne(ctx, &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (store UserStore) LoadUserUUIDString(uniqueID string) (*User, error) {
	user := User{}
	log.Debugf("Fetching user %s", uniqueID)
	ctx := context.TODO()
	collection := store.MongoSession.Database(store.DB).Collection(store.Collections)
	u, err := uuid.FromString(uniqueID)
	if err != nil {
		return nil, err
	}
	err = collection.FindOne(ctx, bson.M{"_id": u}).Decode(&user)
	return &user, err
}

func (store UserStore) LoadUserUUID(uniqueIDBytes uuid.UUID) (*User, error) {
	user := User{}
	log.Debugf("Fetching user uuid %s", uniqueIDBytes.String())
	ctx := context.TODO()
	collection := store.MongoSession.Database(store.DB).Collection(store.Collections)
	err := collection.FindOne(ctx, bson.M{"_id": uniqueIDBytes}).Decode(&user)
	return &user, err
}
