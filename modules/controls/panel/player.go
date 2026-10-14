package panel

// Player starts and stops the sound of a patch: a psalm.Psalm playing through an audio engine is
// one. Its methods are called from the page's goroutines
type Player interface {
	AudioStart() error
	AudioStop() error
	Running() bool
}

// restarter is a player that can also go back to the start (a psalm.Psalm can)
type restarter interface {
	Restart()
}

// live is a player that may render offline instead (a psalm.Psalm without an audio engine)
type live interface {
	Player
	Live() bool
}

// SetPlayer gives the page a start and stop button for a player, and a restart button when the
// player has a Restart method too (back to the start). A panel made for a psalm.Psalm that plays
// through an audio engine has them already; nil takes the buttons away.
//
// Stopping stands the whole patch still where it is (the song, the sequencers, a reverb's tail) and
// starting goes on from there, the notes that were playing let go (they ring out their release).
// While it stands still the panel's changes wait: they are applied when it plays again
func (pn *Panel) SetPlayer(pl Player) *Panel {
	pn.mu.Lock()
	defer pn.mu.Unlock()
	pn.player = pl
	return pn
}

// Player returns the player the page starts and stops, nil for none
func (pn *Panel) Player() Player {
	pn.mu.Lock()
	defer pn.mu.Unlock()
	return pn.player
}

// playerOf returns the player of a patch: the patch itself when it plays through an audio engine
func playerOf(p any) Player {
	if l, ok := p.(live); ok && l.Live() {
		return l
	}
	return nil
}
