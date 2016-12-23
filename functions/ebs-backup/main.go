package main

import (
	"encoding/json"
	"strconv"

	"github.com/apex/go-apex"
	"github.com/segmentio/ebs-backup/internal/engine"
	"github.com/segmentio/go-env"
)

var (
	version  = "v0.0.0"
	name     = env.MustGet("VOLUME_NAME")
	device   = env.MustGet("VOLUME_DEVICE")
	limit    = parseInt("SNAPSHOT_LIMIT")
	copyTags = parseBool("COPY_TAGS")
)

func main() {
	e := engine.New(engine.Config{
		Name:     name,
		Device:   device,
		Limit:    limit,
		CopyTags: copyTags,
	})

	apex.HandleFunc(func(_ json.RawMessage, _ *apex.Context) (interface{}, error) {
		return e.Run()
	})
}

func parseInt(key string) int {
	v, err := strconv.Atoi(env.MustGet(key))
	if err != nil {
		panic("$" + key + ": " + err.Error())
	}

	return v
}

func parseBool(key string) bool {
	b, err := strconv.ParseBool(env.MustGet(key))
	if err != nil {
		panic("$" + key + ": " + err.Error())
	}

	return b
}
