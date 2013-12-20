package main

import (
	"github.com/remogatto/application"
	"time"
)

// mainLoop must implements the Looper interface.
type mainLoop struct {
	*application.BaseLoop
	initialized chan int
	running     bool
	ticker      *time.Ticker
	durationCh  chan string
	startedCh   chan bool
}

// Run() runs the loop.
func (loop *mainLoop) Run() {
	loop.running = true

	for loop.running {
		select {
		// Send a value over the channel in order to
		// signal that the loop started.
		case loop.initialized <- 1:

			// A request to pause the loop is received.
		case <-loop.PauseCh:
			// do something or simply send-back a value to
			// the pause channel.
			loop.PauseCh <- 0

			// A request to terminate the loop is received.
		case <-loop.TerminateCh:
			loop.running = false
			loop.TerminateCh <- 0

			// Receive a tick from the ticker.
		case <-loop.ticker.C:
			// Initiate the exit procedure.
			application.Exit()

			// Receive a duration string and create a proper
			// ticker from it.
		case durationStr := <-loop.durationCh:
			duration, err := time.ParseDuration(durationStr)
			if err != nil {
				panic("Error parsing a duration string.")
			}
			loop.ticker = time.NewTicker(duration)
			application.Logf("A new duration received. Running for %s...", durationStr)
		}
	}
}

func newMainLoop() *mainLoop {
	return &mainLoop{
		BaseLoop:    application.NewBaseLoop(),
		initialized: make(chan int),
		durationCh:  make(chan string),
		startedCh:   make(chan bool),
		ticker:      time.NewTicker(10 * time.Second),
	}
}

func main() {
	// Turn on verbose mode.
	application.Verbose = true

	// Create an instance of mainLoop.
	mainLoop := newMainLoop()

	// Register the loop under a name.
	application.Register("mainLoop", mainLoop)

	sendWrong := true

	// Run the registered loops on separate goroutines.
	go application.Run()

	// A control loop follows. The loop has the responsibility to
	// receive/send messages from/to the loops.
	for {
		select {
		case <-mainLoop.initialized:
			// As soon as the loop is initialized, send a
			// wrong duration string in order to raise an
			// error.
			if sendWrong {
				application.Logf("Sending a wrong duration to the mainLoop")
				mainLoop.durationCh <- "2 seconds"
				sendWrong = false
			}
		case <-application.ExitCh:
			// Catch the exit signal and print a last
			// message.
			application.Logf("Very last message before exiting.")
			// Exit from the control loop.
			return
		case err := <-application.ErrorCh:
			application.Printf("An error was received: \"%v\"\n", err)

			// Restart the loop.
			application.Start("mainLoop")

			// Send a correct duration string.
			mainLoop.durationCh <- "2s"
		}
	}
}
