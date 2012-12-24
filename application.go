package application

import (
	"log"
	"sync"
)

const (
	PAUSE = iota
	TERMINATE
)

type Looper interface {
	Run()
	Pause() chan int
	Terminate() chan int
}

var (
	NumLoops int
	Verbose bool
	loops map[string]Looper
	terminated chan bool
	closing bool
	rwMutex sync.RWMutex
	mutex sync.Mutex
)

func Register(name string, loop Looper) {
	rwMutex.Lock()
	loops[name] = loop
	NumLoops = len(loops)
	rwMutex.Unlock()
}

func Loop(name string) Looper {
	mutex.Lock()
	loop := loops[name]
	mutex.Unlock()
	return loop
}

func Exit() {
	if (!closing) {
		closing = true
		close(terminated)
	}
}

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
	Logf("%v", "Exiting from application...")
	close(exitCh)
}

func Logf(fmt string, v ...interface{}) {
	if Verbose {
		log.Printf(fmt, v...)
	}
}

func init() {
	loops = make(map[string]Looper, 0)
	terminated = make(chan bool)
}
