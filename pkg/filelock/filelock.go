package filelock

import (
	"errors"
	"os"
	"syscall"
	"time"
)

func IsFileLocked(file *os.File) bool {
	lock := syscall.Flock_t{}
	err := syscall.FcntlFlock(file.Fd(), syscall.F_GETLK, &lock)
	if err != nil {
		return false
	}
	return lock.Type != syscall.F_UNLCK
}

func GetDefaultLockfile() string {
	return "/run/uupd.lock"
}

func OpenLockfile(filepath string) (*os.File, error) {
	return os.OpenFile(filepath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
}

type TimeoutConfig struct {
	Tries int
}

func AcquireLock(file *os.File, timeout TimeoutConfig) error {
	maxTries := timeout.Tries
	tries := 0
	fileLocked := false

	for tries < maxTries {
		err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if fileLocked = err == nil; fileLocked {
			break
		}

		tries += 1
		time.Sleep(1 * time.Second)
	}

	if !fileLocked {
		return errors.New("Could not acquire lock to file")
	}

	return nil
}

func ReleaseLock(file *os.File) error {
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_UN); err != nil {
		return err
	}
	return syscall.Close(int(file.Fd()))
}
