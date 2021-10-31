package main

import (
	"case/common"
	"case/common/constants"
	"case/router"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	_ "net/http/pprof"
)

var r *router.Router

func init() {
	log.SetLevel(log.DebugLevel)
}

func main() {
	m := mux.NewRouter()
	r = router.NewRouter(m, "127.0.0.1:"+strconv.Itoa(constants.RouterPort))
	r.HandleRouting(m)
	log.Fatal(r.Serve())
	go gracefulExit()
}

func gracefulExit() {
	defer common.Recover("gracefulExit")
	exitSignal := make(chan os.Signal)
	signal.Notify(exitSignal, syscall.SIGTERM)
	sig := <-exitSignal
	if sig == syscall.SIGTERM {
		r.RemoveAllService()
		os.Exit(0)
	}
}
