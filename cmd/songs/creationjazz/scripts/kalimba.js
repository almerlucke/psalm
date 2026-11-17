// a kalimba answers the motif an octave over the marimba, in the break and the solo of the heavens and the head out: the answer gets more abundant as the piece goes to its climax
// (see shared.statements in score.js)
clock("1/16")
const part = shared.register("kalimba", { legato: 1, vel: 0.7 }, day => (day.number >= 4 && day.number <= 6 ? shared.motifAt(day, shared.statements[day.number][0].map(([bar, shift]) => [bar, shift + 12]), shared.statements[day.number][1] * 0.8, true) : null))
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note, ev) }
