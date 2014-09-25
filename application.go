/*
Package application implements a simple infrastructure for building
concurrent programs..  An application is considered as a set of loops
which interact each other. The application package is responsibile for
the lifecycle of these loops.
*/
package application

import (
	"fmt"
	"log"
	"runtime"
	"sync"
    "os"
)

type Error struct {
	RuntimeError interface{}
	Stack        string
}

func (e Error) Error() string {
	switch err := e.RuntimeError.(type) {
	case error:
		return err.(error).Error()
	case string:
		return err
	}
	return ""
}

type RerunError struct {
	ApplicationError Error
}

func (e RerunError) Error() string {
	return "Run cannot be called more than once!"
}

type BaseLoop struct {
	PauseCh, TerminateCh chan int
}

func NewBaseLoop() *BaseLoop {
	return &BaseLoop{
		PauseCh:     make(chan int),
		TerminateCh: make(chan int),
	}
}

func (baseLoop *BaseLoop) Pause() chan int {
	return baseLoop.PauseCh
}

func (baseLoop *BaseLoop) Terminate() chan int {
	return baseLoop.TerminateCh
}

func (baseLoop BaseLoop) Run() {}

// Looper is an interface for application's loops.
type Looper interface {
	// Run runs the loop.
	// Usually Run contains a for {} statement.
	Run()

	// Pause returns a chan int channel.
	// Values sent to this channel will pause the loop.
	Pause() chan int

	// Terminate returns a chan int channel.
	// Values sent to this channel will terminate the loop.
	Terminate() chan int
}

var (
	// NumLoops is the number of registered loops.
	NumLoops int

	// Verbose is a boolean flag. If true it enables a lot of
	// console output.
	Verbose bool

	// Verbose is a boolean flag. If true it enables a lot of
	// debugging output.
	Debug bool

	// ErrorCh is a receive-only channel from which client code
	// receive errors from application.
	ErrorCh <-chan interface{}

	// Exit receive a boolean value when the application exits.
	ExitCh <-chan bool

	loops      map[string]Looper
	terminated chan bool
	errorCh    chan interface{}
	exitCh     chan bool

	closing, running bool

	// Global mutexes.
	rwMutex sync.RWMutex
	mutex   sync.Mutex
)

// Register registers a loop.
func Register(name string, loop Looper) error {
	defer rwMutex.Unlock()
	rwMutex.Lock()
	_, exists := loops[name]
	if exists {
		return fmt.Errorf("A loop with the same name %s was already registered!", name)
	}
	loops[name] = loop
	NumLoops = len(loops)
	return nil
}

// Loop returns a registered loop.
func Loop(name string) (Looper, error) {
	defer mutex.Unlock()
	mutex.Lock()
	loop, exists := loops[name]
	if !exists {
		return nil, fmt.Errorf("Loop %s doesn't exist", name)
	}
	return loop, nil
}

// SendExit initiates the termination process.
func Exit() {
	if !closing {
		closing = true
		close(terminated)
	}
}

// Start (re)starts the named loop in a separated goroutine. If the
// goroutine panics, the error is recovered and sent to the error
// channel.
func Start(name string) (err error) {
	var loop Looper
	if loop, err = Loop(name); err != nil {
		return err
	}
	go func() {
		defer func() {
			// Recover from a panicking goroutine and
			// forward the error to the error channel.
			if untypedErr := recover(); untypedErr != nil {
				errorCh <- Error{
					untypedErr,
					stacktrace(),
				}

			}
		}()
		Logf("Run %s\n", name)
		loop.Run()
	}()
	return nil
}

// Run starts the all the registered loops in separated goroutines. It
// blocks until Exit() is called.  When the application is terminated,
// it terminates all the registered loops. This is a procedure of two
// phases:
//
//	1. Pause all event loops (i.e: stop all tickers)
//	2. Terminate all event loops
//
// The two phases are required because we have no idea what the relationship
// among the event-loops might be. We are assuming the worst case scenario
// in which the relationships among event-loops form a graph - in such a case
// it is unclear whether an event-loop can be terminated without knowing
// that all event-loops are paused.
func Run() {
	if running {
		applicationError := Error{Stack: stacktrace()}
		errorCh <- RerunError{applicationError}
	}

	running = true
	closing = false

	for name, _ := range loops {
		if err := Start(name); err != nil {
			panic(err)
		}
	}

	<-terminated

	mutex.Lock()
	for name, loop := range loops {
		Logf("Waiting for %s to pause\n", name)
		loop.Pause() <- 0
		<-loop.Pause()
		Logf("%s was paused\n", name)
	}
	mutex.Unlock()

	mutex.Lock()
	for name, loop := range loops {
		Logf("Waiting for %s to terminate\n", name)
		loop.Terminate() <- 0
		<-loop.Terminate()
		delete(loops, name)
		Logf("%s was terminated\n", name)
	}
	mutex.Unlock()

	Logf("%s", "Exiting from application...")
	exitCh <- true
}

// Printf is an helper function that produces formatted log.
func Printf(fmt string, v ...interface{}) {
	log.Printf(fmt, v...)
}

// Logf is an helper function that produces formatted log.
// Logf produces output only if Verbose variable is true.
func Logf(fmt string, v ...interface{}) {
	if Verbose {
		log.Printf(fmt, v...)
	}
}

// Debugf is an helper function that produces formatted log.
// Debugf produces output only if Debug variable is true.
func Debugf(fmt string, v ...interface{}) {
	if Debug {
		log.Printf(fmt, v...)
	}
}

// Fatal is a helper function that logs the error and exits the app when called
func Fatal(v ...interface{}) {
    log.Fatalf("v: %v\n", v)
    os.Exit(1)
}

func stacktrace() string {
	// Write a stack trace
	buf := make([]byte, 10000)
	n := runtime.Stack(buf, true)

	// Incrementally grow the
	// buffer as the stack trace
	// requires.
	for n > len(buf) {
		buf = make([]byte, len(buf)*2)
		n = runtime.Stack(buf, false)
	}
	return string(buf)
}

func init() {
	terminated = make(chan bool)
	exitCh = make(chan bool)
	errorCh = make(chan interface{})
	ErrorCh = errorCh
	ExitCh = exitCh
	loops = make(map[string]Looper, 0)
}
