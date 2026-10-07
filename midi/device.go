package midi

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Device is a raw MIDI device of the operating system (on Linux /dev/snd/midiC<card>D<device>,
// what a USB keyboard shows up as): an input, read in the background, and an output
type Device struct {
	*Queue
	file      *os.File
	path      string
	writeM    sync.Mutex
	closed    chan struct{}
	err       error
	OnMessage func(Message) // when set, called for every message as it arrives (from the reader goroutine)
}

// DeviceInfo describes a raw MIDI device
type DeviceInfo struct {
	Path string
	Name string
}

func (d DeviceInfo) String() string { return fmt.Sprintf("%s (%s)", d.Path, d.Name) }

// Devices lists the raw MIDI devices, with the names of their cards
func Devices() ([]DeviceInfo, error) {
	paths, err := filepath.Glob("/dev/snd/midiC*D*")
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	var infos []DeviceInfo
	for _, p := range paths {
		var card, dev int
		name := ""
		if _, err := fmt.Sscanf(filepath.Base(p), "midiC%dD%d", &card, &dev); err == nil {
			if b, err := os.ReadFile(fmt.Sprintf("/proc/asound/card%d/id", card)); err == nil {
				name = strings.TrimSpace(string(b))
			}
			if b, err := os.ReadFile(fmt.Sprintf("/proc/asound/card%d/midi%d", card, dev)); err == nil {
				if line, _, _ := strings.Cut(string(b), "\n"); strings.TrimSpace(line) != "" {
					name = strings.TrimSpace(name + ": " + strings.TrimSpace(line))
				}
			}
		}
		infos = append(infos, DeviceInfo{Path: p, Name: name})
	}
	return infos, nil
}

// OpenDevice opens a raw MIDI device for reading (and writing, when allowed). An empty path opens
// the first device that is not virtual (a "Virtual" card), or the first device
func OpenDevice(path string) (*Device, error) {
	if path == "" {
		infos, err := Devices()
		if err != nil {
			return nil, err
		}
		if len(infos) == 0 {
			return nil, errors.New("midi: no devices")
		}
		path = infos[0].Path
		for _, i := range infos {
			if !strings.Contains(strings.ToLower(i.Name), "virtual") {
				path = i.Path
				break
			}
		}
	}
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		f, err = os.Open(path)
		if err != nil {
			return nil, err
		}
	}
	d := &Device{Queue: NewQueue(), file: f, path: path, closed: make(chan struct{})}
	go d.read()
	return d, nil
}

// Path returns the path of the device
func (d *Device) Path() string { return d.path }

// Err returns why reading stopped (nil while it reads, or after Close)
func (d *Device) Err() error {
	select {
	case <-d.closed:
		return d.err
	default:
		return nil
	}
}

func (d *Device) read() {
	defer close(d.closed)
	var p Parser
	buf := make([]byte, 256)
	for {
		n, err := d.file.Read(buf)
		if n > 0 {
			at := d.now()
			p.Feed(buf[:n], func(m Message) {
				if m.Kind() == ActiveSensing {
					return
				}
				d.PushAt(m, at)
				if d.OnMessage != nil {
					d.OnMessage(append(Message(nil), m...))
				}
			})
		}
		if err != nil {
			if !errors.Is(err, os.ErrClosed) && !errors.Is(err, io.EOF) {
				d.err = err
			}
			return
		}
	}
}

// Send writes a message to the device
func (d *Device) Send(msg Message) error {
	d.writeM.Lock()
	defer d.writeM.Unlock()
	_, err := d.file.Write(msg)
	return err
}

// Close stops reading and closes the device
func (d *Device) Close() error {
	err := d.file.Close()
	select {
	case <-d.closed:
	case <-time.After(time.Second): // a device that can't be interrupted: leave the reader
	}
	return err
}
