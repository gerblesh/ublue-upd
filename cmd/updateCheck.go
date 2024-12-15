package cmd

import (
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/ublue-os/uupd/drv"
)

func UpdateCheck(cmd *cobra.Command, args []string) {
	var enableUpd bool = true

	initConfiguration := drv.UpdaterInitConfiguration{}.New()
	rpmOstreeUpdater, err := drv.RpmOstreeUpdater{}.New(*initConfiguration)
	if err != nil {
		enableUpd = false
	}

	systemUpdater, err := drv.SystemUpdater{}.New(*initConfiguration)
	if err != nil {
		enableUpd = false
	}

	isBootc, err := drv.BootcCompatible(systemUpdater.BinaryPath)
	if err != nil {
		isBootc = false
	}

	if !isBootc {
		slog.Debug("Using rpm-ostree fallback as system driver")
	}

	systemUpdater.Config.Enabled = isBootc && enableUpd
	rpmOstreeUpdater.Config.Enabled = !isBootc && enableUpd

	var mainSystemDriver drv.SystemUpdateDriver
	if !isBootc {
		slog.Debug("Using the rpm-ostree driver")
		mainSystemDriver = &rpmOstreeUpdater
	} else {
		slog.Debug("Using the bootc driver")
		mainSystemDriver = &systemUpdater
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
