package drv

// Temporary: WILL get removed at some point.
// FIXME: Remove this on Spring 2025 when we all move to dnf5 and bootc ideally

import (
	"encoding/json"
	"os/exec"
	"strings"
	"time"
)

type rpmOstreeStatus struct {
	Deployments []struct {
		Timestamp int64 `json:"timestamp"`
	} `json:"deployments"`
}

type RpmOstreeUpdater struct {
	config     DriverConfiguration
	BinaryPath string
}

func (up RpmOstreeUpdater) Outdated() (bool, error) {
	if up.config.DryRun {
		return false, nil
	}
	oneMonthAgo := time.Now().AddDate(0, -1, 0)
	var timestamp time.Time

	cmd := exec.Command(up.BinaryPath, "status", "--json", "--booted")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, err
	}
	var status rpmOstreeStatus
	err = json.Unmarshal(out, &status)
	if err != nil {
		return false, err
	}
	timestamp = time.Unix(status.Deployments[0].Timestamp, 0).UTC()

	return timestamp.Before(oneMonthAgo), nil
}

func (dr RpmOstreeUpdater) Update() (*[]CommandOutput, error) {
	var finalOutput = []CommandOutput{}
	var cmd *exec.Cmd
	binaryPath := dr.BinaryPath
	cli := []string{binaryPath, "upgrade"}
	cmd = exec.Command(cli[0], cli[1:]...)
	out, err := cmd.CombinedOutput()
	tmpout := CommandOutput{}.New(out, err)
	// tmpout.Cli = cli
	tmpout.Failure = err != nil
	tmpout.Context = "System Update"
	finalOutput = append(finalOutput, *tmpout)
	return &finalOutput, err
}

func (up RpmOstreeUpdater) Steps() int {
	if up.config.Enabled {
		return 1
	}
	return 0
}

func BootcCompatible(binaryPath string) (bool, error) {
	cmd := exec.Command(binaryPath, "status", "--format=json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, nil
	}
	var status bootcStatus
	err = json.Unmarshal(out, &status)
	if err != nil {
		return false, nil
	}
	return !(status.Status.Booted.Incompatible || status.Status.Staged.Incompatible), nil
}

func (up RpmOstreeUpdater) New(config UpdaterInitConfiguration) (RpmOstreeUpdater, error) {
	up.config = DriverConfiguration{
		Title:       "System",
		Description: "System Updates",
		Enabled:     !config.Ci,
		DryRun:      config.DryRun,
		Environment: config.Environment,
	}

	if up.config.DryRun {
		return up, nil
	}

	binaryPath, exists := up.config.Environment["UUPD_RPMOSTREE_BINARY"]
	if !exists || binaryPath == "" {
		up.BinaryPath = "/usr/bin/rpm-ostree"
	} else {
		up.BinaryPath = binaryPath
	}

	return up, nil
}

func (up RpmOstreeUpdater) Check() (bool, error) {
	if up.config.DryRun {
		return true, nil
	}

	// This function may or may not be accurate, rpm-ostree updgrade --check has issues... https://github.com/coreos/rpm-ostree/issues/1579
	// Not worried because we will end up removing rpm-ostree from the equation soon
	cmd := exec.Command(up.BinaryPath, "upgrade", "--check")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return true, err
	}
	return strings.Contains(string(out), "AvailableUpdate"), nil
}

func (up RpmOstreeUpdater) Config() DriverConfiguration {
	return up.config
}

func (up RpmOstreeUpdater) SetEnabled(value bool) {
	up.config.Enabled = value
}
