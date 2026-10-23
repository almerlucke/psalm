package patcher

import "github.com/almerlucke/psalm"

// ArpLate plays the triggers of a tick, as the patcher does after every module
func ArpLate(a *Arp, tick psalm.CtrlTick) { a.late(tick) }

// LooperLayers says how many layers the looper of a module has and whether it records one
func LooperLayers(pt *Patcher, id string) (int, bool) { return pt.Module(id).(*Looper).Layers() }

// LooperScript is the script of a layer of the looper of a module
func LooperScript(pt *Patcher, id string, layer int) string {
	return pt.Module(id).(*Looper).Script(layer)
}

// LooperScriptAll is the script of all the layers of the looper of a module
func LooperScriptAll(pt *Patcher, id string) string { return pt.Module(id).(*Looper).ScriptAll() }
