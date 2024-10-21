package main

import "github.com/dbecorp/hercules/pkg/util"

type Thing map[string]string

func main() {
	myThing := Thing{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	for k := range myThing {
		util.Pprint(k)
	}
	util.Pprint(myThing)
}
