package engine

import (
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/stretchr/testify/assert"
)

func TestVolumes(t *testing.T) {
	assert := assert.New(t)

	var filters []*ec2.Filter

	e := New(Config{
		Name:    "db-*",
		Devices: []string{"/dev/xvdf"},
		EC2: mock{
			DescribeVolumesFunc: func(req *ec2.DescribeVolumesInput) (*ec2.DescribeVolumesOutput, error) {
				filters = req.Filters
				return new(ec2.DescribeVolumesOutput), nil
			},
		},
	})

	_, err := e.volumes()
	assert.NoError(err)
	assert.Equal(3, len(filters))
	assert.Equal("attachment.status", *filters[0].Name)
	assert.Equal("attached", *filters[0].Values[0])
	assert.Equal("attachment.device", *filters[1].Name)
	assert.Equal("/dev/xvdf", *filters[1].Values[0])
	assert.Equal("tag:Name", *filters[2].Name)
	assert.Equal("db-*", *filters[2].Values[0])
}

func TestVolumesMultipleDevices(t *testing.T) {
	assert := assert.New(t)

	var filters []*ec2.Filter

	e := New(Config{
		Name:    "db-*",
		Devices: []string{"/dev/xvdf", "/dev/xvdi"},
		EC2: mock{
			DescribeVolumesFunc: func(req *ec2.DescribeVolumesInput) (*ec2.DescribeVolumesOutput, error) {
				filters = req.Filters
				return new(ec2.DescribeVolumesOutput), nil
			},
		},
	})

	_, err := e.volumes()
	assert.NoError(err)

	devices := aws.StringValueSlice(filters[1].Values)
	assert.Equal(2, len(devices))
	assert.Equal([]string{"/dev/xvdf", "/dev/xvdi"}, devices)
}

func TestVolumesErr(t *testing.T) {
	assert := assert.New(t)

	e := New(Config{
		EC2: mock{
			DescribeVolumesFunc: func(req *ec2.DescribeVolumesInput) (*ec2.DescribeVolumesOutput, error) {
				return nil, errors.New("boom")
			},
		},
	})

	_, err := e.volumes()
	assert.Error(err)
}

func TestSnapshots(t *testing.T) {
	assert := assert.New(t)

	var filters []*ec2.Filter

	e := New(Config{
		EC2: mock{
			DescribeSnapshotsFunc: func(req *ec2.DescribeSnapshotsInput) (*ec2.DescribeSnapshotsOutput, error) {
				filters = req.Filters
				return new(ec2.DescribeSnapshotsOutput), nil
			},
		},
	})

	_, err := e.snapshots("vol-xyz")
	assert.NoError(err)
	assert.Equal(1, len(filters))
	assert.Equal("volume-id", *filters[0].Name)
	assert.Equal("vol-xyz", *filters[0].Values[0])
}

func TestSnapshotsErr(t *testing.T) {
	assert := assert.New(t)

	e := New(Config{
		EC2: mock{
			DescribeSnapshotsFunc: func(req *ec2.DescribeSnapshotsInput) (*ec2.DescribeSnapshotsOutput, error) {
				return nil, errors.New("boom")
			},
		},
	})

	res := e.backup(&ec2.Volume{
		VolumeId: aws.String("vol-xyz"),
	})

	assert.Error(res.Err)
}

func TestSnapshotPending(t *testing.T) {
	assert := assert.New(t)

	e := New(Config{
		EC2: mock{
			DescribeSnapshotsFunc: func(req *ec2.DescribeSnapshotsInput) (*ec2.DescribeSnapshotsOutput, error) {
				return &ec2.DescribeSnapshotsOutput{
					Snapshots: []*ec2.Snapshot{
						{State: aws.String("pending")},
					},
				}, nil
			},
		},
	})

	res := e.backup(&ec2.Volume{
		VolumeId: aws.String("vol-xyz"),
	})

	assert.Error(res.Err)
}

func TestCreateSnapshot(t *testing.T) {
	assert := assert.New(t)

	var req *ec2.CreateSnapshotInput

	e := New(Config{
		Limit: 10,
		EC2: mock{
			DescribeSnapshotsFunc: func(req *ec2.DescribeSnapshotsInput) (*ec2.DescribeSnapshotsOutput, error) {
				return new(ec2.DescribeSnapshotsOutput), nil
			},
			CreateSnapshotFunc: func(i *ec2.CreateSnapshotInput) (*ec2.Snapshot, error) {
				req = i
				id := aws.String("snap-xyz")
				return &ec2.Snapshot{SnapshotId: id}, nil
			},
		},
	})

	res := e.backup(&ec2.Volume{
		VolumeId: aws.String("vol-xyz"),
	})

	assert.NoError(res.Err)
	assert.Equal("snap-xyz", res.CreatedSnapshot)
	assert.Equal("vol-xyz", *req.VolumeId)
}

