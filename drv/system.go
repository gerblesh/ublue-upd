package drv

import (
	"encoding/json"
	"log/slog"
	"os/exec"
	"strings"
	"time"

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
	Logger() *slog.Logger
	SetLogger(value *slog.Logger)
}

type SystemUpdater struct {
	Config     DriverConfiguration
	BinaryPath string
}

func (up SystemUpdater) Outdated() (bool, error) {
	if up.Config.DryRun {
		return false, nil
	}
	oneMonthAgo := time.Now().AddDate(0, -1, 0)
	var timestamp time.Time
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
	timestamp, err = time.Parse(time.RFC3339Nano, status.Status.Booted.Image.Timestamp)
	if err != nil {
		return false, nil
	}
	return timestamp.Before(oneMonthAgo), nil
}

func (up SystemUpdater) Update() (*[]CommandOutput, error) {
	var finalOutput = []CommandOutput{}
	var cmd *exec.Cmd
	binaryPath := up.BinaryPath
	cli := []string{binaryPath, "upgrade", "--quiet"}
	up.Config.logger.Debug("Executing update", slog.Any("cli", cli))
	cmd = exec.Command(cli[0], cli[1:]...)
	out, err := session.RunLog(up.Config.logger, slog.LevelDebug, cmd)
	tmpout := CommandOutput{}.New(out, err)
	if err != nil {
		tmpout.SetFailureContext("System update")
	}
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
	up.Config.logger = config.Logger.With(slog.String("module", strings.ToLower(up.Config.Title)))
	if up.Config.DryRun {
		return up, nil
	}

	bootcBinaryPath, exists := up.Config.Environment["UUPD_BOOTC_BINARY"]
	if !exists || bootcBinaryPath == "" {
		up.BinaryPath = "/usr/bin/bootc"
	} else {
		up.BinaryPath = bootcBinaryPath
	}

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
	up.Config.logger.Debug("Executed update check", slog.String("output", string(out)), slog.Bool("update", updateNecessary))
	return updateNecessary, nil
}

func (up *SystemUpdater) Logger() *slog.Logger {
	return up.Config.logger
}

func (up *SystemUpdater) SetLogger(logger *slog.Logger) {
	up.Config.logger = logger
}
