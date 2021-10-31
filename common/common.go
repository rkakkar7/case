package common

import (
	"encoding/json"
	"runtime/debug"

	log "github.com/sirupsen/logrus"
)

//Recover from panic
func Recover(msg string) {
	if err := recover(); err != nil {
		log.Error(msg, err)
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

func GetPayload(raw *json.RawMessage, output interface{}) error {
	err := json.Unmarshal(*raw, &output)
	if err != nil {
		log.Warnf("GetPayload error: unmarshalling: %v %s", err, string(*raw))
	}
	return err
}
