// Command gennyclap is a CLAP plugin made of genny: a note effect that plays a piece described in a genny spec
// (JSON, see the docs of genny), in time with the transport of the host. The spec is read from a file
// ($GENNY_SPEC, else ~/.config/psalm/genny.json) and read again when the file changes; the plugin keeps a copy of it
// in the project. Build it with build.sh, copy genny.clap to ~/.clap
package main

/*
#cgo CFLAGS: -I${SRCDIR}/../../../local/clap/include -I${SRCDIR}
#cgo LDFLAGS: -lm
#include <stdlib.h>
#include <string.h>
#include "shim.h"
*/
import "C"

import (
	"time"
	"unsafe"
)

func init() {
	logHook = func(handle, severity int, text string) {
		s := C.CString(text)
		C.genny_host_log(C.int(handle), C.int(severity), s)
		C.free(unsafe.Pointer(s))
	}
}

func at(h C.int) *instance {
	if h < 0 || int(h) >= len(instances) {
		return nil
	}
	return instances[h].Load()
}

//export genny_new
func genny_new() C.int {
	in := newInstance()
	for i := range instances {
		in.handle = i
		if instances[i].CompareAndSwap(nil, in) {
			return C.int(i)
		}
	}
	return -1
}

//export genny_free
func genny_free(h C.int) {
	if in := at(h); in != nil {
		instances[h].Store(nil)
	}
}

// genny_start starts the instance's reading of the spec (when the plugin is activated)
//
//export genny_start
func genny_start(h C.int) {
	in := at(h)
	if in == nil || in.stop != nil {
		return
	}
	in.stop, in.done = make(chan struct{}), make(chan struct{})
	go in.run()
}

//export genny_stop
func genny_stop(h C.int) {
	if in := at(h); in != nil && in.stop != nil {
		close(in.stop)
		// a render that is going on is waited for (not long: the host is waiting), so it does not log to a host that is gone
		select {
		case <-in.done:
		case <-time.After(2 * time.Second):
		}
		in.stop, in.done = nil, nil
	}
}

// genny_epoch counts the times the piece was replaced (0: there is none yet); when it changes the notes that sound are let go
//
//export genny_epoch
func genny_epoch(h C.int) C.longlong {
	if in := at(h); in != nil {
		if st := in.store.Load(); st != nil {
			return C.longlong(st.epoch)
		}
	}
	return 0
}

//export genny_collect
func genny_collect(h C.int, from, to C.double, out *C.genny_event, capacity C.int) C.int {
	in := at(h)
	if in == nil {
		return 0
	}
	dst := unsafe.Slice(out, int(capacity))
	n := 0
	for _, e := range in.collect(float64(from), float64(to)) {
		if n >= len(dst) {
			break
		}
		on := C.int(0)
		if e.on {
			on = 1
		}
		dst[n] = C.genny_event{beat: C.double(e.beat), velocity: C.double(e.velocity), tune: C.double(e.tune), key: C.int(e.key), channel: C.int(e.channel), id: C.int(e.id), on: on}
		n++
	}
	return C.int(n)
}

// genny_input takes a note played into the plugin (audio thread)
//
//export genny_input
func genny_input(h, key C.int, velocity C.double, on C.int) {
	if in := at(h); in != nil {
		in.input(int(key), float64(velocity), on != 0)
	}
}

//export genny_param_count
func genny_param_count() C.int { return numParams }

//export genny_param_info
func genny_param_info(index C.int, out *C.genny_param) C.int {
	if index < 0 || int(index) >= numParams {
		return 0
	}
	p := params[index]
	out.id = index
	out.stepped = 0
	if p.stepped {
		out.stepped = 1
	}
	out.min, out.max, out.def = C.double(p.min), C.double(p.max), C.double(p.def)
	name := append([]byte(p.name), 0)
	C.memcpy(unsafe.Pointer(&out.name[0]), unsafe.Pointer(&name[0]), C.size_t(len(name)))
	return 1
}

//export genny_param_set
func genny_param_set(h, id C.int, value C.double) {
	if in := at(h); in != nil && id >= 0 && int(id) < numParams {
		in.set(int(id), float64(value))
	}
}

//export genny_param_get
func genny_param_get(h, id C.int) C.double {
	if in := at(h); in != nil && id >= 0 && int(id) < numParams {
		return C.double(in.get(int(id)))
	}
	return 0
}

// genny_state_write writes the state into buf when it fits, and returns its size
//
//export genny_state_write
func genny_state_write(h C.int, buf *C.char, capacity C.int) C.int {
	in := at(h)
	if in == nil {
		return 0
	}
	b := in.saveState()
	if buf != nil && int(capacity) >= len(b) {
		copy(unsafe.Slice((*byte)(unsafe.Pointer(buf)), len(b)), b)
	}
	return C.int(len(b))
}

//export genny_state_read
func genny_state_read(h C.int, buf *C.char, size C.int) {
	if in := at(h); in != nil {
		in.loadState(C.GoBytes(unsafe.Pointer(buf), size))
	}
}

func main() {}
