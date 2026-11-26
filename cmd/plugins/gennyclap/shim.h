#ifndef GENNY_SHIM_H
#define GENNY_SHIM_H

// one event the Go side hands the shim: a note on or off at a position in beats. The off of a note has the id of
// its on, the shim lets go of the key it started (the parameters can have changed in between)
typedef struct {
	double beat;
	double velocity;
	double tune; // semitones from the key (a note on only): a pitch between two keys
	int key;
	int channel;
	int id;
	int voice; // the voice of the key trigger that made the note, 0 for the piece of the transport
	int on;    // 1 a note on, 0 a note off, 2 not a note: let go of the notes that the voice still has on
} genny_event;

// a parameter of the plugin, as the Go side describes it
typedef struct {
	int id;
	int stepped;
	double min, max, def;
	char name[64];
} genny_param;

// genny_host_log sends a message to the log of the host of a plugin (a severity of CLAP: 1 info, 2 warning, 3 error); any thread
void genny_host_log(int handle, int severity, const char *msg);

#endif
