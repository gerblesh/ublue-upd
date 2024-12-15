package drv

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"syscall"

	"github.com/ublue-os/uupd/pkg/session"
)

func (up BrewUpdater) GetBrewUID() (int, error) {
	inf, err := os.Stat(up.BrewPrefix)
	if err != nil {
		return -1, err
	}

	if !inf.IsDir() {
		return -1, fmt.Errorf("Brew prefix: %v, is not a dir.", up.BrewPrefix)
	}
	stat, ok := inf.Sys().(*syscall.Stat_t)
	if !ok {
		return -1, fmt.Errorf("Unable to retriev UID info for %v", up.BrewPrefix)
	}
	return int(stat.Uid), nil
}

func (up BrewUpdater) Steps() int {
	if up.Config.Enabled {
		return 1
	}
	return 0
}

func (up BrewUpdater) Check() (bool, error) {
	return true, nil
}

func (up BrewUpdater) Update() (*[]CommandOutput, error) {
	var final_output = []CommandOutput{}

	if up.Config.DryRun {
		return &final_output, nil
	}

	cli := []string{up.BrewPath, "update"}
	out, err := session.RunUID(up.Config.logger, slog.LevelDebug, up.BaseUser, cli, up.Config.Environment)
	tmpout := CommandOutput{}.New(out, err)
	tmpout.Context = "Brew Update"
	tmpout.Cli = cli
	tmpout.Failure = err != nil
	if err != nil {
		tmpout.SetFailureContext("Brew update")
		final_output = append(final_output, *tmpout)
		return &final_output, err
	}

	cli = []string{up.BrewPath, "upgrade"}
	out, err = session.RunUID(up.Config.logger, slog.LevelDebug, up.BaseUser, cli, up.Config.Environment)
	tmpout = CommandOutput{}.New(out, err)
	tmpout.Context = "Brew Upgrade"
	tmpout.Cli = cli
	tmpout.Failure = err != nil
	final_output = append(final_output, *tmpout)
	return &final_output, err
}

type BrewUpdater struct {
	Config     DriverConfiguration
	BaseUser   int
	BrewRepo   string
	BrewPrefix string
	BrewCellar string
	BrewPath   string
}

func (up BrewUpdater) New(config UpdaterInitConfiguration) (BrewUpdater, error) {
	up.Config = DriverConfiguration{
		Title:       "Brew",
		Description: "CLI Apps",
		Enabled:     true,
		MultiUser:   false,
		DryRun:      config.DryRun,
		Environment: config.Environment,
	}
	up.Config.logger = config.Logger.With(slog.String("module", strings.ToLower(up.Config.Title)))

	brewPrefix, exists := up.Config.Environment["HOMEBREW_PREFIX"]
	if !exists || brewPrefix == "" {
		up.BrewPrefix = "/home/linuxbrew/.linuxbrew"
	} else {
		up.BrewPrefix = brewPrefix
	}
	brewRepo, exists := up.Config.Environment["HOMEBREW_REPOSITORY"]
	if !exists || brewRepo == "" {
		up.BrewRepo = fmt.Sprintf("%s/Homebrew", up.BrewPrefix)
	} else {
		up.BrewRepo = brewRepo
	}
	brewCellar, exists := up.Config.Environment["HOMEBREW_CELLAR"]
	if !exists || brewCellar == "" {
		up.BrewCellar = fmt.Sprintf("%s/Cellar", up.BrewPrefix)
	} else {
		up.BrewCellar = brewCellar
	}
	brewPath, exists := up.Config.Environment["HOMEBREW_PATH"]
	if !exists || brewPath == "" {
		up.BrewPath = fmt.Sprintf("%s/bin/brew", up.BrewPrefix)
	} else {
		up.BrewPath = brewPath
	}

	if up.Config.DryRun {
		return up, nil
	}

	uid, err := up.GetBrewUID()
	if err != nil {
		return up, err
	}
	up.BaseUser = uid

	return up, nil
}

func (up *BrewUpdater) Logger() *slog.Logger {
	return up.Config.logger
}

func (up *BrewUpdater) SetLogger(logger *slog.Logger) {
	up.Config.logger = logger
}
