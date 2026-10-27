package clap

/*
#include "glue.h"
*/
import "C"

import "runtime"

// The plugins' main thread. A plugin has a thread it expects the calls marked [main-thread] to come from, and its window, timers and file descriptors to be run by:
// plugins made with JUCE wait for it forever when the thread that made them does not run their loop. So every such call (loading a library, making a plugin,
// asking for its parameters, its state, its window) is made by one goroutine that keeps its operating system thread for good, which between calls runs the
// timers, descriptors and window events the plugins asked for (pc_pump)
var mainCh = make(chan func(), 64)

func init() {
	ready := make(chan struct{})
	go func() {
		runtime.LockOSThread()
		C.pc_set_main()
		close(ready)
		for {
			select {
			case f := <-mainCh:
				f()
				continue
			default:
			}
			C.pc_pump(20)
		}
	}()
	<-ready
}

// mainDo runs f on the main thread and waits for it; on the main thread it runs f at once
func mainDo(f func()) {
	if C.pc_is_main() != 0 {
		f()
		return
	}
	done := make(chan struct{})
	mainCh <- func() {
		defer close(done)
		f()
	}
	C.pc_wake()
	<-done
}
