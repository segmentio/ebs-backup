package main

import (
	"flag"
	"os"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/segmentio/ebs-backup/internal/engine"
)

var (
	version  = "v0.0.0"
	name     = flag.String("name", "", "name tags that identify the volumes")
	device   = flag.String("device", "", "the device name")
	limit    = flag.Int("limit", 5, "maximum number of snapshots to keep per volume")
	copyTags = flag.Bool("copy-tags", true, "copy volume tags to the snapshot")
)

func init() {
	log.SetHandler(cli.Default)
	log.SetLevel(log.InfoLevel)
}

func main() {
	flag.Parse()

	if *limit > 1000 || *limit <= 1 {
		log.Fatal("--limit must be less than 1000 and greater than 1")
	}

	if *name == "" {
		log.Fatal("--name must be the volume .Name tag")
	}

	if *device == "" {
		log.Fatal("--device must be given")
	}

	e := engine.New(engine.Config{
		EC2:      ec2.New(session.New(aws.NewConfig())),
		Name:     *name,
		Limit:    *limit,
		Device:   *device,
		CopyTags: *copyTags,
	})

	results, err := e.Run()
	if err != nil {
		log.WithError(err).Fatal("error")
	}

	var code int

	for _, res := range results {
		ctx := log.WithFields(log.Fields{
			"volume":      res.VolumeID,
			"created":     res.CreatedSnapshot,
			"deleted":     res.DeletedSnapshots,
			"copied_tags": res.CopiedTags,
		})

		if res.Err != nil {
			ctx.WithError(res.Err).Error("backup")
			code = 1
			continue
		}

		ctx.Info("backup")
	}

	os.Exit(code)
}
