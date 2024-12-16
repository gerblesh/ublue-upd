package system_test

import (
	"testing"

	"github.com/ublue-os/uupd/drv/generic"
	"github.com/ublue-os/uupd/drv/system"
	appLogging "github.com/ublue-os/uupd/pkg/logging"
)

func InitBaseConfig() system.SystemUpdater {
	var initConfiguration = generic.UpdaterInitConfiguration{
		DryRun:      false,
		Ci:          false,
		Verbose:     false,
		Environment: nil,
		Logger:      appLogging.NewMuteLogger(),
	}
	driv, _ := system.SystemUpdater{}.New(initConfiguration)
	return driv
}

func TestProperSteps(t *testing.T) {
	systemUpdater := InitBaseConfig()
	systemUpdater.Config.Enabled = false

	if systemUpdater.Steps() != 0 {
		t.Fatalf("Expected no steps when module is disabled")
	}

	systemUpdater.Config.Enabled = true
	if systemUpdater.Steps() == 0 {
		t.Fatalf("Expected steps to be added")
	}
}
