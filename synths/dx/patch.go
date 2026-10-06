package dx

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// Operator holds the parameters of one operator, all in the classic 0-99 style ranges
type Operator struct {
	EGRates      [4]int // 0-99, rate 1-4 (attack, decay 1, decay 2, release)
	EGLevels     [4]int // 0-99, level 1-4 (level 3 is the sustain level, level 4 the start/end level)
	BreakPoint   int    // 0-99, keyboard scaling break point (39 is C3)
	LeftDepth    int    // 0-99, level scaling depth below the break point
	RightDepth   int    // 0-99, level scaling depth above the break point
	LeftCurve    int    // 0-3: -LIN, -EXP, +EXP, +LIN
	RightCurve   int    // 0-3
	RateScaling  int    // 0-7, higher notes have faster envelopes
	AmpModSens   int    // 0-3, sensitivity to LFO amplitude modulation
	VelocitySens int    // 0-7, how much velocity lowers the level
	OutputLevel  int    // 0-99
	FixedFreq    bool   // fixed frequency instead of a ratio
	Coarse       int    // 0-31, ratio (0 is 0.5) or fixed frequency decade (1, 10, 100, 1000 Hz)
	Fine         int    // 0-99
	Detune       int    // 0-14, 7 is centered
}

// Patch is a complete 6 operator voice
type Patch struct {
	Name          string
	Ops           [NumOperators]Operator // Ops[0] is operator 1
	PitchEGRates  [4]int                 // 0-99
	PitchEGLevels [4]int                 // 0-99, 50 is no pitch shift
	Algorithm     int                    // 0-31 (algorithm 1-32)
	Feedback      int                    // 0-7
	OscKeySync    bool                   // restart the operators at every note
	LFOSpeed      int                    // 0-99
	LFODelay      int                    // 0-99
	LFOPitchDepth int                    // 0-99
	LFOAmpDepth   int                    // 0-99
	LFOKeySync    bool                   // restart the LFO at every note
	LFOWaveform   int                    // 0-5: triangle, saw down, saw up, square, sine, sample & hold
	PitchModSens  int                    // 0-7
	Transpose     int                    // 0-48, 24 is no transposition
	Level         float64                // output gain (0 is 1), for calibrating, not part of the SysEx data
}

// InitVoice returns the classic initial voice: a single sine (operator 1 at full level)
func InitVoice() Patch {
	p := Patch{
		Name:          "INIT VOICE",
		PitchEGRates:  [4]int{99, 99, 99, 99},
		PitchEGLevels: [4]int{50, 50, 50, 50},
		OscKeySync:    true,
		LFOSpeed:      35,
		LFOKeySync:    true,
		PitchModSens:  3,
		Transpose:     24,
	}
	for i := range p.Ops {
		p.Ops[i] = Operator{
			EGRates:    [4]int{99, 99, 99, 99},
			EGLevels:   [4]int{99, 99, 99, 0},
			BreakPoint: 39,
			Coarse:     1,
			Detune:     7,
		}
	}
	p.Ops[0].OutputLevel = 99
	return p
}

// --- SysEx ---

const (
	sysexStart   = 0xF0
	sysexEnd     = 0xF7
	yamahaID     = 0x43
	bulkVoices   = 32
	packedSize   = 128
	unpackedSize = 155
)

var (
	ErrNotSysEx  = errors.New("dx: not a DX voice sysex message")
	ErrChecksum  = errors.New("dx: sysex checksum mismatch")
	ErrTruncated = errors.New("dx: sysex message too short")
)

func checksum(data []byte) byte {
	sum := 0
	for _, b := range data {
		sum += int(b)
	}
	return byte(-sum) & 0x7F
}

func clamp(v int, hi int) int {
	return max(0, min(int(v), hi))
}

