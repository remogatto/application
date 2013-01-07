/* 
Package application implements a simple infrastructure for building
programs.
An application is considered as a set of loops which interact each
other. The application package is responsibile for running, pausing
and stopping this loops.
*/
package application

import (
	"log"
	"sync"
)

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

	loops      map[string]Looper
	terminated chan bool
	closing    bool
	rwMutex    sync.RWMutex
	mutex      sync.Mutex
)

// Register registers a loop.
func Register(name string, loop Looper) {
	rwMutex.Lock()
	loops[name] = loop
	NumLoops = len(loops)
	rwMutex.Unlock()
}

// Loop returns a registered loop.
func Loop(name string) Looper {
	mutex.Lock()
	loop := loops[name]
	mutex.Unlock()
	return loop
}

// Exit initiates the termination process.
func Exit() {
	if !closing {
		closing = true
		close(terminated)
	}
}

// Run runs the registered loops in separated goroutines.
// When the application is terminated, it terminates all the registered
// loops. This is a procedure of two phases:
//
//	1. Pause all event loops (i.e: stop all tickers)
//	2. Terminate all event loops
//
// The two phases are required because we have no idea what the relationship
// among the event-loops might be. We are assuming the worst case scenario
// in which the relationships among event-loops form a graph - in such a case
// it is unclear whether an event-loop can be terminated without knowing
// that all event-loops are paused.
func Run(exitCh chan bool) {
	for name, loop := range loops {
		Logf("Run %s\n", name)
		go loop.Run()
	}
	<-terminated
	for name, loop := range loops {
		Logf("Waiting for %s to pause\n", name)
		loop.Pause() <- 0
		<-loop.Pause()
		Logf("%s was paused\n", name)
	}
	for name, loop := range loops {
		Logf("Waiting for %s to terminate\n", name)
		loop.Terminate() <- 0
		<-loop.Terminate()
		Logf("%s was terminated\n", name)
	}
	Logf("%s", "Exiting from application...")
	close(exitCh)
}

// Logf is an helper function that produces formatted log.
// Logf produces output only if Verbose variable is true.
func Logf(fmt string, v ...interface{}) {
	if Verbose {
		log.Printf(fmt, v...)
	}
}

func init() {
	loops = make(map[string]Looper, 0)
	terminated = make(chan bool)
}
