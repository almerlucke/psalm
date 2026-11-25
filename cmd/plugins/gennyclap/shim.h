#ifndef GENNY_SHIM_H
#define GENNY_SHIM_H

// one event the Go side hands the shim: a note on or off at a position in beats
typedef struct {
	double beat;
	double velocity;
	int key;
	int on;
} genny_event;

#endif
