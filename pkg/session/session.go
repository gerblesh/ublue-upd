package session

import (
	"bufio"
	"context"
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

	if err := command.Wait(); err != nil {
		logger.Warn("Error occoured starting external command", slog.Any("error", err))
	}
	scanner := bufio.NewScanner(multiReader)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		logger.With(slog.Bool("subcommand", true)).Log(context.TODO(), level, scanner.Text())
	}
	if err := command.Wait(); err != nil {
		logger.Warn("Error occoured while waiting for external command", slog.Any("error", err))
	}

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

func ParseUserFromVariant(uidVariant dbus.Variant, nameVariant dbus.Variant) (User, error) {
	uid, ok := uidVariant.Value().(uint32)
	if !ok {
		return User{}, fmt.Errorf("invalid UID type, expected uint32")
	}

	name, ok := nameVariant.Value().(string)
	if !ok {
		return User{}, fmt.Errorf("invalid Name type, expected string")
	}

	return User{
		UID:  int(uid),
		Name: name,
	}, nil
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
		parsed, err := ParseUserFromVariant(data[0], data[1])
		if err != nil {
			return nil, err
		}

		users = append(users, parsed)
	}
	return users, nil
}

func Notify(users []User, summary string, body string) error {
	for _, user := range users {
		// we don't care if these exit
		_, _ = RunUID(nil, slog.LevelDebug, user.UID, []string{"/usr/bin/notify-send", "--app-name", "uupd", summary, body}, nil)
	}
	return nil
}
