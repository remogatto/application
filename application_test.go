package application

import (
	"errors"
	"fmt"
	pt "github.com/remogatto/prettytest"
	"testing"
)

type basicTestSuite struct {
	pt.Suite
}

type errorTestSuite struct {
	pt.Suite
}

type CountLoop struct {
	*BaseLoop
	initialized     chan int
	running         bool
	count           int
	countCh, nextCh chan int
}

func (loop *CountLoop) Initialized() chan int {
	return loop.initialized
}

func (loop *CountLoop) Run() {
	loop.running = true
	for loop.running {
		select {
		case loop.initialized <- 1:
		case <-loop.PauseCh:
			loop.PauseCh <- 0
		case <-loop.TerminateCh:
			loop.running = false
			loop.TerminateCh <- 0
		case <-loop.nextCh:
			loop.count += 2
		case <-loop.countCh:
			loop.countCh <- loop.count
		}
	}
}

type PanicLoop struct {
	*BaseLoop
	initialized      chan int
	raiseStringError chan int
	raiseError       chan int
}

func (loop *PanicLoop) Initialized() chan int {
	return loop.initialized
}

func (loop *PanicLoop) Run() {
	for {
		select {
		case loop.initialized <- 1:

		case <-loop.raiseStringError:
			panic("That's an error!")
		case <-loop.raiseError:
			panic(errors.New("That's an error!"))
		}
	}
}

var (
	oddLoop, evenLoop *CountLoop
	panicLoop         *PanicLoop
)

func (t *basicTestSuite) BeforeAll() {
	oddLoop = &CountLoop{
		BaseLoop:    NewBaseLoop(),
		initialized: make(chan int),
		nextCh:      make(chan int),
		countCh:     make(chan int),
		count:       0,
	}
	evenLoop = &CountLoop{
		BaseLoop:    NewBaseLoop(),
		initialized: make(chan int),
		nextCh:      make(chan int),
		countCh:     make(chan int),
		count:       1,
	}
	Register("oddLoop", oddLoop)
	Register("evenLoop", evenLoop)

	go Run()
}

func (t *basicTestSuite) AfterAll() {
	Exit()
}

func (t *basicTestSuite) TestNumLoops() {
	t.Equal(2, NumLoops)
}

func (t *basicTestSuite) TestInitialized() {
	t.Equal(1, <-evenLoop.Initialized())
	t.Equal(1, <-oddLoop.Initialized())
}

func (t *basicTestSuite) TestRun() {
	oddLoop.nextCh <- 1
	oddLoop.nextCh <- 1
	oddLoop.countCh <- 1
	oddCount := <-oddLoop.countCh

	evenLoop.nextCh <- 1
	evenLoop.nextCh <- 1
	evenLoop.countCh <- 1

	evenCount := <-evenLoop.countCh

	t.Equal(4, oddCount)
	t.Equal(5, evenCount)
}

func (t *basicTestSuite) TestRegisterTwice() {
	err := Register("oddLoop", oddLoop)
	t.True(err != nil)
}

func (t *basicTestSuite) TestRunMoreThanOnce() {
	go Run()
	err, ok := (<-ErrorCh).(RerunError)
	t.True(ok)
	t.True(err.ApplicationError.Stack != "")
}

func (t *errorTestSuite) TestRuntimeErrorAndRestart() {
	panicLoop = &PanicLoop{
		BaseLoop:         NewBaseLoop(),
		initialized:      make(chan int),
		raiseStringError: make(chan int),
		raiseError:       make(chan int),
	}

	// cheating with global variables
	running = false
	terminated = make(chan bool)

	Register("panicLoop", panicLoop)
	go Run()

	count := 0
	for count < 2 {
		select {
		case <-panicLoop.initialized:
			// raise errors
			if count == 0 {
				panicLoop.raiseStringError <- 1
			} else {
				panicLoop.raiseError <- 1
			}
		case x := <-ErrorCh:
			t.Equal("That's an error!", x.(Error).Error())
			if count == 0 {
				// test start errors
				err := Start("panicLoo")
				t.True(err != nil, fmt.Sprintf("Error message %s", err))

				err = Start("panicLoop")
				t.True(err == nil)
			}
			count++

		}
	}
	Exit()
}

func TestApplication(t *testing.T) {
	pt.Run(t, new(basicTestSuite), new(errorTestSuite))
}
