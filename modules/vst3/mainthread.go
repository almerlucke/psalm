package vst3

/*
#include "glue.h"
*/
import "C"

import "runtime"

// The plugins' main thread: every call marked as made on the main thread of the host (loading a bundle, making a plugin, asking for its parameters, its state, its window) is
// made by one goroutine that keeps its operating system thread for good, which between calls runs the timers, descriptors and window events the plugins asked for
// (v3_pump): plugins made with JUCE wait forever for such a thread
var mainCh = make(chan func(), 64)

func init() {
	ready := make(chan struct{})
	go func() {
		runtime.LockOSThread()
		C.v3_set_main()
		close(ready)
		for {
			select {
			case f := <-mainCh:
				f()
				continue
			default:
			}
			C.v3_pump(20)
		}
	}()
	<-ready
}

// mainDo runs f on the main thread and waits for it; on the main thread it runs f at once
func mainDo(f func()) {
	if C.v3_is_main() != 0 {
		f()
		return
	}
	done := make(chan struct{})
	mainCh <- func() {
		defer close(done)
		f()
	}
	C.v3_wake()
	<-done
}
