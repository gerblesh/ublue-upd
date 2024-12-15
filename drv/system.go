package drv

import (
	"encoding/json"
	"log/slog"
	"os/exec"
	"strings"
	"time"
)

type bootcStatus struct {
	Status struct {
		Booted struct {
			Incompatible bool `json:"incompatible"`
			Image        struct {
				Timestamp string `json:"timestamp"`
			} `json:"image"`
		} `json:"booted"`
		Staged struct {
			Incompatible bool `json:"incompatible"`
			Image        struct {
				Timestamp string `json:"timestamp"`
			} `json:"image"`
		}
	} `json:"status"`
}

// Workaround interface to decouple individual drivers
// (TODO: Remove this whenever rpm-ostree driver gets deprecated)
type SystemUpdateDriver interface {
	Steps() int
	Outdated() (bool, error)
	Check() (bool, error)
	Update() (*[]CommandOutput, error)
	Config() DriverConfiguration
	SetEnabled(value bool)
}

type SystemUpdater struct {
	config     DriverConfiguration
	BinaryPath string
}

func (dr SystemUpdater) Outdated() (bool, error) {
	if dr.config.DryRun {
		return false, nil
	}
	oneMonthAgo := time.Now().AddDate(0, -1, 0)
	var timestamp time.Time
	cmd := exec.Command(dr.BinaryPath, "status", "--format=json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, err
	}
	var status bootcStatus
	err = json.Unmarshal(out, &status)
	if err != nil {
		return false, err
	}
	timestamp, err = time.Parse(time.RFC3339Nano, status.Status.Booted.Image.Timestamp)
	if err != nil {
		return false, nil
	}
	return timestamp.Before(oneMonthAgo), nil
}

func (dr SystemUpdater) Update() (*[]CommandOutput, error) {
	var finalOutput = []CommandOutput{}
	var cmd *exec.Cmd
	binaryPath := dr.BinaryPath
	cli := []string{binaryPath, "upgrade"}
	cmd = exec.Command(cli[0], cli[1:]...)
	out, err := cmd.CombinedOutput()
	tmpout := CommandOutput{}.New(out, err)
	if err != nil {
		tmpout.SetFailureContext("System update")
	}
	finalOutput = append(finalOutput, *tmpout)
	return &finalOutput, err
}

func (up SystemUpdater) Steps() int {
	if up.config.Enabled {
		return 1
	}
	return 0
}

func (up SystemUpdater) New(config UpdaterInitConfiguration) (SystemUpdater, error) {
	up.config = DriverConfiguration{
		Title:       "Bootc",
		Description: "System Image",
		Enabled:     !config.Ci,
		DryRun:      config.DryRun,
		Environment: config.Environment,
	}

	if up.config.DryRun {
		return up, nil
	}

	bootcBinaryPath, exists := up.config.Environment["UUPD_BOOTC_BINARY"]
	if !exists || bootcBinaryPath == "" {
		up.BinaryPath = "/usr/bin/bootc"
	} else {
		up.BinaryPath = bootcBinaryPath
	}
	slog.Debug("Reported bootc binary path", slog.String("binary", up.BinaryPath))

	return up, nil
}

func (up SystemUpdater) Check() (bool, error) {
	if up.config.DryRun {
		return true, nil
	}

	cmd := exec.Command(up.BinaryPath, "upgrade", "--check")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return true, err
	}
	updateNecessary := !strings.Contains(string(out), "No changes in:")
	slog.Debug("Executed bootc update check", slog.String("output", string(out)), slog.Bool("necessary_update", updateNecessary))
	return updateNecessary, nil
}

func (up SystemUpdater) Config() DriverConfiguration {
	return up.config
}

func (up SystemUpdater) SetEnabled(value bool) {
	up.config.Enabled = value
}
