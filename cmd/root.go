package cmd

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/user"
	"path"
	"path/filepath"

	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
	appLogging "github.com/ublue-os/uupd/pkg/logging"
	"golang.org/x/term"
)

func assertRoot(cmd *cobra.Command, args []string) {
	currentUser, err := user.Current()

	if err != nil {
		log.Fatalf("Error fetching current user: %v", err)
	}
	if currentUser.Uid != "0" {
		log.Fatalf("uupd needs to be invoked as root.")
	}
}

var (
	rootCmd = &cobra.Command{
		Use:               "uupd",
		Short:             "uupd (Universal Update) is the successor to ublue-update, built for bootc",
		PersistentPreRunE: initLogging,
		PreRun:            assertRoot,
		Run:               Update,
	}

	waitCmd = &cobra.Command{
		Use:    "wait",
		Short:  "Waits for ostree sysroot to unlock",
		PreRun: assertRoot,
		Run:    Wait,
	}

	updateCheckCmd = &cobra.Command{
		Use:    "update-check",
		Short:  "Check for updates to the booted image",
		PreRun: assertRoot,
		Run:    UpdateCheck,
	}

	hardwareCheckCmd = &cobra.Command{
		Use:    "hw-check",
		Short:  "Run hardware checks",
		PreRun: assertRoot,
		Run:    HwCheck,
	}

	imageOutdatedCmd = &cobra.Command{
		Use:    "is-img-outdated",
		Short:  "Print 'true' or 'false' based on if the current booted image is over 1 month old",
		PreRun: assertRoot,
		Run:    ImageOutdated,
	}

	fLogFile   string
	fLogLevel  string
	fNoLogging bool
	fLogJson   bool
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func initLogging(cmd *cobra.Command, args []string) error {
	var logWriter *os.File = os.Stdout
	if fLogFile != "-" {
		abs, err := filepath.Abs(path.Clean(fLogFile))
		if err != nil {
			return err
		}
		logWriter, err = os.OpenFile(abs, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0755)
		if err != nil {
			return err
		}
	}

	logLevel, err := appLogging.StrToLogLevel(fLogLevel)
	if err != nil {
		return err
	}

	main_app_logger := slog.New(appLogging.SetupAppLogger(logWriter, logLevel, fLogFile != "-" || fLogJson))

	if fNoLogging {
		slog.SetDefault(appLogging.NewMuteLogger())
	} else {
		slog.SetDefault(main_app_logger)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(waitCmd)
	rootCmd.AddCommand(updateCheckCmd)
	rootCmd.AddCommand(hardwareCheckCmd)
	rootCmd.AddCommand(imageOutdatedCmd)
	rootCmd.Flags().BoolP("hw-check", "c", false, "Run hardware check before running updates")
	rootCmd.Flags().BoolP("dry-run", "n", false, "Do a dry run")
	rootCmd.Flags().BoolP("verbose", "v", false, "Display command outputs after run")
	rootCmd.Flags().Bool("ci", false, "Makes some modifications to behavior if is running in CI")

	rootCmd.PersistentFlags().BoolVar(&fLogJson, "json", false, "Print logs as json (used for testing)")
	rootCmd.PersistentFlags().StringVar(&fLogFile, "log-file", "-", "File where user-facing logs will be written to")
	rootCmd.PersistentFlags().StringVar(&fLogLevel, "log-level", "info", "Log level for user-facing logs")
	rootCmd.PersistentFlags().BoolVar(&fNoLogging, "quiet", false, "Make logs quiet")

	interactiveProgress := true
	if fLogFile != "-" {
		interactiveProgress = false
	}
	isTerminal := term.IsTerminal(int(os.Stdout.Fd()))
	if !isTerminal {
		interactiveProgress = false
	}
	if !text.ANSICodesSupported {
		interactiveProgress = false
		text.DisableColors()
	}

	rootCmd.Flags().BoolP("no-progress", "p", !interactiveProgress, "Do not show progress bars")
}
