package common

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strings"

	log "github.com/sirupsen/logrus"
)

//Recover from panic
func Recover(msg string) {
	if err := recover(); err != nil {
		log.Error(msg, err)
		log.Error(IdentifyPanic())
		log.Errorf("%s: %s", err, debug.Stack())
	}
}

//WithRecover recovers a panic in go routine
func WithRecover(routine func(), msg string) {
	defer Recover(msg)
	routine()
}

func FNV64(data []byte) uint64 {
	var hash uint64 = 14695981039346656037
	for _, c := range data {
		hash *= 1099511628211
		hash ^= uint64(c)
	}
	return hash
}

func IdentifyPanic() string {
	var name, file string
	var line int
	var pc [16]uintptr

	n := runtime.Callers(3, pc[:])
	for _, pc := range pc[:n] {
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		file, line = fn.FileLine(pc)
		name = fn.Name()
		if !strings.HasPrefix(name, "runtime.") {
			break
		}
	}

	switch {
	case name != "":
		return fmt.Sprintf("%v:%v", name, line)
	case file != "":
		return fmt.Sprintf("%v:%v", file, line)
	}

	return fmt.Sprintf("pc:%x", pc)
}

// GetHostName returns HostName reported by kernel, empty string if error
func GetHostName() string {
	hostName, err := os.Hostname()
	if err != nil {
		hostName = ""
	}
	return hostName
}

func GetPayload(raw *json.RawMessage, output interface{}) error {
	err := json.Unmarshal(*raw, &output)
	if err != nil {
		log.Warnf("GetPayload error: unmarshalling: %v %s", err, string(*raw))
	}
	return err
}
