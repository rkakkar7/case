package main

import (
	"math/rand"
	"net/http"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	_ "net/http/pprof"

	"case/common/constants"
	"case/db/redis"
	"case/userserver"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

func main() {
	redis.Init(true, true)
	rand.Seed(time.Now().UnixNano())
	go func() {
		log.Println(http.ListenAndServe(":"+strconv.Itoa(constants.UserserverPort-1000), nil))
	}()
	userserver.ListenAndServe(constants.UserserverPort)
}
