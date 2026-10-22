package patcher

import "github.com/almerlucke/psalm"

// ArpLate plays the triggers of a tick, as the patcher does after every module
func ArpLate(a *Arp, tick psalm.CtrlTick) { a.late(tick) }

// LooperLayers says how many layers the looper of a module has and whether it records one
func LooperLayers(pt *Patcher, id string) (int, bool) { return pt.Module(id).(*Looper).Layers() }
