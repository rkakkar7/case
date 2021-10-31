package redis

import (
	"case/common/constants"

	log "github.com/sirupsen/logrus"

	"github.com/go-redis/redis"
)

var (
	redisCluster []*redis.Client
	redisPubSub  []*redis.Client
)

const (
	Nil = redis.Nil
)

//Init Redis clients
func Init(receipt, pubsub bool) {
	var err error

	if receipt {
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

	if pubsub {
		redisPubSub = make([]*redis.Client, len(constants.RedisPubSub))
		for i := 0; i < len(constants.RedisPubSub); i++ {
			address := constants.RedisPubSub[i]
			client := redis.NewClient(&redis.Options{
				Addr: address,
			})

			_, err = client.Ping().Result()
			if err != nil {
				log.Fatalf("redis[%s]: error: %v", address, err)
			}

			redisPubSub[i] = client
		}
	}
}

// func GetRedis(key string) *redis.Client {
// 	id := common.FNV64([]byte(key)) % uint64(len(redisCluster))
// 	return redisCluster[id]
// }
