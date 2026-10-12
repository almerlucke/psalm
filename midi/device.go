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
	"syscall"
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
// the first device that is not virtual (a "Virtual" card); when there are only virtual ones it
// fails rather than read a port nothing plays into (give their path to use one). A device another
// program holds fails at once instead of waiting for it
func OpenDevice(path string) (*Device, error) {
	if path == "" {
		infos, err := Devices()
		if err != nil {
			return nil, err
		}
		var virtual []string
		for _, i := range infos {
			if !strings.Contains(strings.ToLower(i.Name), "virtual") {
				path = i.Path
				break
			}
			virtual = append(virtual, i.Path)
		}
		if path == "" {
			if len(virtual) > 0 {
				return nil, fmt.Errorf("midi: no keyboard connected, only virtual ports (%s)", strings.Join(virtual, ", "))
			}
			return nil, errors.New("midi: no keyboard connected")
		}
	}
	f, err := openRaw(path)
	if err != nil {
		return nil, err
	}
	d := &Device{Queue: NewQueue(), file: f, path: path, closed: make(chan struct{})}
	go d.read()
	return d, nil
}

// openRaw opens a raw MIDI device, for reading and writing or else for reading. It opens without
// blocking (a busy device fails instead of waiting until it is free), then reads blocking
func openRaw(path string) (*os.File, error) {
	fd, err := syscall.Open(path, syscall.O_RDWR|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		fd, err = syscall.Open(path, syscall.O_RDONLY|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	}
	if errors.Is(err, syscall.EBUSY) {
		return nil, fmt.Errorf("midi: %s is in use by another program", path)
	}
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: path, Err: err}
	}
	if err := syscall.SetNonblock(fd, false); err != nil {
		syscall.Close(fd)
		return nil, err
	}
	return os.NewFile(uintptr(fd), path), nil
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
