package handler

// Result describes the information about a successful EBS volume snapshot.
type Result struct {
	Name       string `json:"Name"`
	SnapshotID string `json:"SnapshotID"`
	VolumeID   string `json:"VolumeID"`
	Error      string `json:"Error"`
}

// Response contains a list of Results.
type Response []Result
