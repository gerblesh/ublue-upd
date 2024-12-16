package generic_test

import (
	"log"
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

func TestProperEnvironment(t *testing.T) {
	valuesExpected := map[string]string{
		"SIGMA":   "true",
		"CHUD":    "false",
		"AMOGUS":  "sus",
		"NOTHING": "",
	}

	for key, value := range valuesExpected {
		testmap := generic.GetEnvironment([]string{key + "=" + value})
		valuegot, exists := testmap[key]
		if !exists {
			log.Fatalf("Could not get environment variable at all: %s", key)
		}
		if valuegot != value {
			log.Fatalf("Value gotten from variable was not expected: Got %s, Expected %s", valuegot, value)
		}
	}
}
