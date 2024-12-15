package drv

import (
	"os/exec"

	"github.com/ublue-os/uupd/pkg/percent"
	"github.com/ublue-os/uupd/pkg/session"
)

type FlatpakUpdater struct {
	config       DriverConfiguration
	Tracker      *TrackerConfiguration
	binaryPath   string
	users        []session.User
	usersEnabled bool
}

func (up FlatpakUpdater) Steps() int {
	if up.config.Enabled {
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
	up.config = DriverConfiguration{
		Title:           "Flatpak",
		Description:     "System Apps",
		UserDescription: &userdesc,
		Enabled:         true,
		MultiUser:       true,
		DryRun:          config.DryRun,
		Environment:     config.Environment,
	}
	up.usersEnabled = false
	up.Tracker = nil

	binaryPath, exists := up.config.Environment["UUPD_FLATPAK_BINARY"]
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

	if up.config.DryRun {
		percent.ChangeTrackerMessageFancy(*up.Tracker.Writer, up.Tracker.Tracker, up.Tracker.Progress, percent.TrackerMessage{Title: up.config.Title, Description: up.config.Description})
		up.Tracker.Tracker.IncrementSection(nil)

		var err error = nil
		for _, user := range up.users {
			up.Tracker.Tracker.IncrementSection(err)
			percent.ChangeTrackerMessageFancy(*up.Tracker.Writer, up.Tracker.Tracker, up.Tracker.Progress, percent.TrackerMessage{Title: up.config.Title, Description: *up.config.UserDescription + " " + user.Name})
		}
		return &finalOutput, nil
	}

	percent.ChangeTrackerMessageFancy(*up.Tracker.Writer, up.Tracker.Tracker, up.Tracker.Progress, percent.TrackerMessage{Title: up.config.Title, Description: up.config.Description})
	cli := []string{up.binaryPath, "update", "-y"}
	flatpakCmd := exec.Command(cli[0], cli[1:]...)
	out, err := flatpakCmd.CombinedOutput()
	tmpout := CommandOutput{}.New(out, err)
	tmpout.Context = up.config.Description
	tmpout.Cli = cli
	tmpout.Failure = err != nil
	finalOutput = append(finalOutput, *tmpout)

	err = nil
	for _, user := range up.users {
		up.Tracker.Tracker.IncrementSection(err)
		context := *up.config.UserDescription + " " + user.Name
		percent.ChangeTrackerMessageFancy(*up.Tracker.Writer, up.Tracker.Tracker, up.Tracker.Progress, percent.TrackerMessage{Title: up.config.Title, Description: context})
		cli := []string{up.binaryPath, "update", "-y"}
		out, err := session.RunUID(user.UID, cli, nil)
		tmpout = CommandOutput{}.New(out, err)
		tmpout.Context = context
		tmpout.Cli = cli
		tmpout.Failure = err != nil
		finalOutput = append(finalOutput, *tmpout)
	}
	return &finalOutput, nil
}

func (up FlatpakUpdater) Config() DriverConfiguration {
	return up.config
}

func (up FlatpakUpdater) SetEnabled(value bool) {
	up.config.Enabled = value
}