// unpackVoice decodes a 128 byte packed voice (as used in 32 voice bulk dumps)
func unpackVoice(d []byte) Patch {
	var p Patch
	for i := range NumOperators {
		o := d[i*17 : i*17+17]
		op := &p.Ops[NumOperators-1-i] // stored operator 6 first
		for k := range 4 {
			op.EGRates[k] = clamp(int(o[k]), 99)
			op.EGLevels[k] = clamp(int(o[4+k]), 99)
		}
		op.BreakPoint = clamp(int(o[8]), 99)
		op.LeftDepth = clamp(int(o[9]), 99)
		op.RightDepth = clamp(int(o[10]), 99)
		op.LeftCurve = int(o[11] & 0x03)
		op.RightCurve = int(o[11]>>2) & 0x03
		op.RateScaling = int(o[12] & 0x07)
		op.Detune = clamp(int(o[12]>>3)&0x0F, 14)
		op.AmpModSens = int(o[13] & 0x03)
		op.VelocitySens = int(o[13]>>2) & 0x07
		op.OutputLevel = clamp(int(o[14]), 99)
		op.FixedFreq = o[15]&0x01 == 1
		op.Coarse = int(o[15]>>1) & 0x1F
		op.Fine = clamp(int(o[16]), 99)
	}
	for k := range 4 {
		p.PitchEGRates[k] = clamp(int(d[102+k]), 99)
		p.PitchEGLevels[k] = clamp(int(d[106+k]), 99)
	}
	p.Algorithm = int(d[110] & 0x1F)
	p.Feedback = int(d[111] & 0x07)
	p.OscKeySync = (d[111]>>3)&0x01 == 1
	p.LFOSpeed = clamp(int(d[112]), 99)
	p.LFODelay = clamp(int(d[113]), 99)
	p.LFOPitchDepth = clamp(int(d[114]), 99)
	p.LFOAmpDepth = clamp(int(d[115]), 99)
	p.LFOKeySync = d[116]&0x01 == 1
	p.LFOWaveform = clamp(int(d[116]>>1)&0x07, 5)
	p.PitchModSens = int(d[116]>>4) & 0x07
	p.Transpose = clamp(int(d[117]), 48)
	p.Name = decodeName(d[118:128])
	return p
}

// packVoice encodes a voice in the 128 byte packed format
func packVoice(p Patch) []byte {
	d := make([]byte, packedSize)
	for i := range NumOperators {
		o := d[i*17 : i*17+17]
		op := p.Ops[NumOperators-1-i]
		for k := range 4 {
			o[k] = byte(clamp(op.EGRates[k], 99))
			o[4+k] = byte(clamp(op.EGLevels[k], 99))
		}
		o[8] = byte(clamp(op.BreakPoint, 99))
		o[9] = byte(clamp(op.LeftDepth, 99))
		o[10] = byte(clamp(op.RightDepth, 99))
		o[11] = byte(clamp(op.LeftCurve, 3) | clamp(op.RightCurve, 3)<<2)
		o[12] = byte(clamp(op.RateScaling, 7) | clamp(op.Detune, 14)<<3)
		o[13] = byte(clamp(op.AmpModSens, 3) | clamp(op.VelocitySens, 7)<<2)
		o[14] = byte(clamp(op.OutputLevel, 99))
		fixed := 0
		if op.FixedFreq {
			fixed = 1
		}
		o[15] = byte(fixed | clamp(op.Coarse, 31)<<1)
		o[16] = byte(clamp(op.Fine, 99))
	}
	for k := range 4 {
		d[102+k] = byte(clamp(p.PitchEGRates[k], 99))
		d[106+k] = byte(clamp(p.PitchEGLevels[k], 99))
	}
	d[110] = byte(clamp(p.Algorithm, 31))
	sync := 0
	if p.OscKeySync {
		sync = 1
	}
	d[111] = byte(clamp(p.Feedback, 7) | sync<<3)
	d[112] = byte(clamp(p.LFOSpeed, 99))
	d[113] = byte(clamp(p.LFODelay, 99))
	d[114] = byte(clamp(p.LFOPitchDepth, 99))
	d[115] = byte(clamp(p.LFOAmpDepth, 99))
	lfoSync := 0
	if p.LFOKeySync {
		lfoSync = 1
	}
	d[116] = byte(lfoSync | clamp(p.LFOWaveform, 5)<<1 | clamp(p.PitchModSens, 7)<<4)
	d[117] = byte(clamp(p.Transpose, 48))
	copy(d[118:128], encodeName(p.Name))
	return d
}

