package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/apex/log"
	"github.com/apex/log/handlers/logfmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/segmentio/ebs-backup/internal/engine"
	"github.com/segmentio/ebs-backup/internal/handler"
)

var env = []string{
	"VOLUME_NAME",
	"VOLUME_DEVICES",
	"SNAPSHOT_LIMIT",
	"COPY_TAGS",
}

func init() {
	log.SetHandler(logfmt.New(os.Stdout))
	log.SetLevel(log.InfoLevel)
}

func main() {
	lambda.Start(HandleRequest)
}

func HandleRequest() (r handler.Response, err error) {
	c, err := config()
	if err != nil {
		return r, err
	}

	e := engine.New(c)

	results, err := e.Run()
	if err != nil {
		return r, err
	}

	errOccurred := false

	for _, res := range results {
		fields := log.Fields{
			"name_tag":    e.Name,
			"snapshot_id": res.CreatedSnapshot,
			"volume_id":   res.VolumeID,
		}
		result := handler.Result{
			Name:       e.Name,
			SnapshotID: res.CreatedSnapshot,
			VolumeID:   res.VolumeID,
		}
		if res.Err == nil {
			log.WithFields(fields).Info("snapshot")
		} else {
			errOccurred = true
			fields["error"] = res.Err.Error()
			log.WithFields(fields).Error("snapshot")
			result.Error = res.Err.Error()
		}
		r = append(r, result)
	}

	if errOccurred {
		return r, errors.New("An error occurred")
	}
	return r, nil
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
