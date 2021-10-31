package constants

import "case/common/types"

var RouterPort int = 8080
var UserserverPort int = 8081

var UserService string = "userservice"

var RouterHeader = struct {
	Heartbeat         types.RouterHeader
	HeartbeatAck      types.RouterHeader
	TimeSync          types.RouterHeader
	User              types.RouterHeader
	Game              types.RouterHeader
	GameToDelta       types.RouterHeader
	Router            types.RouterHeader
	Status            types.RouterHeader
	Close             types.RouterHeader
	ConnectionRefused types.RouterHeader
	GameToUser        types.RouterHeader
	UserToGame        types.RouterHeader
	Connect           types.RouterHeader
	Disconnect        types.RouterHeader
}{
	Heartbeat:         "HB",
	HeartbeatAck:      "HBA",
	TimeSync:          "TS",
	User:              "US",
	Game:              "GA",
	GameToDelta:       "GD",
	Router:            "RU",
	Status:            "ST",
	Close:             "CL",
	ConnectionRefused: "CR",
	GameToUser:        "GU",
	UserToGame:        "UG",
	Connect:           "CO",
	Disconnect:        "DC",
}

var UserPayloadURI = struct {
	GetFriends    types.PayloadURI
	UpdateFriends types.PayloadURI
	UserCreate    types.PayloadURI
	LoadGameState types.PayloadURI
	SaveGameState types.PayloadURI
	GetAll        types.PayloadURI
}{
	GetFriends:    "GF",
	UpdateFriends: "UF",
	UserCreate:    "UC",
	LoadGameState: "LG",
	SaveGameState: "SG",
	GetAll:        "GA",
}

var (
	RedisCluster = []string{"localhost:6379", "localhost:6379"}
)

var MongoDBURI = "mongodb://localhost:27017"
var MongoDBName = "case"
var MongoDBCollection = "users"
var SaveGameStateSubject = "saveGameState"
var SaveGameStateSubjectQ = "saveGameStateQ"
var UpdateFriendsSubject = "updateFriends"
var UpdateFriendsSubjectQ = "updateFriendsQ"
