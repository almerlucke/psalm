package patcher

import "github.com/almerlucke/psalm"

// ArpLate plays the triggers of a tick, as the patcher does after every module
func ArpLate(a *Arp, tick psalm.CtrlTick) { a.late(tick) }
