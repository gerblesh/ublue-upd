package drv

import (
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/ublue-os/uupd/lib"
)

type UpdaterInitConfiguration struct {
	DryRun      bool
	Ci          bool
	Verbose     bool
	Environment map[string]string
}

func GetEnvironment(data []string, getkeyval func(item string) (key, val string)) map[string]string {
	items := make(map[string]string)
	for _, item := range data {
		key, val := getkeyval(item)
		items[key] = val
	}
	return items
}

func (up UpdaterInitConfiguration) New() *UpdaterInitConfiguration {
	up.DryRun = false
	up.Ci = false
	up.Environment = GetEnvironment(os.Environ(), func(item string) (key, val string) {
		splits := strings.Split(item, "=")
		key = splits[0]
		val = splits[1]
		return
	})

	return &up
}

type CommandOutput struct {
	Stdout  string
	Failure bool
	Stderr  error
	Context string
}

func (output CommandOutput) New(out []byte, err error) *CommandOutput {
	return &CommandOutput{
		Context: "",
		Failure: err != nil,
		Stderr:  nil,
		Stdout:  string(out),
	}
}

func (out *CommandOutput) SetFailureContext(context string) {
	out.Failure = true
	out.Context = context
}

type DriverConfiguration struct {
	Title           string
	Description     string
	Enabled         bool
	MultiUser       bool
	DryRun          bool
	UserDescription *string
}

type TrackerConfiguration struct {
	Tracker  *lib.IncrementTracker
	Writer   *progress.Writer
	Progress bool
}

type UpdateDriver interface {
	Steps() int
	Check() (*[]CommandOutput, error)
	Update() (*[]CommandOutput, error)
	New(config UpdaterInitConfiguration) (*UpdateDriver, error)
}

type MultiUserUpdateDriver interface {
	*UpdateDriver
	SetUsers(users []lib.User)
}
