// This code is extracted from GoSpeccy
// (https://github.com/remogatto/gospeccy) and it was mainly
// contributed by Atom (http://github.com/0xe2-0x9a-0x9b)

package application

import (
	"os"
	"sync"
	"os/signal"
)

type SignalHandler interface {
	// Function to be called upon receiving an os.Signal.
	//
	// A single signal is passed to all installed signal handlers.
	// The [order in which this function is called in respect to other handlers] is unspecified.
	HandleSignal(signal os.Signal)
}

// Actually, this is a set
var signalHandlers = make(map[SignalHandler]bool)

var signalHandlers_mutex sync.Mutex

// Installs the specified handler.
// Trying to re-install an already installed handler is effectively a NOOP.
func InstallSignalHandler(handler SignalHandler) {
	signalHandlers_mutex.Lock()
	signalHandlers[handler] = true
	signalHandlers_mutex.Unlock()
}

// Uninstalls the specified handler.
// Trying to uninstall an non-existent handler is effectively a NOOP.
func UninstallSignalHandler(handler SignalHandler) {
	signalHandlers_mutex.Lock()
	delete(signalHandlers, handler)
	signalHandlers_mutex.Unlock()
}

func init() {
	go func() {
		c := make(chan os.Signal, 10)
		signal.Notify(c)
		for sig := range c {
			signalHandlers_mutex.Lock()
			handlers_copy := make([]SignalHandler, len(signalHandlers))
			{
				i := 0
				for handler, _ := range signalHandlers {
					handlers_copy[i] = handler
					i++
				}
			}
			signalHandlers_mutex.Unlock()

			for _, handler := range handlers_copy {
				handler.HandleSignal(sig)
			}
		}
	}()
}
