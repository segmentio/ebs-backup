package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/apex/go-apex"
	"github.com/apex/log"
	"github.com/apex/log/handlers/logfmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/segmentio/ebs-backup/internal/engine"
)

const (
	version = "v0.0.0"
)

var env = []string{
	"VOLUME_NAME",
	"VOLUME_DEVICES",
	"SNAPSHOT_LIMIT",
	"COPY_TAGS",
}

func init() {
	log.SetHandler(logfmt.New(os.Stderr))
	log.SetLevel(log.InfoLevel)
}

func main() {
	apex.HandleFunc(func(_ json.RawMessage, _ *apex.Context) (interface{}, error) {
		c, err := config()
		if err != nil {
			return nil, err
		}

		e := engine.New(c)

		results, err := e.Run()
		if err != nil {
			return nil, err
		}

		for _, res := range results {
			if res.Err != nil {
				return nil, res.Err
			}
		}

		return results, nil
	})
}

func config() (c engine.Config, err error) {
	for _, name := range env {
		if v := os.Getenv(name); v == "" {
			return c, fmt.Errorf("$%s env var is empty", name)
		}
	}

	limit, err := parseInt("SNAPSHOT_LIMIT")
	if err != nil {
		return c, err
	}

	copytags, err := parseBool("COPY_TAGS")
	if err != nil {
		return c, err
	}

	if limit < 1 {
		return c, fmt.Errorf("$SNAPSHOT_LIMIT must be more than 1")
	}

	devices := split(os.Getenv("VOLUME_DEVICES"))
	if len(devices) == 0 {
		return c, fmt.Errorf("$VOLUME_DEVICES is required")
	}

	c.EC2 = ec2.New(session.New(aws.NewConfig()))
	c.Name = os.Getenv("VOLUME_NAME")
	c.Devices = devices
	c.Limit = limit
	c.CopyTags = copytags
	return c, nil
}

func parseInt(key string) (int, error) {
	v, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return -1, fmt.Errorf("$%s : %s", key, err)
	}

	return v, nil
}

func parseBool(key string) (bool, error) {
	v, err := strconv.ParseBool(os.Getenv(key))
	if err != nil {
		return false, fmt.Errorf("$%s : %s", key, err)
	}

	return v, nil
}

func split(s string) (ret []string) {
	for _, s := range strings.Split(s, ",") {
		ret = append(ret, strings.TrimSpace(s))
	}
	return ret
}
