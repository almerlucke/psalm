package clap

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// shownNumber reads a number and its unit from what a plugin shows ("423.4 ms" is 0.4234 of "s", "1.2 kHz" 1200 of "Hz", "-1" a number with no unit)
func shownNumber(s string) (n float64, unit string, ok bool) {
	s = strings.TrimSpace(s)
	end := 0
	for end < len(s) && (s[end] == '-' || s[end] == '+' || s[end] == '.' || (s[end] >= '0' && s[end] <= '9')) {
		end++
	}
	n, err := strconv.ParseFloat(s[:end], 64)
	if err != nil {
		return 0, "", false
	}
	unit = strings.TrimSpace(s[end:])
	switch unit {
	case "ms":
		return n / 1000, "s", true
	case "kHz":
		return n * 1000, "Hz", true
	}
	return n, unit, true
}

// expand is the key of the parameter that key stands for: the key itself, or the one key that ends with it (" a osc 1 type")
func (p *Plugin) expand(key string) (string, error) {
	if _, ok := p.keys[key]; ok {
		return key, nil
	}
	found := ""
	for k := range p.keys {
		if strings.HasSuffix(k, " "+key) {
			if found != "" {
				return "", fmt.Errorf("%q is %q and %q", key, found, k)
			}
			found = k
		}
	}
	if found == "" {
		return "", fmt.Errorf("no parameter %q", key)
	}
	return found, nil
}

// Resolve finds the value (0 to 1, what a message sets) that makes the plugin show text for the parameter with this key: the text as it shows it ("Sine", "Poly",
// "Reverb 1"), @ and a value from 0 to 1 for one that shows nothing, or a number and its unit ("423 ms", "12 cents", "-1"): the value that shows the nearest number in that unit. A key may be the end of one ("a osc 1
// type"). It returns the whole key
func (p *Plugin) resolveMain(key, text string) (string, float64, error) {
	full, err := p.expand(key)
	if err != nil {
		return "", 0, err
	}
	if raw, ok := strings.CutPrefix(text, "@"); ok { // a parameter that shows nothing: its value as it is
		v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
		if err != nil || v < 0 || v > 1 {
			return "", 0, fmt.Errorf("%s: %q is not a value from 0 to 1", full, text)
		}
		return full, v, nil
	}
	const steps = 2000
	shown := make([]string, steps+1)
	for i := range shown {
		shown[i] = p.Text(full, float64(i)/steps)
	}
	// the text as it shows: the middle of the first run of values that show it
	for i := 0; i <= steps; i++ {
		if strings.EqualFold(shown[i], text) {
			j := i
			for j < steps && strings.EqualFold(shown[j+1], text) {
				j++
			}
			return full, float64(i+j) / 2 / steps, nil
		}
	}
	// a number: the nearest in its unit
	want, unit, ok := shownNumber(text)
	if !ok {
		return "", 0, fmt.Errorf("%s never shows %q", full, text)
	}
	best, dist := -1, math.Inf(1)
	for i, s := range shown {
		if n, u, ok := shownNumber(s); ok && u == unit {
			if d := math.Abs(n - want); d < dist {
				best, dist = i, d
			}
		}
	}
	if best < 0 {
		return "", 0, fmt.Errorf("%s never shows a number in %q (as %q)", full, unit, text)
	}
	// a number shown over many values (a step of a stepped parameter): the middle of its run
	hi := best
	for hi < steps && shown[hi+1] == shown[best] {
		hi++
	}
	lo := best
	for lo > 0 && shown[lo-1] == shown[best] {
		lo--
	}
	return full, float64(lo+hi) / 2 / steps, nil
}

// Resolve is resolveMain on the main thread of the plugins (it asks the plugin for the text of thousands of values)
func (p *Plugin) Resolve(key, text string) (full string, value float64, err error) {
	mainDo(func() { full, value, err = p.resolveMain(key, text) })
	return
}
