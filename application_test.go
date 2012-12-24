package application

import (
	pt "github.com/remogatto/prettytest"
	"testing"
)

const COUNT = 3

type testSuite struct {
	pt.Suite
}

type OddLoop struct {
	pause, terminate chan int
	running bool
}

func (loop *OddLoop) Pause() chan int {
	return loop.pause
}

func (loop *OddLoop) Terminate() chan int {
	return loop.terminate
}

func (loop *OddLoop) Run() {
	loop.running = true
	for loop.running {
		select {
		case <-loop.pause:
			loop.pause <- 0
			break
		case <-loop.terminate:
			loop.running = false
			loop.terminate <- 0
			break
			
		}
	}
}

type EvenLoop struct {
	pause, terminate chan int
	running bool
}

func (loop *EvenLoop) Pause() chan int {
	return loop.pause
}

func (loop *EvenLoop) Terminate() chan int {
	return loop.terminate
}

func (loop *EvenLoop) Run() {
	loop.running = true
	for loop.running {
		select {
		case <-loop.pause:
			loop.pause <- 0
			break
		case <-loop.terminate:
			loop.running = false
			loop.terminate <- 0
			break
			
		}
			
	}
}

func (t *testSuite) Should_register_new_loops() {
	oddLoop := &OddLoop{
	pause: make(chan int), 
	terminate: make(chan int), 
	}
	evenLoop := &EvenLoop{
	pause: make(chan int), 
	terminate: make(chan int), 
	}
	Register("Odd Counter", oddLoop)
	Register("Even Counter", evenLoop)
	t.Equal(2, NumLoops)
}

func (t *testSuite) Should_wait_until_all_loops_are_terminated() {
	oddLoop := &OddLoop{
	pause: make(chan int), 
	terminate: make(chan int), 
	}
	evenLoop := &EvenLoop{
	pause: make(chan int), 
	terminate: make(chan int), 
	}
	Register("Odd Counter", oddLoop)
	Register("Even Counter", evenLoop)
	exitCh := make(chan bool)
	go Run(exitCh)
	Exit()
	<-exitCh
	t.False(oddLoop.running)
	t.False(evenLoop.running)
}

func TestApplication(t *testing.T) {
	pt.RunWithFormatter(
		t,
		&pt.BDDFormatter{"application"},
		new(testSuite),
	)
}

