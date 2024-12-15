package percent

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/ublue-os/uupd/pkg/session"
)

type Incrementer struct {
	doneIncrements int
	MaxIncrements  int
}

type IncrementTracker struct {
	Tracker     *progress.Tracker
	incrementer *Incrementer
}

var CuteColors = progress.StyleColors{
	Message: text.Colors{text.FgWhite},
	Error:   text.Colors{text.FgRed},
	Percent: text.Colors{text.FgHiBlue},
	Pinned:  text.Colors{text.BgHiBlack, text.FgWhite, text.Bold},
	Stats:   text.Colors{text.FgHiBlack},
	Time:    text.Colors{text.FgBlue},
	Tracker: text.Colors{text.FgHiBlue},
	Value:   text.Colors{text.FgBlue},
	Speed:   text.Colors{text.FgBlue},
}

func NewProgressWriter() progress.Writer {
	pw := progress.NewWriter()

	pw.SetTrackerLength(25)
	pw.Style().Visibility.TrackerOverall = true
	pw.Style().Visibility.Time = true
	pw.Style().Visibility.Tracker = true
	pw.Style().Visibility.Value = true
	pw.SetMessageLength(32)
	pw.SetSortBy(progress.SortByPercentDsc)
	pw.SetStyle(progress.StyleBlocks)
	pw.SetTrackerPosition(progress.PositionRight)
	pw.SetUpdateFrequency(time.Millisecond * 100)
	pw.Style().Options.PercentFormat = "%4.1f%%"

	pw.Style().Colors = CuteColors

	var targetUser int
	baseUser, exists := os.LookupEnv("SUDO_UID")
	if !exists || baseUser == "" {
		targetUser = 0
	} else {
		var err error
		targetUser, err = strconv.Atoi(baseUser)
		if err != nil {
			slog.Error("Failed parsing provided user as UID", slog.String("user_value", baseUser))
			return pw
		}
	}

	if targetUser != 0 {
		var accentColorSet progress.StyleColors
		// Get accent color: https://flatpak.github.io/xdg-desktop-portal/docs/doc-org.freedesktop.portal.Settings.html
		cli := []string{"busctl", "--user", "--json=short", "call", "org.freedesktop.portal.Desktop", "/org/freedesktop/portal/desktop", "org.freedesktop.portal.Settings", "ReadOne", "ss", "org.freedesktop.appearance", "accent-color"}
		out, err := session.RunUID(nil, slog.LevelDebug, targetUser, cli, nil)
		if err != nil {
			return pw
		}
		var accent Accent
		err = json.Unmarshal(out, &accent)
		if err != nil {
			return pw
		}

		raw_color := accent.Data[0].Data

		highlightColor, lowColor := findClosestColor(raw_color)

		validHighlightColor := text.Colors{highlightColor}
		validLowColor := text.Colors{lowColor}

		accentColorSet.Percent = validHighlightColor
		accentColorSet.Tracker = validHighlightColor
		accentColorSet.Time = validLowColor
		accentColorSet.Value = validLowColor
		accentColorSet.Speed = validLowColor
		pw.Style().Colors = accentColorSet
	}
	return pw
}

func NewIncrementTracker(tracker *progress.Tracker, max_increments int) *IncrementTracker {
	return &IncrementTracker{
		Tracker:     tracker,
		incrementer: &Incrementer{MaxIncrements: max_increments},
	}
}

type TrackerMessage struct {
	Title       string
	Description string
}

func ChangeTrackerMessageFancy(writer progress.Writer, tracker *IncrementTracker, progress bool, message TrackerMessage) {
	if !progress {
		slog.Info("Updating",
			slog.String("title", message.Title),
			slog.String("description", message.Description),
			slog.Int64("progress", tracker.Tracker.Value()),
			slog.Int64("total", tracker.Tracker.Total),
		)
		return
	}
	percentage := math.Round((float64(tracker.Tracker.Value()) / float64(tracker.Tracker.Total)) * 100)
	fmt.Printf("\033]9;4;1;%d\a", int(percentage))
	finalMessage := fmt.Sprintf("Updating %s (%s)", message.Description, message.Title)
	writer.SetMessageLength(len(finalMessage))
	tracker.Tracker.UpdateMessage(finalMessage)
}

func (it *IncrementTracker) IncrementSection(err error) {
	var increment_step float64
	if it.incrementer.doneIncrements == 0 {
		increment_step = 1
	} else {
		increment_step = float64(it.Tracker.Total / int64(it.incrementer.MaxIncrements))
	}
	if err == nil {
		it.Tracker.Increment(int64(increment_step))
	} else {
		it.Tracker.IncrementWithError(int64(increment_step))
	}
	it.incrementer.doneIncrements++
}

func (it *IncrementTracker) CurrentStep() int {
	return it.incrementer.doneIncrements
}

func ResetOscProgress() {
	// OSC escape sequence to reset all previous OSC progress hints to 0%.
	// Documentation is on https://conemu.github.io/en/AnsiEscapeCodes.html#ConEmu_specific_OSC
	print("\033]9;4;0\a")
}
