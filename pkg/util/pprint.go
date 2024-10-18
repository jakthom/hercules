package util

import (
	"encoding/json"
	"fmt"
)

func Pprint(i interface{}) string {
	payload, _ := json.MarshalIndent(i, "", "\t")
	stringified := string(payload)
	fmt.Println(stringified)
	return stringified
}

func Stringify(i interface{}) string {
	payload, _ := json.Marshal(i)
	return string(payload)
}
