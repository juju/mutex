// Copyright 2016-2017 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

// +build !windows

package mutex

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/juju/errors"
)

func acquire(spec Spec, timeout <-chan time.Time) (Releaser, error) {
	done := make(chan struct{})
	defer close(done)
	select {
	case result := <-acquireFlock(spec.Name, done):
		if result.err != nil {
			return nil, errors.Trace(result.err)
		}
		return result.m, nil
	case <-timeout:
		return nil, ErrTimeout
	case <-spec.Cancel:
		return nil, ErrCancelled
	}
}

var (
	mu     sync.Mutex
	active = make(map[string][]*waiter)
)

type waiter struct {
	ch   chan<- acquireResult
	done <-chan struct{}
}

type acquireResult struct {
	m   Releaser
	err error
}

// acquireFlock returns a channel on which an acquireResult will be
// sent when the mutex acquisition completes, successfully or not.
//
// In Go, there is no natural way to cancel a flock syscall, so we
// allow a flock syscall to continue on even when there is nothing
// waiting for it, and then release the lock in that case. To avoid
// an unbounded collection of goroutines, we ensure that there is
// only one goroutine making a flock syscall at a time, per mutex
// name.
func acquireFlock(name string, done <-chan struct{}) <-chan acquireResult {
	result := make(chan acquireResult)
	w := &waiter{result, done}

	mu.Lock()
	defer mu.Unlock()
	if waiters, ok := active[name]; ok {
		active[name] = append(waiters, w)
		return result
	}

	flockName := filepath.Join(os.TempDir(), "juju-"+name)
	chownFromRoot := func() (err error) {
		if cmd, ok := os.LookupEnv("SUDO_COMMAND"); ok && cmd != "" {
			var uid, gid int
			uid, err = strconv.Atoi(os.Getenv("SUDO_UID"))
			if err != nil {
				err = errors.Annotate(err, "parsing SUDO_UID")
				return
			}
			gid, err = strconv.Atoi(os.Getenv("SUDO_GID"))
			if err != nil {
				err = errors.Annotate(err, "parsing SUDO_GID")
				return
			}
			return syscall.Chown(flockName, uid, gid)
		}
		return nil
	}
	open := func() (fd int, err error) {
		fd, err = syscall.Open(flockName, syscall.O_CREAT|syscall.O_RDONLY|syscall.O_CLOEXEC, 0600)
		if err != nil {
			if os.IsPermission(err) {
				err = errors.Annotatef(err, "unable to open %s", flockName)
			}
			return
		}
		// Attempting to open a lock file as root whilst using sudo can cause
		// the lock file to have the wrong permissions for a non-sudo user.
		// Subsequent calls to acquire the lock file can then lead to cryptic
		// error messages. Let's attempt to help people out, either by
		// correcting the permissions, or explaining why we can't help them.
		// info: lp 1758369
		if chownErr := chownFromRoot(); chownErr != nil {
			// The file has the wrong permissions, but we should let the acquire
			// continue.
		}
		return
	}
	flock := func() (Releaser, error) {
		fd, err := open()
		if err != nil {
			return nil, errors.Trace(err)
		}
		if err := syscall.Flock(fd, syscall.LOCK_EX); err != nil {
			syscall.Close(fd)
			return nil, errors.Trace(err)
		}
		return &mutex{fd: fd}, nil
	}
	acquire := func() bool {
		releaser, err := flock()
		result := acquireResult{releaser, err}
		mu.Lock()
		defer mu.Unlock()

		var sent bool
		waiters := active[name]
		for !sent && len(waiters) > 0 {
			w, waiters = waiters[0], waiters[1:]
			select {
			case w.ch <- result:
				sent = true
			case <-w.done:
				// waiter is done, so just
				// remove it from the list
				// and try the next one
			}
		}
		if !sent && releaser != nil {
			// No active waiters, release the lock.
			releaser.Release()
		}
		if len(waiters) > 0 {
			active[name] = waiters
			return true
		}
		delete(active, name)
		return false
	}
	go func() {
		for acquire() {
		}
	}()

	active[name] = []*waiter{w}
	return result
}

// mutex implements Releaser using the flock syscall.
type mutex struct {
	mu sync.Mutex
	fd int
}

// Release is part of the Releaser interface.
func (m *mutex) Release() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fd == 0 {
		return
	}
	err := syscall.Close(m.fd)
	if err != nil {
		panic(err)
	}
	m.fd = 0
}
