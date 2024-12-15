package drv_test

import (
	"testing"

	"github.com/ublue-os/uupd/drv"
	appLogging "github.com/ublue-os/uupd/pkg/logging"
)

func InitBaseConfig() drv.SystemUpdater {
	var initConfiguration = drv.UpdaterInitConfiguration{
		DryRun:      false,
		Ci:          false,
		Verbose:     false,
		Environment: nil,
		Logger:      appLogging.NewMuteLogger(),
	}
	driv, _ := drv.SystemUpdater{}.New(initConfiguration)
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

func TestFallBack(t *testing.T) {
	var environment drv.EnvironmentMap = drv.EnvironmentMap{
		"TEST_FALLBACK_GOOD": "true",
	}
	if value := drv.EnvOrFallback(environment, "TEST_FALLBACK_GOOD", "FALSE"); value != "true" {
		t.Fatalf("Getting the proper value fails, %s", value)
	}
	if value := drv.EnvOrFallback(environment, "TEST_FALLBACK_BAD", "FALSE"); value != "FALSE" {
		t.Fatalf("Getting the fallback fails, %s", value)
	}
}
