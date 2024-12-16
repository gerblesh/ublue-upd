package flatpak

import (
	"log/slog"
	"os/exec"
	"strings"

	. "github.com/ublue-os/uupd/drv/generic"
	"github.com/ublue-os/uupd/pkg/percent"
	"github.com/ublue-os/uupd/pkg/session"
)

type FlatpakUpdater struct {
	Config       DriverConfiguration
	Tracker      *TrackerConfiguration
	binaryPath   string
	users        []session.User
	usersEnabled bool
}

func (up FlatpakUpdater) Steps() int {
	if up.Config.Enabled {
		var steps = 1
		if up.usersEnabled {
			steps += len(up.users)
		}
		return steps
	}
	return 0
}

func (up FlatpakUpdater) New(config UpdaterInitConfiguration) (FlatpakUpdater, error) {
	userdesc := "Apps for User:"
	up.Config = DriverConfiguration{
		Title:           "Flatpak",
		Description:     "System Apps",
		UserDescription: &userdesc,
		Enabled:         true,
		MultiUser:       true,
		DryRun:          config.DryRun,
		Environment:     config.Environment,
	}
	up.Config.Logger = config.Logger.With(slog.String("module", strings.ToLower(up.Config.Title)))
	up.usersEnabled = false
	up.Tracker = nil

	binaryPath, exists := up.Config.Environment["UUPD_FLATPAK_BINARY"]
	if !exists || binaryPath == "" {
		up.binaryPath = "/usr/bin/flatpak"
	} else {
		up.binaryPath = binaryPath
	}

	return up, nil
}

func (up *FlatpakUpdater) SetUsers(users []session.User) {
	up.users = users
	up.usersEnabled = true
}

func (up FlatpakUpdater) Check() (bool, error) {
	return true, nil
}

func (up FlatpakUpdater) Update() (*[]CommandOutput, error) {
	var finalOutput = []CommandOutput{}

	if up.Config.DryRun {
		percent.ChangeTrackerMessageFancy(*up.Tracker.Writer, up.Tracker.Tracker, up.Tracker.Progress, percent.TrackerMessage{Title: up.Config.Title, Description: up.Config.Description})
		up.Tracker.Tracker.IncrementSection(nil)

		var err error = nil
		for _, user := range up.users {
			up.Tracker.Tracker.IncrementSection(err)
			percent.ChangeTrackerMessageFancy(*up.Tracker.Writer, up.Tracker.Tracker, up.Tracker.Progress, percent.TrackerMessage{Title: up.Config.Title, Description: *up.Config.UserDescription + " " + user.Name})
		}
		return &finalOutput, nil
	}

	percent.ChangeTrackerMessageFancy(*up.Tracker.Writer, up.Tracker.Tracker, up.Tracker.Progress, percent.TrackerMessage{Title: up.Config.Title, Description: up.Config.Description})
	cli := []string{up.binaryPath, "update", "-y", "--noninteractive"}
	flatpakCmd := exec.Command(cli[0], cli[1:]...)
	out, err := session.RunLog(up.Config.Logger, slog.LevelDebug, flatpakCmd)
	tmpout := CommandOutput{}.New(out, err)
	tmpout.Context = up.Config.Description
	tmpout.Cli = cli
	tmpout.Failure = err != nil
	finalOutput = append(finalOutput, *tmpout)

	err = nil
	for _, user := range up.users {
		up.Tracker.Tracker.IncrementSection(err)
		context := *up.Config.UserDescription + " " + user.Name
		percent.ChangeTrackerMessageFancy(*up.Tracker.Writer, up.Tracker.Tracker, up.Tracker.Progress, percent.TrackerMessage{Title: up.Config.Title, Description: context})
		cli := []string{up.binaryPath, "update", "-y"}
		out, err := session.RunUID(up.Config.Logger, slog.LevelDebug, user.UID, cli, nil)
		tmpout = CommandOutput{}.New(out, err)
		tmpout.Context = context
		tmpout.Cli = cli
		tmpout.Failure = err != nil
		finalOutput = append(finalOutput, *tmpout)
	}
	return &finalOutput, nil
}
