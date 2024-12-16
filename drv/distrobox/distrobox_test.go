package distrobox_test

import (
	"log"
	"testing"

	"github.com/ublue-os/uupd/drv/distrobox"
	"github.com/ublue-os/uupd/drv/generic"
	appLogging "github.com/ublue-os/uupd/pkg/logging"
	"github.com/ublue-os/uupd/pkg/session"
)

func InitBaseConfig() distrobox.DistroboxUpdater {
	var initConfiguration = generic.UpdaterInitConfiguration{
		DryRun:      false,
		Ci:          false,
		Verbose:     false,
		Environment: nil,
		Logger:      appLogging.NewMuteLogger(),
	}
	driv, _ := distrobox.DistroboxUpdater{}.New(initConfiguration)
	return driv
}

func TestProperSteps(t *testing.T) {
	updater := InitBaseConfig()
	updater.Config.Enabled = false

	if updater.Steps() != 0 {
		t.Fatalf("Expected no steps when module is disabled")
	}

	updater.Config.Enabled = true
	if updater.Steps() == 0 {
		t.Fatalf("Expected steps to be added")
	}
}

func TestProperUserSteps(t *testing.T) {
	updater := InitBaseConfig()

	mockUser := []session.User{
		{UID: 0, Name: "root"},
		{UID: 1, Name: "roote"},
		{UID: 2, Name: "rooto"},
	}
	updater.SetUsers(mockUser)

	if reported := updater.Steps(); reported != 1+len(mockUser) {
		log.Fatalf("Incorrect number of steps for users: %d", reported)
	}
	updater.Config.Enabled = false
	if reported := updater.Steps(); reported != 0 {
		log.Fatalf("Incorrect number of steps for users: %d", reported)
	}
}
