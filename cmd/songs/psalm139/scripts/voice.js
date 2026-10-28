// the voice: the recording of the psalm (psalm139_voice.wav), played once from the start of the Knit, to its end, as it is. It is a streamer: the file is read from the disk while it
// plays. The note is one of the sound, which plays to its end whatever the length of the note; it is played at the first step of the Knit
clock("1/16")
let armed = false
function section(name) { armed = name === "Knit" }
function trigger() {
  if (!armed) return
  armed = false
  note(36, 0.9, 1)
}
