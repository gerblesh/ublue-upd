package rpmostree_test

import (
	"testing"

	"github.com/ublue-os/uupd/drv/generic"
	"github.com/ublue-os/uupd/drv/rpmostree"
	appLogging "github.com/ublue-os/uupd/pkg/logging"
)

func InitBaseConfig() rpmostree.RpmOstreeUpdater {
	var initConfiguration = generic.UpdaterInitConfiguration{
		DryRun:      false,
		Ci:          false,
		Verbose:     false,
		Environment: nil,
		Logger:      appLogging.NewMuteLogger(),
	}
	driv, _ := rpmostree.RpmOstreeUpdater{}.New(initConfiguration)
	return driv
}

func TestProperSteps(t *testing.T) {
	rpmostreeUpdater := InitBaseConfig()
	rpmostreeUpdater.Config.Enabled = false

	if rpmostreeUpdater.Steps() != 0 {
		t.Fatalf("Expected no steps when module is disabled")
	}

	rpmostreeUpdater.Config.Enabled = true
	if rpmostreeUpdater.Steps() == 0 {
		t.Fatalf("Expected steps to be added")
	}
}
