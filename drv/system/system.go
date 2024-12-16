package system

import (
	"encoding/json"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	. "github.com/ublue-os/uupd/drv/generic"
	"github.com/ublue-os/uupd/drv/rpmostree"
	"github.com/ublue-os/uupd/pkg/session"
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
}

type SystemUpdater struct {
	Config     DriverConfiguration
	BinaryPath string
}

// Checks if it is at least a month old considering how that works
func IsOutdatedOneMonthTimestamp(current time.Time, target time.Time) bool {
	return target.Before(current.AddDate(0, -1, 0))
}

func (up SystemUpdater) Outdated() (bool, error) {
	if up.Config.DryRun {
		return false, nil
	}

	cmd := exec.Command(up.BinaryPath, "status", "--format=json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, err
	}

	var status bootcStatus
	err = json.Unmarshal(out, &status)
	if err != nil {
		return false, err
	}

	timestamp, err := time.Parse(time.RFC3339Nano, status.Status.Booted.Image.Timestamp)
	if err != nil {
		return false, nil
	}
	return IsOutdatedOneMonthTimestamp(time.Now(), timestamp), nil
}

func (up SystemUpdater) Update() (*[]CommandOutput, error) {
	var finalOutput = []CommandOutput{}
	var cmd *exec.Cmd
	binaryPath := up.BinaryPath
	cli := []string{binaryPath, "upgrade", "--quiet"}
	up.Config.Logger.Debug("Executing update", slog.Any("cli", cli))
	cmd = exec.Command(cli[0], cli[1:]...)
	out, err := session.RunLog(up.Config.Logger, slog.LevelDebug, cmd)
	tmpout := CommandOutput{}.New(out, err)
	tmpout.Failure = err != nil
	tmpout.Context = "System Update"
	finalOutput = append(finalOutput, *tmpout)
	return &finalOutput, err
}

func (up SystemUpdater) Steps() int {
	if up.Config.Enabled {
		return 1
	}
	return 0
}

func (up SystemUpdater) New(config UpdaterInitConfiguration) (SystemUpdater, error) {
	up.Config = DriverConfiguration{
		Title:       "System",
		Description: "System Image",
		Enabled:     !config.Ci,
		DryRun:      config.DryRun,
		Environment: config.Environment,
	}
	up.Config.Logger = config.Logger.With(slog.String("module", strings.ToLower(up.Config.Title)))
	up.BinaryPath = EnvOrFallback(config.Environment, "UUPD_BOOTC_BINARY", "/usr/bin/bootc")

	return up, nil
}

func (up SystemUpdater) Check() (bool, error) {
	if up.Config.DryRun {
		return true, nil
	}

	cmd := exec.Command(up.BinaryPath, "upgrade", "--check")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return true, err
	}

	updateNecessary := !strings.Contains(string(out), "No changes in:")
	up.Config.Logger.Debug("Executed update check", slog.String("output", string(out)), slog.Bool("update", updateNecessary))
	return updateNecessary, nil
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

func InitializeSystemDriver(initConfiguration UpdaterInitConfiguration) (SystemUpdateDriver, DriverConfiguration, bool, error) {
	var enableUpd bool = true

	rpmOstreeUpdater, err := rpmostree.RpmOstreeUpdater{}.New(initConfiguration)
	if err != nil {
		enableUpd = false
	}

	systemUpdater, err := SystemUpdater{}.New(initConfiguration)
	if err != nil {
		enableUpd = false
	}

	isBootc, err := BootcCompatible(systemUpdater.BinaryPath)
	if err != nil {
		isBootc = false
	}

	if !isBootc {
		slog.Debug("Using rpm-ostree fallback as system driver")
	}

	// The system driver to be applied needs to have the correct "enabled" value since it will NOT update from here onwards.
	systemUpdater.Config.Enabled = systemUpdater.Config.Enabled && isBootc && enableUpd
	rpmOstreeUpdater.Config.Enabled = rpmOstreeUpdater.Config.Enabled && !isBootc && enableUpd

	var finalConfig DriverConfiguration
	var mainSystemDriver SystemUpdateDriver
	if isBootc {
		mainSystemDriver = &systemUpdater
		finalConfig = systemUpdater.Config
	} else {
		mainSystemDriver = &rpmOstreeUpdater
		finalConfig = systemUpdater.Config
	}

	return mainSystemDriver, finalConfig, isBootc, err
}