func TestCreateSnapshotErr(t *testing.T) {
	assert := assert.New(t)

	e := New(Config{
		EC2: mock{
			DescribeSnapshotsFunc: func(req *ec2.DescribeSnapshotsInput) (*ec2.DescribeSnapshotsOutput, error) {
				return new(ec2.DescribeSnapshotsOutput), nil
			},
			CreateSnapshotFunc: func(req *ec2.CreateSnapshotInput) (*ec2.Snapshot, error) {
				return nil, errors.New("boom")
			},
		},
	})

	res := e.backup(&ec2.Volume{
		VolumeId: aws.String("vol-xyz"),
	})

	assert.Error(res.Err)
}

func TestDeleteSnapshots(t *testing.T) {
	assert := assert.New(t)
	start := time.Unix(0, 0)

	snapshots := []*ec2.Snapshot{
		{
			SnapshotId: aws.String("snap-001"),
			StartTime:  aws.Time(start.Add(time.Hour * 1)),
			State:      aws.String("completed"),
		},
		{
			SnapshotId: aws.String("snap-002"),
			StartTime:  aws.Time(start.Add(time.Hour * 2)),
			State:      aws.String("completed"),
		},
		{
			SnapshotId: aws.String("snap-003"),
			StartTime:  aws.Time(start.Add(time.Hour * 3)),
			State:      aws.String("completed"),
		},
	}

	var deleted []string

	e := New(Config{
		Limit: 3,
		EC2: mock{
			DescribeSnapshotsFunc: func(req *ec2.DescribeSnapshotsInput) (*ec2.DescribeSnapshotsOutput, error) {
				return &ec2.DescribeSnapshotsOutput{
					Snapshots: snapshots,
				}, nil
			},

			CreateSnapshotFunc: func(*ec2.CreateSnapshotInput) (*ec2.Snapshot, error) {
				return &ec2.Snapshot{
					SnapshotId: aws.String("snap-004"),
					StartTime:  aws.Time(time.Now()),
				}, nil
			},

			DeleteSnapshotFunc: func(req *ec2.DeleteSnapshotInput) (*ec2.DeleteSnapshotOutput, error) {
				deleted = append(deleted, *req.SnapshotId)
				return nil, nil
			},
		},
	})

	res := e.backup(&ec2.Volume{
		VolumeId: aws.String("vol-xyz"),
	})

	assert.NoError(res.Err)
	assert.Equal(1, len(deleted))
	assert.Equal("snap-001", deleted[0])
	assert.Equal(1, len(res.DeletedSnapshots))
	assert.Equal("snap-001", res.DeletedSnapshots[0])
}

func TestDeleteErr(t *testing.T) {
	assert := assert.New(t)

	snapshots := []*ec2.Snapshot{
		{
			SnapshotId: aws.String("snap-001"),
			State:      aws.String("completed"),
			StartTime:  aws.Time(time.Now()),
		},
	}

	e := New(Config{
		Limit: 1,
		EC2: mock{
			DescribeSnapshotsFunc: func(req *ec2.DescribeSnapshotsInput) (*ec2.DescribeSnapshotsOutput, error) {
				return &ec2.DescribeSnapshotsOutput{
					Snapshots: snapshots,
				}, nil
			},

			DeleteSnapshotFunc: func(req *ec2.DeleteSnapshotInput) (*ec2.DeleteSnapshotOutput, error) {
				return nil, errors.New("boom")
			},

			CreateSnapshotFunc: func(req *ec2.CreateSnapshotInput) (*ec2.Snapshot, error) {
				return &ec2.Snapshot{
					SnapshotId: aws.String("snap-002"),
					StartTime:  aws.Time(time.Now()),
				}, nil
			},
		},
	})

	res := e.backup(&ec2.Volume{
		VolumeId: aws.String("vol-xyz"),
	})

	assert.Error(res.Err)
	assert.Equal(res.Err.Error(), "boom")
}

type mock struct {
	ec2iface.EC2API
	DescribeVolumesFunc   func(*ec2.DescribeVolumesInput) (*ec2.DescribeVolumesOutput, error)
	DescribeSnapshotsFunc func(*ec2.DescribeSnapshotsInput) (*ec2.DescribeSnapshotsOutput, error)
	CreateSnapshotFunc    func(*ec2.CreateSnapshotInput) (*ec2.Snapshot, error)
	DeleteSnapshotFunc    func(*ec2.DeleteSnapshotInput) (*ec2.DeleteSnapshotOutput, error)
	CreateTagsFunc        func(*ec2.CreateTagsInput) (*ec2.CreateTagsOutput, error)
}

func (m mock) DescribeVolumes(i *ec2.DescribeVolumesInput) (*ec2.DescribeVolumesOutput, error) {
	return m.DescribeVolumesFunc(i)
}

func (m mock) DescribeSnapshots(i *ec2.DescribeSnapshotsInput) (*ec2.DescribeSnapshotsOutput, error) {
	return m.DescribeSnapshotsFunc(i)
}

func (m mock) CreateSnapshot(i *ec2.CreateSnapshotInput) (*ec2.Snapshot, error) {
	return m.CreateSnapshotFunc(i)
}

func (m mock) DeleteSnapshot(i *ec2.DeleteSnapshotInput) (*ec2.DeleteSnapshotOutput, error) {
	return m.DeleteSnapshotFunc(i)
}

func (m mock) CreateTags(i *ec2.CreateTagsInput) (*ec2.CreateTagsOutput, error) {
	return m.CreateTagsFunc(i)
}
