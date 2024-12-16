package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/spf13/cobra"
	"github.com/ublue-os/uupd/checks"
	"github.com/ublue-os/uupd/drv/brew"
	"github.com/ublue-os/uupd/drv/distrobox"
	"github.com/ublue-os/uupd/drv/flatpak"
	drv "github.com/ublue-os/uupd/drv/generic"
	"github.com/ublue-os/uupd/drv/system"

	"github.com/ublue-os/uupd/pkg/filelock"
	"github.com/ublue-os/uupd/pkg/percent"
	"github.com/ublue-os/uupd/pkg/session"
)

func Update(cmd *cobra.Command, args []string) {
	lockfile, err := filelock.OpenLockfile(filelock.GetDefaultLockfile())
	if err != nil {
		slog.Error("Failed creating and opening lockfile. Is uupd already running?", slog.Any("error", err))
		return
	}
	defer func(lockfile *os.File) {
		err := filelock.ReleaseLock(lockfile)
		if err != nil {
			slog.Error("Failed releasing lock", slog.Any("error", err))
		}
	}(lockfile)
	if err := filelock.AcquireLock(lockfile, filelock.TimeoutConfig{Tries: 5}); err != nil {
		slog.Error(fmt.Sprintf("%v, is uupd already running?", err))
		return
	}

	hwCheck, err := cmd.Flags().GetBool("hw-check")
	if err != nil {
		slog.Error("Failed to get hw-check flag", "error", err)
		return
	}
	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		slog.Error("Failed to get dry-run flag", "error", err)
		return
	}
	verboseRun, err := cmd.Flags().GetBool("verbose")
	if err != nil {
		slog.Error("Failed to get verbose flag", "error", err)
		return
	}

	if hwCheck {
		err := checks.RunHwChecks()
		if err != nil {
			slog.Error("Hardware checks failed", "error", err)
			return
		}
		slog.Info("Hardware checks passed")
	}

	users, err := session.ListUsers()
	if err != nil {
		slog.Error("Failed to list users", "users", users)
		return
	}

	initConfiguration := drv.UpdaterInitConfiguration{}.New()
	_, exists := os.LookupEnv("CI")
	initConfiguration.Ci = exists
	initConfiguration.DryRun = dryRun
	initConfiguration.Verbose = verboseRun

	brewUpdater, err := brew.BrewUpdater{}.New(*initConfiguration)
	brewUpdater.Config.Enabled = err == nil

	flatpakUpdater, err := flatpak.FlatpakUpdater{}.New(*initConfiguration)
	flatpakUpdater.Config.Enabled = err == nil
	flatpakUpdater.SetUsers(users)

	distroboxUpdater, err := distrobox.DistroboxUpdater{}.New(*initConfiguration)
	distroboxUpdater.Config.Enabled = err == nil
	distroboxUpdater.SetUsers(users)

	mainSystemDriver, mainSystemDriverConfig, _, err := system.InitializeSystemDriver(*initConfiguration)

	enableUpd, err := mainSystemDriver.Check()
	if err != nil {
		slog.Error("Failed checking for updates")
	}
	mainSystemDriverConfig.Enabled = mainSystemDriverConfig.Enabled && enableUpd

	slog.Debug("System Updater module status", slog.Bool("enabled", mainSystemDriverConfig.Enabled))

	totalSteps := brewUpdater.Steps() + flatpakUpdater.Steps() + distroboxUpdater.Steps()
	if mainSystemDriverConfig.Enabled {
		totalSteps += mainSystemDriver.Steps()
	}
	pw := percent.NewProgressWriter()
	pw.SetNumTrackersExpected(1)
	pw.SetAutoStop(false)

	progressEnabled, err := cmd.Flags().GetBool("no-progress")
	if err != nil {
		slog.Error("Failed to get no-progress flag", "error", err)
		return
	}
	// Move this to its actual boolean value (~no-progress)
	progressEnabled = !progressEnabled

	if progressEnabled {
		go pw.Render()
		percent.ResetOscProgress()
	}

	// -1 because 0 index
	tracker := percent.NewIncrementTracker(&progress.Tracker{Message: "Updating", Units: progress.UnitsDefault, Total: int64(totalSteps - 1)}, totalSteps-1)
	pw.AppendTracker(tracker.Tracker)

	var trackerConfig = &drv.TrackerConfiguration{
		Tracker:  tracker,
		Writer:   &pw,
		Progress: progressEnabled,
	}
	flatpakUpdater.Tracker = trackerConfig
	distroboxUpdater.Tracker = trackerConfig

	var outputs = []drv.CommandOutput{}

	systemOutdated, err := mainSystemDriver.Outdated()

	if err != nil {
		slog.Error("Failed checking if system is out of date")
	}

	if systemOutdated {
		const OUTDATED_WARNING = "There hasn't been an update in over a month. Consider rebooting or running updates manually"
		err := session.Notify(users, "System Warning", OUTDATED_WARNING)
		if err != nil {
			slog.Error("Failed showing warning notification")
		}
		slog.Warn(OUTDATED_WARNING)
	}

	// This section is ugly but we cant really do much about it.
	// Using interfaces doesn't preserve the "Config" struct state and I dont know any other way to make this work without cursed workarounds.

	if mainSystemDriverConfig.Enabled {
		slog.Debug(fmt.Sprintf("%s module", mainSystemDriverConfig.Title), slog.String("module_name", mainSystemDriverConfig.Title), slog.Any("module_configuration", mainSystemDriverConfig))
		percent.ChangeTrackerMessageFancy(pw, tracker, progressEnabled, percent.TrackerMessage{Title: mainSystemDriverConfig.Title, Description: mainSystemDriverConfig.Description})
		var out *[]drv.CommandOutput
		out, err = mainSystemDriver.Update()
		outputs = append(outputs, *out...)
		tracker.IncrementSection(err)
	}

	if brewUpdater.Config.Enabled {
		slog.Debug(fmt.Sprintf("%s module", brewUpdater.Config.Title), slog.String("module_name", brewUpdater.Config.Title), slog.Any("module_configuration", brewUpdater.Config))
		percent.ChangeTrackerMessageFancy(pw, tracker, progressEnabled, percent.TrackerMessage{Title: brewUpdater.Config.Title, Description: brewUpdater.Config.Description})
		var out *[]drv.CommandOutput
		out, err = brewUpdater.Update()
		outputs = append(outputs, *out...)
		tracker.IncrementSection(err)
	}

	if flatpakUpdater.Config.Enabled {
		slog.Debug(fmt.Sprintf("%s module", flatpakUpdater.Config.Title), slog.String("module_name", flatpakUpdater.Config.Title), slog.Any("module_configuration", flatpakUpdater.Config))
		var out *[]drv.CommandOutput
		out, err = flatpakUpdater.Update()
		outputs = append(outputs, *out...)
		tracker.IncrementSection(err)
	}

	if distroboxUpdater.Config.Enabled {
		slog.Debug(fmt.Sprintf("%s module", distroboxUpdater.Config.Title), slog.String("module_name", distroboxUpdater.Config.Title), slog.Any("module_configuration", distroboxUpdater.Config))
		var out *[]drv.CommandOutput
		out, err = distroboxUpdater.Update()
		outputs = append(outputs, *out...)
		tracker.IncrementSection(err)
	}

	if progressEnabled {
		pw.Stop()
		percent.ResetOscProgress()
	}
	if verboseRun {
		slog.Info("Verbose run requested")

		for _, output := range outputs {
			slog.Info(output.Context, slog.Any("output", output))
		}

		return
	}

	var failures = []drv.CommandOutput{}
	for _, output := range outputs {
		if output.Failure {
			failures = append(failures, output)
		}
	}

	if len(failures) > 0 {
		slog.Warn("Exited with failed updates.")

		for _, output := range failures {
			slog.Info(output.Context, slog.Any("output", output))
		}

		return
	}

	slog.Info("Updates Completed Successfully")
}
