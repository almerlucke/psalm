package midi

// Parser turns a MIDI byte stream into messages: it follows running status (the status byte left
// out when it repeats), takes the real time messages out wherever they come (even inside another
// message) and collects system exclusive messages up to their end (F7) or the next status
type Parser struct {
	running  byte   // the running status, 0 when there is none
	msg      []byte // the message being read
	need     int    // data bytes the message needs
	sysex    bool
	MaxSysEx int // the longest system exclusive message kept (longer ones are dropped), 0 is 64 KiB
}

// dataBytes returns how many data bytes follow a status byte (-1 for system exclusive)
func dataBytes(status byte) int {
	switch status & 0xF0 {
	case 0x80, 0x90, 0xA0, 0xB0, 0xE0:
		return 2
	case 0xC0, 0xD0:
		return 1
	}
	switch status {
	case 0xF0:
		return -1
	case 0xF1, 0xF3:
		return 1
	case 0xF2:
		return 2
	}
	return 0
}

// Feed parses bytes and calls emit for every complete message. The message is only valid during
// the call (copy it to keep it)
func (p *Parser) Feed(data []byte, emit func(Message)) {
	maxSysEx := p.MaxSysEx
	if maxSysEx <= 0 {
		maxSysEx = 1 << 16
	}
	for _, b := range data {
		switch {
		case b >= 0xF8:
			// real time: on its own, it doesn't touch what is being read
			if b != 0xF9 && b != 0xFD {
				emit(Message{b})
			}

		case b == 0xF7:
			if p.sysex {
				p.msg = append(p.msg, b)
				if len(p.msg) <= maxSysEx {
					emit(p.msg)
				}
				p.sysex = false
			}
			p.msg = p.msg[:0]
			p.need = 0

		case b >= 0x80:
			// a status: it ends an unfinished system exclusive and drops an unfinished message
			if p.sysex {
				p.sysex = false
			}
			p.msg = append(p.msg[:0], b)
			p.need = dataBytes(b)
			switch {
			case b < 0xF0:
				p.running = b
			default:
				p.running = 0 // system common cancels running status
			}
			switch {
			case p.need < 0:
				p.sysex = true
			case p.need == 0:
				if b >= 0xF0 && b != 0xF4 && b != 0xF5 {
					emit(p.msg)
				}
				p.msg = p.msg[:0]
			}

		default:
			// a data byte
			if p.sysex {
				if len(p.msg) <= maxSysEx {
					p.msg = append(p.msg, b)
				}
				continue
			}
			if len(p.msg) == 0 {
				if p.running == 0 {
					continue // no status to belong to
				}
				p.msg = append(p.msg, p.running)
				p.need = dataBytes(p.running)
			}
			p.msg = append(p.msg, b)
			if len(p.msg)-1 == p.need {
				emit(p.msg)
				p.msg = p.msg[:0]
			}
		}
	}
}
