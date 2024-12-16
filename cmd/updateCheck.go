package cmd

import (
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/ublue-os/uupd/drv/generic"
	"github.com/ublue-os/uupd/drv/system"
)

func UpdateCheck(cmd *cobra.Command, args []string) {
	initConfiguration := generic.UpdaterInitConfiguration{}.New()
	initConfiguration.Ci = false
	initConfiguration.DryRun = false
	initConfiguration.Verbose = false

	mainSystemDriver, _, _, err := system.InitializeSystemDriver(*initConfiguration)
	if err != nil {
		slog.Error("Failed")
		return
	}

	updateAvailable, err := mainSystemDriver.Check()
	if err != nil {
		slog.Error("Failed checking for updates", slog.Any("error", err))
		return
	}
	if updateAvailable {
		slog.Info("Update Available")
	} else {
		slog.Info("No updates available")
	}
}
