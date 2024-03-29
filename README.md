

# mutex
`import "github.com/juju/mutex/v2"`

* [Overview](#pkg-overview)
* [Index](#pkg-index)

## <a name="pkg-overview">Overview</a>
package mutex provides a named machine level mutex shareable between processes.
[![GoDoc](https://godoc.org/github.com/juju/mutex?status.svg)](https://godoc.org/github.com/juju/mutex)

Mutexes have names. Each each name, only one mutex for that name can be
acquired at the same time, within and across process boundaries. If a
process dies while the mutex is held, the mutex is automatically released.

The Linux/MacOS implementation uses flock, while the Windows implementation
uses a named mutex.




## <a name="pkg-index">Index</a>
* [Variables](#pkg-variables)
* [type Clock](#Clock)
* [type Releaser](#Releaser)
  * [func Acquire(spec Spec) (Releaser, error)](#Acquire)
* [type Spec](#Spec)
  * [func (s *Spec) Validate() error](#Spec.Validate)


#### <a name="pkg-files">Package files</a>
[doc.go](/src/github.com/juju/mutex/doc.go) [errors.go](/src/github.com/juju/mutex/errors.go) [legacy_mutex_linux.go](/src/github.com/juju/mutex/legacy_mutex_linux.go) [mutex.go](/src/github.com/juju/mutex/mutex.go) [mutex_flock.go](/src/github.com/juju/mutex/mutex_flock.go) 



## <a name="pkg-variables">Variables</a>
``` go
var (
    ErrTimeout   = errors.New("timeout acquiring mutex")
    ErrCancelled = errors.New("cancelled acquiring mutex")
)
```



## <a name="Clock">type</a> [Clock](/src/target/mutex.go?s=682:907#L28)
``` go
type Clock interface {
    // After waits for the duration to elapse and then sends the
    // current time on the returned channel.
    After(time.Duration) <-chan time.Time

    // Now returns the current clock time.
    Now() time.Time
}
```
Clock provides an interface for dealing with clocks.










## <a name="Releaser">type</a> [Releaser](/src/target/mutex.go?s=337:624#L19)
``` go
type Releaser interface {
    // Release releases the mutex. Release may be called multiple times, but
    // only the first call will release this instance of the mutex. Release is
    // unable to release the mutex successfully it will call panic to forcibly
    // release the mutex.
    Release()
}
```
Releaser defines the Release method that is the only thing that can be done
to a acquired mutex.







### <a name="Acquire">func</a> [Acquire](/src/target/mutex.go?s=1757:1798#L61)
``` go
func Acquire(spec Spec) (Releaser, error)
```
Acquire will attempt to acquire the named mutex. If the Timout value
is hit, ErrTimeout is returned. If the Cancel channel is signalled,
ErrCancelled is returned.





## <a name="Spec">type</a> [Spec](/src/target/mutex.go?s=986:1583#L38)
``` go
type Spec struct {
    // Name is required, and must start with a letter and contain at most
    // 40 letters, numbers or dashes.
    Name string

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
```
Spec defines the name of the mutex and behaviour of the Acquire function.










### <a name="Spec.Validate">func</a> (\*Spec) [Validate](/src/target/mutex.go?s=2640:2671#L105)
``` go
func (s *Spec) Validate() error
```
Validate checks the attributes of Spec for validity.








- - -
Generated by [godoc2md](http://godoc.org/github.com/davecheney/godoc2md)
