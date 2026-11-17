// a marimba answers the motif of the bell (see bells.js): two bars after every statement, a figure of its own for every section (shared.answers in score.js: the light
// rising, the firmament low, the waters flowing, the creatures calling, the lights sparkling, mankind tall, the rest falling), on ii V7
clock("1/16")
const part = shared.register("marimba", { legato: 1, vel: 0.7 }, day => shared.motifAt(day, shared.statements[day.number][0], shared.statements[day.number][1], true))
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note, ev) }
