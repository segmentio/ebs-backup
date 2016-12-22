package engine

import (
	"errors"
	"sort"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/tj/go-sync/semaphore"
)

// byTime sorts snapshots by time.
type byTime []*ec2.Snapshot

func (v byTime) Less(i, j int) bool { return (*v[i].StartTime).After(*v[j].StartTime) }
func (v byTime) Len() int           { return len(v) }
func (v byTime) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }

// Result represents a backup result.
type Result struct {
	VolumeID         string
	CreatedSnapshot  string
	DeletedSnapshots []string
	CopiedTags       bool
	Err              error
}

// Config is the engine Config.
type Config struct {
	EC2      *ec2.EC2
	Device   string
	Name     string
	Limit    int
	CopyTags bool
}

// Engine represents a backup engine.
type Engine struct {
	Config
}

// New returns a new Engine.
func New(c Config) Engine {
	return Engine{c}
}

// Run runs the backups for all volumes that are
// matched by the configured `.Name` tag.
// The method returns a slice of results or an error
// if backups were not started. If a slice of results
// is returned each result should be checked for `.Err`.
func (e *Engine) Run() ([]Result, error) {
	volumes, err := e.volumes()
	if err != nil {
		return nil, err
	}

	sema := make(semaphore.Semaphore, 10)
	resc := make(chan Result)

	go func() {
		for _, v := range volumes {
			volume := v

			sema.Run(func() {
				res := e.backup(volume)
				res.VolumeID = *volume.VolumeId
				resc <- res
			})
		}

		sema.Wait()
		close(resc)
	}()

	results := make([]Result, 0, len(volumes))

	for res := range resc {
		results = append(results, res)
	}

	return results, nil
}

// Volume returns all volumes that need backup.
//
// The method returns all volumes that satisfy
// all the given rules:
//
//    - Have a tag "Name" that matches the configured `.Name`
//    - Have an `attachment.status` of `"attached"`
//    - Attached at the configured `.Device`
//
func (e *Engine) volumes() ([]*ec2.Volume, error) {
	resp, err := e.EC2.DescribeVolumes(&ec2.DescribeVolumesInput{
		Filters: []*ec2.Filter{
			filter("attachment.status", "attached"),
			filter("attachment.device", e.Device),
			filter("tag:Name", e.Name),
		},
	})
	if err != nil {
		return nil, err
	}

	return resp.Volumes, nil
}

// Backup will create a snapshot for the given `v`.
//
// Backup will first check if there is a snapshot
// in-progress if there is, it will abort and return
// a result with `.Err`.
//
// After the snapshot is created the method copies the
// volume tags and adds them to the snapshot, if `.CopyTags` is true.
//
// The method then checks if there's a need to delete
// the oldest snapshot for the volume and does so if `len(snapshots) > .Limit`.
func (e *Engine) backup(v *ec2.Volume) Result {
	var res Result

	snapshots, err := e.snapshots(*v.VolumeId)
	if err != nil {
		res.Err = err
		return res
	}

	for _, s := range snapshots {
		if *s.State == "PENDING" {
			res.Err = errors.New("volume has a snapshot in pending state")
			return res
		}
	}

	s, err := e.EC2.CreateSnapshot(&ec2.CreateSnapshotInput{
		VolumeId: v.VolumeId,
	})
	if err != nil {
		res.Err = err
		return res
	}
	res.CreatedSnapshot = *s.SnapshotId
	snapshots = append(snapshots, s)

	if len(snapshots) > e.Limit {
		set := byTime(snapshots)
		sort.Sort(set)

		ids, err := e.delete(set[e.Limit:])
		if err != nil {
			res.Err = err
			return res
		}

		res.DeletedSnapshots = ids
	}

	if e.CopyTags {
		_, err := e.EC2.CreateTags(&ec2.CreateTagsInput{
			Resources: []*string{s.SnapshotId},
			Tags:      v.Tags[:],
		})
		if err != nil {
			res.Err = err
			return res
		}

		res.CopiedTags = true
	}

	return res
}

// Snapshots returns all snapshots that belong to the volume `id`.
func (e *Engine) snapshots(id string) ([]*ec2.Snapshot, error) {
	resp, err := e.EC2.DescribeSnapshots(&ec2.DescribeSnapshotsInput{
		Filters: []*ec2.Filter{filter("volume-id", id)},
	})
	if err != nil {
		return nil, err
	}
	return resp.Snapshots, nil
}

// Delete deletes the given set of snapshots
// and returns ids of all deleted snapshots.
// If one of the snapshots fails to be deleted
// the error is returned immediately.
func (e *Engine) delete(set []*ec2.Snapshot) ([]string, error) {
	ids := make([]string, 0, len(set))

	for _, s := range set {
		_, err := e.EC2.DeleteSnapshot(&ec2.DeleteSnapshotInput{
			SnapshotId: s.SnapshotId,
		})
		if err != nil {
			return nil, err
		}

		ids = append(ids, *s.SnapshotId)
	}

	return ids, nil
}

// filter returns an ec2.Filter with `key`, `value`.
func filter(key, value string) *ec2.Filter {
	return &ec2.Filter{
		Name:   &key,
		Values: []*string{&value},
	}
}
