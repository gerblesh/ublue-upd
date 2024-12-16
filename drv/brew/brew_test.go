package brew_test

import (
	"testing"

	"github.com/ublue-os/uupd/drv/brew"
	"github.com/ublue-os/uupd/drv/generic"
	appLogging "github.com/ublue-os/uupd/pkg/logging"
)

func InitBaseConfig() brew.BrewUpdater {
	var initConfiguration = generic.UpdaterInitConfiguration{
		DryRun:      false,
		Ci:          false,
		Verbose:     false,
		Environment: nil,
		Logger:      appLogging.NewMuteLogger(),
	}
	driv, _ := brew.BrewUpdater{}.New(initConfiguration)
	return driv
}

func TestProperSteps(t *testing.T) {
	brewUpdater := InitBaseConfig()
	brewUpdater.Config.Enabled = false

	if brewUpdater.Steps() != 0 {
		t.Fatalf("Expected no steps when module is disabled")
	}

	brewUpdater.Config.Enabled = true
	if brewUpdater.Steps() == 0 {
		t.Fatalf("Expected steps to be added")
	}
}
