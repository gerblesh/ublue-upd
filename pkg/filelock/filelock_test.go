package filelock_test

import (
	"testing"

	"github.com/ublue-os/uupd/pkg/filelock"
)

const LockForTest = "/tmp/uupd-sigmatest.lock"

var defaultTimeout = filelock.TimeoutConfig{1}

func TestFullLock(t *testing.T) {
	file, err := filelock.OpenLockfile(LockForTest)
	if err != nil {
		t.Fatalf("Failed even opening the file, %v", err)
	}
	defer file.Close()
	err = filelock.AcquireLock(file, defaultTimeout)
	if err != nil {
		t.Fatalf("Failed acquiring lock file, %v", err)
	}
	err = filelock.ReleaseLock(file)
	if err != nil {
		t.Fatal("Failed releasing lock file")
	}
}

func TestLockAcquired(t *testing.T) {
	file1, err := filelock.OpenLockfile(LockForTest)
	if err != nil {
		t.Fatalf("Failed even opening the file, %v", err)
	}
	defer file1.Close()
	err = filelock.AcquireLock(file1, defaultTimeout)
	if err != nil {
		t.Fatalf("Failed acquiring lock file, %v", err)
	}
	file2, err := filelock.OpenLockfile(LockForTest)
	if err != nil {
		t.Fatalf("Failed even opening the file, %v", err)
	}
	defer file2.Close()
	err = filelock.AcquireLock(file2, defaultTimeout)
	if err == nil {
		t.Fatalf("Expected failing to lock file, %v", err)
	}
}