// unpackSingle decodes a 155 byte unpacked voice (as used in single voice dumps)
func unpackSingle(d []byte) Patch {
	var p Patch
	for i := range NumOperators {
		o := d[i*21 : i*21+21]
		op := &p.Ops[NumOperators-1-i]
		for k := range 4 {
			op.EGRates[k] = clamp(int(o[k]), 99)
			op.EGLevels[k] = clamp(int(o[4+k]), 99)
		}
		op.BreakPoint = clamp(int(o[8]), 99)
		op.LeftDepth = clamp(int(o[9]), 99)
		op.RightDepth = clamp(int(o[10]), 99)
		op.LeftCurve = clamp(int(o[11]), 3)
		op.RightCurve = clamp(int(o[12]), 3)
		op.RateScaling = clamp(int(o[13]), 7)
		op.AmpModSens = clamp(int(o[14]), 3)
		op.VelocitySens = clamp(int(o[15]), 7)
		op.OutputLevel = clamp(int(o[16]), 99)
		op.FixedFreq = o[17] == 1
		op.Coarse = clamp(int(o[18]), 31)
		op.Fine = clamp(int(o[19]), 99)
		op.Detune = clamp(int(o[20]), 14)
	}
	for k := range 4 {
		p.PitchEGRates[k] = clamp(int(d[126+k]), 99)
		p.PitchEGLevels[k] = clamp(int(d[130+k]), 99)
	}
	p.Algorithm = clamp(int(d[134]), 31)
	p.Feedback = clamp(int(d[135]), 7)
	p.OscKeySync = d[136] == 1
	p.LFOSpeed = clamp(int(d[137]), 99)
	p.LFODelay = clamp(int(d[138]), 99)
	p.LFOPitchDepth = clamp(int(d[139]), 99)
	p.LFOAmpDepth = clamp(int(d[140]), 99)
	p.LFOKeySync = d[141] == 1
	p.LFOWaveform = clamp(int(d[142]), 5)
	p.PitchModSens = clamp(int(d[143]), 7)
	p.Transpose = clamp(int(d[144]), 48)
	p.Name = decodeName(d[145:155])
	return p
}

// packSingle encodes a voice in the 155 byte unpacked format
func packSingle(p Patch) []byte {
	d := make([]byte, unpackedSize)
	b := func(v bool) byte {
		if v {
			return 1
		}
		return 0
	}
	for i := range NumOperators {
		o := d[i*21 : i*21+21]
		op := p.Ops[NumOperators-1-i]
		for k := range 4 {
			o[k] = byte(clamp(op.EGRates[k], 99))
			o[4+k] = byte(clamp(op.EGLevels[k], 99))
		}
		o[8] = byte(clamp(op.BreakPoint, 99))
		o[9] = byte(clamp(op.LeftDepth, 99))
		o[10] = byte(clamp(op.RightDepth, 99))
		o[11] = byte(clamp(op.LeftCurve, 3))
		o[12] = byte(clamp(op.RightCurve, 3))
		o[13] = byte(clamp(op.RateScaling, 7))
		o[14] = byte(clamp(op.AmpModSens, 3))
		o[15] = byte(clamp(op.VelocitySens, 7))
		o[16] = byte(clamp(op.OutputLevel, 99))
		o[17] = b(op.FixedFreq)
		o[18] = byte(clamp(op.Coarse, 31))
		o[19] = byte(clamp(op.Fine, 99))
		o[20] = byte(clamp(op.Detune, 14))
	}
	for k := range 4 {
		d[126+k] = byte(clamp(p.PitchEGRates[k], 99))
		d[130+k] = byte(clamp(p.PitchEGLevels[k], 99))
	}
	d[134] = byte(clamp(p.Algorithm, 31))
	d[135] = byte(clamp(p.Feedback, 7))
	d[136] = b(p.OscKeySync)
	d[137] = byte(clamp(p.LFOSpeed, 99))
	d[138] = byte(clamp(p.LFODelay, 99))
	d[139] = byte(clamp(p.LFOPitchDepth, 99))
	d[140] = byte(clamp(p.LFOAmpDepth, 99))
	d[141] = b(p.LFOKeySync)
	d[142] = byte(clamp(p.LFOWaveform, 5))
	d[143] = byte(clamp(p.PitchModSens, 7))
	d[144] = byte(clamp(p.Transpose, 48))
	copy(d[145:155], encodeName(p.Name))
	return d
}

