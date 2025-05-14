package util

import (
	"encoding/json"

	"github.com/rs/zerolog/log"
)

func Pprint(i interface{}) string {
	payload, _ := json.MarshalIndent(i, "", "\t")
	stringified := string(payload)
	log.Debug().Msg(stringified)
	return stringified
}

func Stringify(i interface{}) string {
	payload, _ := json.Marshal(i)
	return string(payload)
}
