package generic_test

import (
	"testing"

	"github.com/ublue-os/uupd/drv/generic"
)

func TestFallBack(t *testing.T) {
	var environment generic.EnvironmentMap = generic.EnvironmentMap{
		"TEST_FALLBACK_GOOD": "true",
	}
	if value := generic.EnvOrFallback(environment, "TEST_FALLBACK_GOOD", "FALSE"); value != "true" {
		t.Fatalf("Getting the proper value fails, %s", value)
	}
	if value := generic.EnvOrFallback(environment, "TEST_FALLBACK_BAD", "FALSE"); value != "FALSE" {
		t.Fatalf("Getting the fallback fails, %s", value)
	}
}
