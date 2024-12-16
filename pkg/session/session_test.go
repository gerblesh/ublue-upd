package session_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/godbus/dbus/v5"
	"github.com/ublue-os/uupd/pkg/session"
)

func TestUserParsingInvalidUID(t *testing.T) {
	t.Parallel()
	testVariants := []dbus.Variant{
		dbus.MakeVariant(0.3),
		dbus.MakeVariant(-1),
		dbus.MakeVariant(math.MaxInt),
		dbus.MakeVariant(math.MinInt),
	}

	userName := dbus.MakeVariant("root")
	for _, uidVariant := range testVariants {
		t.Run(fmt.Sprintf("variant: %v", uidVariant.Value()), func(t *testing.T) {
			t.Parallel()
			_, err := session.ParseUserFromVariant(uidVariant, userName)
			if err == nil {
				t.Fatalf("Parser accepted invalid input: %v %v", uidVariant, userName)
			}
		})
	}
}

func TestUserParsingInvalidName(t *testing.T) {
	t.Parallel()
	testVariants := []dbus.Variant{
		dbus.MakeVariant(0.3),
		dbus.MakeVariant(-1),
		dbus.MakeVariant(math.MaxInt),
		dbus.MakeVariant(math.MinInt),
	}

	uidVariant := dbus.MakeVariant(uint32(0))
	for _, nameVariant := range testVariants {
		t.Run(fmt.Sprintf("variant: %v", uidVariant.Value()), func(t *testing.T) {
			t.Parallel()
			_, err := session.ParseUserFromVariant(uidVariant, nameVariant)
			if err == nil {
				t.Fatalf("Parser accepted invalid input: %v", err)
			}
		})
	}
}

func TestUserParsingValidUser(t *testing.T) {
	t.Parallel()
	testVariants := []struct {
		UidVariant  dbus.Variant
		NameVariant dbus.Variant
	}{
		{dbus.MakeVariant(uint32(0)), dbus.MakeVariant("root")},
		{dbus.MakeVariant(uint32(10)), dbus.MakeVariant("bob")},
		{dbus.MakeVariant(uint32(20)), dbus.MakeVariant("beatryz")},
		{dbus.MakeVariant(uint32(math.MaxUint16)), dbus.MakeVariant("zorg_the_destroyer")},
	}

	for _, variant := range testVariants {
		t.Run(fmt.Sprintf("variant: %v", variant.NameVariant.Value()), func(t *testing.T) {
			t.Parallel()
			_, err := session.ParseUserFromVariant(variant.UidVariant, variant.NameVariant)
			if err != nil {
				t.Fatalf("Parser rejected valid input: %v", err)
			}
		})
	}
}
