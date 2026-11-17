// the bell that states the motif (the opening of the head: G B flat | C E flat) once in every section, where its changes begin on I vi: faintly as the light comes in the
// intro, with the flute in the head, to open the solos and the break, high at the climax of the head out, and in the tag, where it is the last thing heard before the
// chord rings
clock("1/16")
const part = shared.register("bells", { legato: 1.6, vel: 0.7 }, day => shared.motifAt(day, shared.statements[day.number][0], shared.statements[day.number][1], false))
function section(name) { part.section(name) }
function trigger(ev) { part.trigger(song, note, ev) }
