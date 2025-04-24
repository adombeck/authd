package localentries

import (
	"errors"
	"sync/atomic"

	"github.com/ubuntu/authd/internal/testsdetection"
)

var (
	writeLockImpl   = writeLock
	writeUnlockImpl = writeUnlock

	overrideLocked atomic.Bool
)

var (
	// ErrLock is the error when locking the database fails.
	ErrLock = errors.New("failed to lock the shadow password database")

	// ErrUnlock is the error when unlocking the database fails.
	ErrUnlock = errors.New("failed to unlock the shadow password database")
)

// WriteLock locks for writing the the local user entries database by using
// the standard libc lckpwdf() function.
// While the database is locked read operations can happen, but no other process
// is allowed to write.
// Note that this call will block all the other processes trying to access the
// database in write mode, while it will return an error if called while the
// lock is already hold by this process.
func WriteLock() error {
	return writeLockImpl()
}

// WriteUnlock unlocks for writing the local user entries database by using
// the standard libc ulckpwdf() function.
// As soon as this function is called all the other waiting processes will be
// allowed to take the lock.
func WriteUnlock() error {
	return writeUnlockImpl()
}

// OverrideLocking is a function to override the locking functions
// for testing purposes.
// It simulates the real behavior but without actual file locking.
func OverrideLocking() {
	testsdetection.MustBeTesting()

	writeLockImpl = func() error {
		if !overrideLocked.CompareAndSwap(false, true) {
			return ErrLock
		}
		return nil
	}

	writeUnlockImpl = func() error {
		if !overrideLocked.CompareAndSwap(true, false) {
			return ErrUnlock
		}
		return nil
	}
}

// RestoreLocking restores the locking overridden done by [OverrideLocking].
func RestoreLocking() {
	testsdetection.MustBeTesting()

	writeLockImpl = writeLock
	writeUnlockImpl = writeUnlock
}
