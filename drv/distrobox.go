package drv

import (
	"log/slog"
	"strings"

	"github.com/ublue-os/uupd/pkg/percent"
	"github.com/ublue-os/uupd/pkg/session"
)

type DistroboxUpdater struct {
	Config       DriverConfiguration
	Tracker      *TrackerConfiguration
	binaryPath   string
	users        []session.User
	usersEnabled bool
}

func (up DistroboxUpdater) Steps() int {
	if up.Config.Enabled {
		var steps = 1
		if up.usersEnabled {
			steps += len(up.users)
		}
		return steps
	}
	return 0
}

func (up DistroboxUpdater) New(config UpdaterInitConfiguration) (DistroboxUpdater, error) {
	userdesc := "Distroboxes for User:"
	up.Config = DriverConfiguration{
		Title:           "Distrobox",
		Description:     "Rootful Distroboxes",
		UserDescription: &userdesc,
		Enabled:         true,
		MultiUser:       true,
		DryRun:          config.DryRun,
		Environment:     config.Environment,
	}
	up.Config.logger = config.Logger.With(slog.String("module", strings.ToLower(up.Config.Title)))
	up.usersEnabled = false
	up.Tracker = nil

	binaryPath, exists := up.Config.Environment["UUPD_DISTROBOX_BINARY"]
	if !exists || binaryPath == "" {
		up.binaryPath = "/usr/bin/distrobox"
	} else {
		up.binaryPath = binaryPath
	}

	return up, nil
}

func (up *DistroboxUpdater) SetUsers(users []session.User) {
	up.users = users
	up.usersEnabled = true
}

func (up DistroboxUpdater) Check() (bool, error) {
	return true, nil
}

func (up DistroboxUpdater) Update() (*[]CommandOutput, error) {
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
	cli := []string{up.binaryPath, "upgrade", "-a"}
	out, err := session.RunUID(up.Config.logger, slog.LevelDebug, 0, cli, nil)
	tmpout := CommandOutput{}.New(out, err)
	tmpout.Context = up.Config.Description
	tmpout.Cli = cli
	tmpout.Failure = err != nil
	finalOutput = append(finalOutput, *tmpout)

	err = nil
	for _, user := range up.users {
		up.Tracker.Tracker.IncrementSection(err)
		context := *up.Config.UserDescription + " " + user.Name
		percent.ChangeTrackerMessageFancy(*up.Tracker.Writer, up.Tracker.Tracker, up.Tracker.Progress, percent.TrackerMessage{Title: up.Config.Title, Description: *up.Config.UserDescription + " " + user.Name})
		cli := []string{up.binaryPath, "upgrade", "-a"}
		out, err := session.RunUID(up.Config.logger, slog.LevelDebug, user.UID, cli, nil)
		tmpout = CommandOutput{}.New(out, err)
		tmpout.Context = context
		tmpout.Cli = cli
		tmpout.Failure = err != nil
		finalOutput = append(finalOutput, *tmpout)
	}
	return &finalOutput, nil
}

func (up *DistroboxUpdater) Logger() *slog.Logger {
	return up.Config.logger
}

func (up *DistroboxUpdater) SetLogger(logger *slog.Logger) {
	up.Config.logger = logger
}