func decodeName(b []byte) string {
	r := make([]byte, len(b))
	for i, c := range b {
		if c < 32 || c > 126 {
			c = ' '
		}
		r[i] = c
	}
	return strings.TrimRight(string(r), " ")
}

func encodeName(name string) []byte {
	b := []byte(fmt.Sprintf("%-10s", name))[:10]
	for i, c := range b {
		if c < 32 || c > 126 {
			b[i] = ' '
		}
	}
	return b
}

// ParseSysEx parses a 32 voice bulk dump or a single voice dump (also several messages after each
// other) and returns the voices it contains
func ParseSysEx(data []byte) ([]Patch, error) {
	var patches []Patch

	for len(data) > 0 {
		// skip to the next message
		start := -1
		for i, b := range data {
			if b == sysexStart {
				start = i
				break
			}
		}
		if start < 0 {
			break
		}
		data = data[start:]

		if len(data) < 6 {
			return nil, ErrTruncated
		}
		if data[1] != yamahaID {
			return nil, ErrNotSysEx
		}

		var (
			format   = data[3]
			count    = int(data[4])<<7 | int(data[5])
			body     []byte
			msgBytes int
		)

		switch {
		case format == 9 && count == bulkVoices*packedSize:
			msgBytes = 6 + count + 2
		case format == 0 && count == unpackedSize:
			msgBytes = 6 + count + 2
		default:
			return nil, fmt.Errorf("%w (format %d, %d bytes)", ErrNotSysEx, format, count)
		}

		if len(data) < msgBytes {
			return nil, ErrTruncated
		}
		body = data[6 : 6+count]
		if checksum(body) != data[6+count] {
			return nil, ErrChecksum
		}

		if format == 9 {
			for v := range bulkVoices {
				patches = append(patches, unpackVoice(body[v*packedSize:(v+1)*packedSize]))
			}
		} else {
			patches = append(patches, unpackSingle(body))
		}

		data = data[msgBytes:]
	}

	if len(patches) == 0 {
		return nil, ErrNotSysEx
	}

	return patches, nil
}

// LoadSysEx reads a .syx file with a 32 voice bulk dump or single voice dumps
func LoadSysEx(path string) ([]Patch, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseSysEx(data)
}

// EncodeBank encodes 32 voices as a bulk dump (fewer voices are padded with init voices)
func EncodeBank(patches []Patch) []byte {
	body := make([]byte, 0, bulkVoices*packedSize)
	for v := range bulkVoices {
		p := InitVoice()
		if v < len(patches) {
			p = patches[v]
		}
		body = append(body, packVoice(p)...)
	}

	msg := []byte{sysexStart, yamahaID, 0x00, 9, 0x20, 0x00}
	msg = append(msg, body...)
	return append(msg, checksum(body), sysexEnd)
}

// EncodeSingle encodes one voice as a single voice dump
func EncodeSingle(p Patch) []byte {
	body := packSingle(p)
	msg := []byte{sysexStart, yamahaID, 0x00, 0, 0x01, 0x1B}
	msg = append(msg, body...)
	return append(msg, checksum(body), sysexEnd)
}
