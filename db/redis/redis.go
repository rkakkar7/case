package redis

import (
	"case/common"
	"case/common/constants"
	"case/common/types"
	"encoding/json"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/go-redis/redis"
)

var (
	redisCluster []*redis.Client
)

const (
	Nil = redis.Nil
)

//Init Redis clients
func Init() {
	var err error
	redisCluster = make([]*redis.Client, len(constants.RedisCluster))
	for i := 0; i < len(constants.RedisCluster); i++ {
		address := constants.RedisCluster[i]
		client := redis.NewClient(&redis.Options{
			Addr: address,
		})

		_, err = client.Ping().Result()
		if err != nil {
			log.Fatalf("redis[%s]: error: %v", address, err)
		}
		redisCluster[i] = client
	}
}

func GetRedis(key string) *redis.Client {
	id := common.FNV64([]byte(key)) % uint64(len(redisCluster))
	return redisCluster[id]
}

func GetReceipt(userID string) (types.GameState, error) {
	var gameState types.GameState
	GameStateBytes, err := GetRedis(userID).Get("gamestate:" + userID).Bytes()
	if err == nil {
		err = json.Unmarshal(GameStateBytes, &gameState)
	}
	return gameState, err
}

func GetGameStates(friends []string) map[string]string {
	members := make(map[*redis.Client][]string)
	for _, value := range friends {
		client := GetRedis(value)
		members[client] = append(members[client], "gamestate:"+value)
	}

	gameStates := make(map[string]string, len(friends))
	stateLock := &sync.Mutex{}

	wg := sync.WaitGroup{}
	wg.Add(len(members))
	for client, keys := range members {
		client := client
		keys := keys
		go func(wg *sync.WaitGroup) {
			reply, err := client.MGet(keys...).Result()
			stateLock.Lock()
			defer stateLock.Unlock()
			if err != nil {
				log.Errorf("GetGameStates: redis: receipt %v", err)
			} else {
				for index, key := range keys {
					res := reply[index]
					if res != nil {
						memberID := strings.Split(key, ":")[1]
						gameStates[memberID] = res.(string)
					}
				}
			}
			wg.Done()
		}(&wg)
	}
	wg.Wait()

	return gameStates
}
