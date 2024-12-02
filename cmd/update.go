package cmd

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/spf13/cobra"
	"github.com/ublue-os/uupd/checks"
	"github.com/ublue-os/uupd/drv"
	"github.com/ublue-os/uupd/lib"
)

type Failure struct {
	Err    error
	Output string
}

func Update(cmd *cobra.Command, args []string) {
	lock, err := lib.AcquireLock()
	if err != nil {
		log.Fatalf("%v, is uupd already running?", err)
	}
	defer lib.ReleaseLock(lock)
	systemDriver, err := drv.GetSystemUpdateDriver()
	if err != nil {
		log.Fatalf("Failed to get system update driver: %v", err)
	}
	dryRun, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		log.Fatalf("Failed to get dry-run flag: %v", err)
	}
	var outdated = !dryRun
	if !dryRun {
		outdated, err = systemDriver.ImageOutdated()
		if err != nil {
			log.Fatalf("Unable to determine if image is outdated: %v", err)
		}
	}
	if outdated {
		lib.Notify("System Warning", "There hasn't been an update in over a month. Consider rebooting or running updates manually")
		log.Printf("There hasn't been an update in over a month. Consider rebooting or running updates manually")
	}

	hwCheck, err := cmd.Flags().GetBool("hw-check")
	if err != nil {
		log.Fatalf("Failed to get hw-check flag: %v", err)
	}

	if hwCheck && !dryRun {
		err := checks.RunHwChecks()
		if err != nil {
			log.Fatalf("Hardware checks failed: %v", err)
		}
		log.Println("Hardware checks passed")
	}

	users, err := lib.ListUsers()
	if err != nil {
		log.Fatalf("Failed to list users")
	}

	// Check if system update is available
	var updateAvailable = !dryRun
	if !dryRun {
		log.Printf("Checking for system updates (%s)", systemDriver.Name)
		updateAvailable, err = systemDriver.UpdateAvailable()
		// ignore error on dry run
		if err != nil {
			log.Fatalf("Failed to check for image updates: %v", err)
		}
		log.Printf("System updates available: %t (%s)", updateAvailable, systemDriver.Name)
	}
	systemUpdate := 0
	if updateAvailable || dryRun {
		systemUpdate = 1
	}

	// Check if brew is installed
	brewUid, brewErr := drv.GetBrewUID()
	brewUpdate := 0
	if brewErr == nil || dryRun {
		brewUpdate = 1
	}

	totalSteps := brewUpdate + systemUpdate + 1 + len(users) + 1 + len(users) // system + Brew + Flatpak (users + root) + Distrobox (users + root)
	pw := lib.NewProgressWriter()
	pw.SetNumTrackersExpected(1)
	go pw.Render()
	// -1 because 0 index
	tracker := lib.NewIncrementTracker(&progress.Tracker{Message: "Updating", Units: progress.UnitsDefault, Total: int64(totalSteps - 1)}, totalSteps-1)
	pw.AppendTracker(tracker.Tracker)

	failures := make(map[string]Failure)

	if updateAvailable {
		tracker.IncrementSection()
		lib.ChangeTrackerMessageFancy(pw, tracker, "Updating System")
		out, err := systemDriver.Update()
		if err != nil {
			failures[systemDriver.Name] = Failure{
				err,
				string(out),
			}
			tracker.IncrementSectionError()
		} else {
			tracker.IncrementSection()
		}
	}

	if brewUpdate == 1 {
		lib.ChangeTrackerMessageFancy(pw, tracker, "Updating CLI apps (Brew)")
		if !dryRun {
			out, err := drv.BrewUpdate(brewUid)
			if err != nil {
				failures["Brew"] = Failure{
					err,
					string(out),
				}
				tracker.IncrementSectionError()
			} else {
				tracker.IncrementSection()
			}
		}
	}

	// Run flatpak updates
	lib.ChangeTrackerMessageFancy(pw, tracker, "Updating System Apps (Flatpak)")
	if dryRun {
		tracker.IncrementSection()
	} else {
		flatpakCmd := exec.Command("/usr/bin/flatpak", "update", "-y")
		out, err := flatpakCmd.CombinedOutput()
		if err != nil {
			failures["Flatpak"] = Failure{
				err,
				string(out),
			}
			tracker.IncrementSectionError()
		} else {
			tracker.IncrementSection()
		}
	}
	for _, user := range users {
		lib.ChangeTrackerMessageFancy(pw, tracker, fmt.Sprintf("Updating Apps for User: %s (Flatpak)", user.Name))
		if dryRun {
			tracker.IncrementSection()
		} else {
			out, err := lib.RunUID(user.UID, []string{"/usr/bin/flatpak", "update", "-y"}, nil)
			if err != nil {
				failures[fmt.Sprintf("Flatpak User: %s", user.Name)] = Failure{
					err,
					string(out),
				}
				tracker.IncrementSectionError()
			} else {
				tracker.IncrementSection()
			}

		}
	}

	// Run distrobox updates
	lib.ChangeTrackerMessageFancy(pw, tracker, "Updating System Distroboxes")
	if dryRun {
		tracker.IncrementSection()
	} else {
		// distrobox doesn't support sudo, run with systemd-run
		out, err := lib.RunUID(0, []string{"/usr/bin/distrobox", "upgrade", "-a"}, nil)
		if err != nil {
			failures["Distrobox"] = Failure{
				err,
				string(out),
			}
			tracker.IncrementSectionError()
		} else {
			tracker.IncrementSection()
		}
	}

	for _, user := range users {
		lib.ChangeTrackerMessageFancy(pw, tracker, fmt.Sprintf("Updating Distroboxes for User: %s", user.Name))
		if dryRun {
			tracker.IncrementSection()
		} else {
			out, err := lib.RunUID(user.UID, []string{"/usr/bin/distrobox", "upgrade", "-a"}, nil)
			if err != nil {
				failures[fmt.Sprintf("Distrobox User: %s", user.Name)] = Failure{
					err,
					string(out),
				}
				tracker.IncrementSectionError()
			} else {
				tracker.IncrementSection()
			}
		}
	}

	if len(failures) > 0 && !dryRun {
		pw.SetAutoStop(false)
		pw.Stop()
		failedSystemsList := make([]string, 0, len(failures))
		for systemName := range failures {
			failedSystemsList = append(failedSystemsList, systemName)
		}
		failedSystemsStr := strings.Join(failedSystemsList, ", ")
		lib.Notify("Updates failed", fmt.Sprintf("uupd failed to update: %s, consider seeing logs with `journalctl -exu uupd.service`", failedSystemsStr))
		log.Printf("Updates Completed with Failures:")
		for name, fail := range failures {
			indentedOutput := "\t |  "
			lines := strings.Split(fail.Output, "\n")
			for i, line := range lines {
				if i > 0 {
					indentedOutput += "\n\t |  "
				}
				indentedOutput += line
			}
			log.Printf("---> %s \n\t | Failure error: %v \n\t | Command Output: \n%s", name, fail.Err, indentedOutput)
		}
		return
	}
	log.Printf("Updates Completed")
}
