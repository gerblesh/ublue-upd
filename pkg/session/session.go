package session

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os/exec"

	"github.com/godbus/dbus/v5"
)

type User struct {
	UID  int
	Name string
}

// Runs any specified Command while logging it to the logger
// Made to work just like (Command).CombinedOutput()
func RunLog(logger *slog.Logger, level slog.Level, command *exec.Cmd) ([]byte, error) {
	if logger == nil {
		return command.CombinedOutput()
	}

	stdout, _ := command.StdoutPipe()
	stderr, _ := command.StderrPipe()
	multiReader := io.MultiReader(stdout, stderr)

	command.Start()
	scanner := bufio.NewScanner(multiReader)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		logger.With(slog.Bool("subcommand", true)).Log(nil, level, scanner.Text())
	}
	command.Wait()

	return scanner.Bytes(), scanner.Err()
}

func RunUID(logger *slog.Logger, level slog.Level, uid int, command []string, env map[string]string) ([]byte, error) {
	// Just fork systemd-run, we don't need to rewrite systemd-run with dbus
	cmdArgs := []string{
		"/usr/bin/systemd-run",
		"--machine",
		fmt.Sprintf("%d@", uid),
		"--pipe",
		"--quiet",
	}
	if uid != 0 {
		cmdArgs = append(cmdArgs, "--user")
	}
	cmdArgs = append(cmdArgs, command...)

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)

	return RunLog(logger, level, cmd)
}

func ListUsers() ([]User, error) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return []User{}, fmt.Errorf("failed to connect to system bus: %v", err)
	}
	defer conn.Close()

	var resp [][]dbus.Variant
	object := conn.Object("org.freedesktop.login1", "/org/freedesktop/login1")
	err = object.Call("org.freedesktop.login1.Manager.ListUsers", 0).Store(&resp)
	if err != nil {
		return []User{}, err
	}

	var users []User
	for _, data := range resp {
		if len(data) < 2 {
			return []User{}, fmt.Errorf("Malformed dbus response")
		}
		uidVariant := data[0]
		nameVariant := data[1]

		uid, ok := uidVariant.Value().(uint32)
		if !ok {
			return []User{}, fmt.Errorf("invalid UID type, expected uint32")
		}

		name, ok := nameVariant.Value().(string)
		if !ok {
			return []User{}, fmt.Errorf("invalid Name type, expected string")
		}

		users = append(users, User{
			UID:  int(uid),
			Name: name,
		})
	}
	return users, nil
}

func Notify(summary string, body string) error {
	users, err := ListUsers()
	if err != nil {
		return err
	}
	for _, user := range users {
		// we don't care if these exit
		_, _ = RunUID(slog.Default(), slog.LevelDebug, user.UID, []string{"/usr/bin/notify-send", "--app-name", "uupd", summary, body}, nil)
	}
	return nil
}
