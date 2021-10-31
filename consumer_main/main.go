package main

import (
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"case/consumer"
	"case/db/redis"

	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

func main() {
	redis.Init()
	consumer.Start()
	exitSignal := make(chan os.Signal)
	signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
	<-exitSignal
}
