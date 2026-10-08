package signal

import (
	"os"
	"os/signal"
	"strings"
	"time"
)

// WaitForSignal waits for one of the signals and returns it
func WaitForSignal(sig ...os.Signal) os.Signal {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, sig...)
	defer signal.Stop(sigc) // a second one ends the program the usual way, if stopping hangs
	return <-sigc
}

// parentInterval is how often WaitForStop looks whether go run is still there
const parentInterval = 250 * time.Millisecond

// WaitForStop waits until the program should stop and says why: one of the signals (Ctrl-C), or,
// for a program started by go run, go run going away. Stopping go run (from an editor, or with
// kill) doesn't stop the program it started: without this it would play on, holding its sound card
// and its ports. After it returns a second signal ends the program the usual way
func WaitForStop(sig ...os.Signal) string {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, sig...)
	defer signal.Stop(sigc)

	var orphaned <-chan time.Time
	if startedByGoRun() {
		parent := os.Getppid()
		ticker := time.NewTicker(parentInterval)
		defer ticker.Stop()
		gone := make(chan time.Time)
		go func() {
			for t := range ticker.C {
				if os.Getppid() != parent {
					gone <- t
					return
				}
			}
		}()
		orphaned = gone
	}

	select {
	case s := <-sigc:
		return s.String()
	case <-orphaned:
		return "go run stopped"
	}
}

// startedByGoRun tells whether the program runs from go run's build directory
func startedByGoRun() bool {
	exe, err := os.Executable()
	return err == nil && strings.Contains(exe, "go-build")
}
