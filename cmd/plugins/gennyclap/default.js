// A Genny script: render(p) makes a loop of notes, once for each seed (p.seed). Times are in steps (p.stepsPerBeat, 4 =
// sixteenth notes); a note is {pitch, start, duration, velocity, channel}. It can return the notes, or
// {stepsPerBeat, length, notes}: length (in steps) is the loop. Everything under genny.* of the psalm scripts is here
// (genny.rand, genny.sequence, genny.bucket, genny.euclidPattern, genny.mutate, genny.lsystem, ...), and scale(),
// chord(), pitch() and log(). Save this file and the plugin plays the new version at the next pass.

function render(p) {
	const key = scale("D minor")
	const notes = []
	const bars = 4

	// a melody that wanders over the scale: a Euclidean rhythm of 7 hits in 16 steps, a random walk of degrees
	const rhythm = genny.euclidPattern(16, 7)
	const walk = genny.rwalk(3, -2, 9, 2)
	for (let bar = 0; bar < bars; bar++) {
		for (let step = 0; step < 16; step++) {
			if (!rhythm[step]) continue
			const degree = Math.round(walk.next())
			notes.push({
				pitch: key.pitch(degree, 4),
				start: bar * 16 + step,
				duration: 1.5,
				velocity: step % 4 === 0 ? 0.9 : 0.65,
				channel: 0,
			})
		}
	}

	// a bass on the roots of a progression, on its own channel
	const roots = [1, 6, 4, 5]
	for (let bar = 0; bar < bars; bar++) {
		notes.push({pitch: key.pitch(roots[bar], 2), start: bar * 16, duration: 14, velocity: 0.8, channel: 1})
	}

	return {stepsPerBeat: p.stepsPerBeat, length: bars * 16, notes}
}
