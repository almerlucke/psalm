#ifndef GENNY_SHIM_H
#define GENNY_SHIM_H

// one event the Go side hands the shim: a note on or off at a position in beats. The off of a note has the id of
// its on, the shim lets go of the key it started (the parameters can have changed in between)
typedef struct {
	double beat;
	double velocity;
	int key;
	int channel;
	int id;
	int on;
} genny_event;

// a parameter of the plugin, as the Go side describes it
typedef struct {
	int id;
	int stepped;
	double min, max, def;
	char name[64];
} genny_param;

#endif
