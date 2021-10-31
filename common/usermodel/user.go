package usermodel

import (
	"case/common/types"
	"encoding/json"
	"time"

	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
)

type User struct {
	UID         uuid.UUID `bson:"_id,omitempty"`
	Name        string    `bson:"name"`
	CreatedAt   time.Time `bson:"createdAt"`
	JSON        bson.M    `bson:"-"`
	Score       int64     `bson:"score,omitempty"`
	GamesPlayed int64     `bson:"gamesPlayed,omitempty"`
	Highscore   int64     `bson:"highScore,omitempty"`
}

func (user *User) GetReceiptExpiry() time.Duration {
	expiry := 3 * time.Hour
	return expiry
}

func (user *User) GetGameStateByte() []byte {
	var receipt = user.GetGameState()
	out, err := json.Marshal(receipt)
	if err != nil {
		log.Errorf("GetReceipt error: marshalling receipt: %v", err)
		return []byte(`{}`)
	}
	return out
}

func (user *User) GetGameState() types.GameState {
	return types.GameState{
		UID:         user.UID,
		GamesPlayed: user.GamesPlayed,
		Highscore:   user.Highscore,
	}
}
