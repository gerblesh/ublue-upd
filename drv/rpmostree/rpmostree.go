package rpmostree

// Temporary: WILL get removed at some point.
// FIXME: Remove this on Spring 2025 when we all move to dnf5 and bootc ideally

import (
	"encoding/json"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	. "github.com/ublue-os/uupd/drv/generic"
	"github.com/ublue-os/uupd/pkg/session"
)

type rpmOstreeStatus struct {
	Deployments []struct {
		Timestamp int64 `json:"timestamp"`
	} `json:"deployments"`
}

type RpmOstreeUpdater struct {
	Config     DriverConfiguration
	BinaryPath string
}

// Checks if it is at least a month old considering how that works
func IsOutdatedOneMonthTimestamp(current time.Time, target time.Time) bool {
	return target.Before(current.AddDate(0, -1, 0))
}

func (up RpmOstreeUpdater) Outdated() (bool, error) {
	if up.Config.DryRun {
		return false, nil
	}
	var timestamp time.Time

	cmd := exec.Command(up.BinaryPath, "status", "--json", "--booted")
	out, err := session.RunLog(up.Config.Logger, slog.LevelDebug, cmd)
	if err != nil {
		return false, err
	}
	var status rpmOstreeStatus
	err = json.Unmarshal(out, &status)
	if err != nil {
		return false, err
	}
	timestamp = time.Unix(status.Deployments[0].Timestamp, 0).UTC()

	return IsOutdatedOneMonthTimestamp(time.Now(), timestamp), nil
}

func (up RpmOstreeUpdater) Update() (*[]CommandOutput, error) {
	var finalOutput = []CommandOutput{}
	var cmd *exec.Cmd
	binaryPath := up.BinaryPath
	cli := []string{binaryPath, "upgrade"}
	out, err := session.RunLog(up.Config.Logger, slog.LevelDebug, cmd)
	tmpout := CommandOutput{}.New(out, err)
	tmpout.Cli = cli
	tmpout.Failure = err != nil
	tmpout.Context = "System Update"
	finalOutput = append(finalOutput, *tmpout)
	return &finalOutput, err
}

func (up RpmOstreeUpdater) Steps() int {
	if up.Config.Enabled {
		return 1
	}
	return 0
}

func (up RpmOstreeUpdater) New(config UpdaterInitConfiguration) (RpmOstreeUpdater, error) {
	up.Config = DriverConfiguration{
		Title:       "System",
		Description: "System Updates",
		Enabled:     !config.Ci,
		DryRun:      config.DryRun,
		Environment: config.Environment,
	}
	up.Config.Logger = config.Logger.With(slog.String("module", strings.ToLower(up.Config.Title)))
	up.BinaryPath = EnvOrFallback(up.Config.Environment, "UUPD_RPMOSTREE_BINARY", "/usr/bin/rpm-ostree")

	return up, nil
}

func (up RpmOstreeUpdater) Check() (bool, error) {
	if up.Config.DryRun {
		return true, nil
	}

	// This function may or may not be accurate, rpm-ostree updgrade --check has issues... https://github.com/coreos/rpm-ostree/issues/1579
	// Not worried because we will end up removing rpm-ostree from the equation soon
	cmd := exec.Command(up.BinaryPath, "upgrade", "--check")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return true, err
	}

	updateNecessary := strings.Contains(string(out), "AvailableUpdate")
	up.Config.Logger.Debug("Executed update check", slog.String("output", string(out)), slog.Bool("update", updateNecessary))
	return updateNecessary, nil
}
