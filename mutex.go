// Copyright 2016 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package mutex

import (
	"crypto/md5"
	"fmt"
	"time"

	"github.com/juju/errors"
)

// Releaser defines the Release method that is the only thing that can be done
// to an acquired mutex.
type Releaser interface {
	// Release releases the mutex. Release may be called multiple times, but
	// only the first call will release this instance of the mutex. Release is
	// unable to release the mutex successfully it will call panic to forcibly
	// release the mutex.
	Release()
}

// Clock provides an interface for dealing with clocks.
type Clock interface {
	// After waits for the duration to elapse and then sends the
	// current time on the returned channel.
	After(time.Duration) <-chan time.Time

	// Now returns the current clock time.
	Now() time.Time
}

// Spec defines the name of the mutex and behaviour of the Acquire function.
type Spec struct {
	// Name is required.
	Name string

	// Prefix can be used to control the mutex scope.
	//
	// For example:
	//
	// r, _ := mutex.Acquire(Spec{Prefix: "p0", Name: "n", Clock: clock.WallClock, Delay: time.Second})
	// defer r.Release()
	//
	// r, _ := mutex.Acquire(Spec{Prefix: "p1", Name: "n", Clock: clock.WallClock, Delay: time.Second})
	// defer r.Release()
	//
	// // This line will be executed immediately because we are acquiring locks in different scopes.
	Prefix string

	// Clock must be provided and is exposed for testing purposes.
	Clock Clock

	// Delay defines how often to check for lock acquisition, for
	// compatibility code that requires polling.
	Delay time.Duration

	// Timeout allows the caller to specify how long to wait. If Timeout
	// is zero, then the call will block forever.
	Timeout time.Duration

	// Cancel if signalled will cause the Acquire method to return with ErrCancelled.
	Cancel <-chan struct{}
}

// Acquire will attempt to acquire the named mutex. If the Timout value
// is hit, ErrTimeout is returned. If the Cancel channel is signalled,
// ErrCancelled is returned.
func Acquire(spec Spec) (Releaser, error) {
	if err := spec.Validate(); err != nil {
		return nil, errors.Trace(err)
	}

	var timeout <-chan time.Time
	if spec.Timeout > 0 {
		timeout = spec.Clock.After(spec.Timeout)
	}

	return acquire(spec, timeout)
}

// Validate checks the attributes of Spec for validity.
func (s *Spec) Validate() error {
	if s.Name == "" {
		return errors.NotValidf("missing Name")
	}
	if len(s.Prefix) > 7 {
		return errors.NotValidf("prefix length cannot be greater than 7")
	}
	if s.Clock == nil {
		return errors.NotValidf("missing Clock")
	}
	if s.Delay <= 0 {
		return errors.NotValidf("non positive Delay")
	}
	if s.Timeout < 0 {
		return errors.NotValidf("negative Timeout")
	}
	return nil
}

// GetMutexName returns the internal name of the mutex
func (s *Spec) GetMutexName() string {
	h := md5.New()
	h.Write([]byte(s.Name))

	// The MD5 hash is 32 chars long
	// We can fit up to 40 charts with the prefix,
	// so we can use up to 7 chars plus the dash.
	prefix := "mutex"
	if s.Prefix != "" {
		prefix = s.Prefix
	}

	return fmt.Sprintf("%s-%x", prefix, h.Sum(nil))
}
