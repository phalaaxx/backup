package main

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"
)

// ETimeout is a lock timeout error
var ETimeout = errors.New("timeout")

// DoLock acquires a file lock to prevent double runs and then starts r.Run
func DoLock(r *RsyncOptions) (err error) {
	// generate lock file name
	LockFile := fmt.Sprintf("%s/backup.lock", r.GetTarget())
	// open or create lock file if it does not yet exist
	var File *os.File
	if File, err = os.Open(LockFile); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		// create new lock file
		File, err := os.Create(LockFile)
		if err != nil {
			return err
		}
		// set file permissions
		if err := File.Chmod(0644); err != nil {
			return err
		}
	}
	// acquire lock on file
	c := make(chan error)
	DoLockFile := func() {
		// try to acquire lock
		c <- syscall.Flock(int(File.Fd()), syscall.LOCK_EX)
	}
	// release lock from file
	DoUnlockFile := func() {
		err := <-c
		if err == nil {
			syscall.Flock(int(File.Fd()), syscall.LOCK_UN)
		}
	}
	// run lock in goroutine
	go DoLockFile()
	// wait for lock or timeout
	select {
	case err := <-c:
		if err != nil {
			return err
		}
		// run backup procedure and return
		return r.Run()
	case <-time.After(time.Second * 60):
		// timeout, handle properly
		go DoUnlockFile()
		return ETimeout
	}
	return nil
}
