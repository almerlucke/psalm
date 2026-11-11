package keyboard

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// The keys of Config.Routes for the pitch bend and the aftertouch (the controllers are 0-127)
const (
	RoutePressure = 128 // the aftertouch (channel pressure)
	RouteBend     = 129 // the pitch bend, its middle the middle of the range
)

// Route sends a controller (or the bend, or the aftertouch) to any setting of a synth, scaled to
// the setting's range: the wheel sweeping the cutoff from 300 to 8000 Hz, a slider moving the
// vowel. It sets the setting: the controller at half sets the middle of the range
type Route struct {
	Name     string  // the setting, as the synth takes it in a message ("cutoff", "fold", "vowel")
	Min, Max float64 // the setting at the controller's bottom and top (Max under Min turns it round)
	Exp      bool    // exponentially from Min to Max (frequencies, times; both above 0), not in a line
	Int      bool    // whole numbers (a waveform, a mode)
}

// Value returns the setting for a controller's position (0-1)
func (r Route) Value(v float64) float64 {
	v = max(0, min(v, 1))
	var x float64
	if r.Exp && r.Min > 0 && r.Max > 0 {
		x = r.Min * math.Pow(r.Max/r.Min, v)
	} else {
		x = r.Min + (r.Max-r.Min)*v
	}
	if r.Int {
		x = math.Round(x)
	}
	return x
}

// ParseRoute reads a route as text, "source:name:min:max[:exp|:int]", the source a controller
// number, "bend" or "pressure": "1:cutoff:300:8000:exp" is the mod wheel sweeping the cutoff
func ParseRoute(text string) (int, Route, error) {
	parts := strings.Split(text, ":")
	if len(parts) < 4 || len(parts) > 5 {
		return 0, Route{}, fmt.Errorf("keyboard: route %q: want source:name:min:max[:exp|:int]", text)
	}
	var source int
	switch parts[0] {
	case "bend":
		source = RouteBend
	case "pressure", "aftertouch":
		source = RoutePressure
	default:
		cc, err := strconv.Atoi(parts[0])
		if err != nil || cc < 0 || cc > 127 {
			return 0, Route{}, fmt.Errorf("keyboard: route %q: the source is a controller (0-127), bend or pressure", text)
		}
		source = cc
	}
	r := Route{Name: parts[1]}
	if r.Name == "" {
		return 0, Route{}, fmt.Errorf("keyboard: route %q: no setting", text)
	}
	var err1, err2 error
	r.Min, err1 = strconv.ParseFloat(parts[2], 64)
	r.Max, err2 = strconv.ParseFloat(parts[3], 64)
	if err1 != nil || err2 != nil {
		return 0, Route{}, fmt.Errorf("keyboard: route %q: bad range", text)
	}
	if len(parts) == 5 {
		switch parts[4] {
		case "exp":
			r.Exp = true
			if r.Min <= 0 || r.Max <= 0 {
				return 0, Route{}, fmt.Errorf("keyboard: route %q: an exponential range is above 0", text)
			}
		case "int":
			r.Int = true
		default:
			return 0, Route{}, fmt.Errorf("keyboard: route %q: %q is exp or int", text, parts[4])
		}
	}
	return source, r, nil
}

// routed sends the routes of a source at a position (0-1), true when it has any
func (k *Keyboard) routed(source int, v float64) bool {
	routes := k.cfg.Routes[source]
	if len(routes) == 0 {
		return false
	}
	msg := make(map[string]any, len(routes))
	for _, r := range routes {
		msg[r.Name] = r.Value(v)
	}
	k.send(msg)
	return true
}
