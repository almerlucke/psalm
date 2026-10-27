// glue between Go and a VST3 plugin (C++ inside, C outside): loading a bundle, making a plugin (its component and its controller), its parameters, the events and
// parameter changes of a block, the processing of it, its state and its window. The Go side holds only the opaque pointers; buffers are passed in each call and
// never kept
#ifndef PSALM_VST3_GLUE_H
#define PSALM_VST3_GLUE_H

#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct v3_lib v3_lib;
typedef struct v3_inst v3_inst;

// the main thread of the plugins (see the Go side): called once by the thread that is it, then v3_pump over and over (the timers, descriptors and windows of the
// plugins); v3_wake gets it out of a wait, v3_is_main says whether the caller is it
void v3_set_main(void);
int v3_is_main(void);
void v3_wake(void);
void v3_pump(int timeout_ms);

// a bundle (the folder of the plugin, "Name.vst3"): opened once; err gets the reason when it fails
v3_lib *v3_open(const char *bundle, char *err, int errcap);
int v3_class_count(v3_lib *lib);
// the audio module class i: its class id (32 hex digits), name, vendor and sub categories ("Instrument|Synth", "Fx|Reverb"); 0 when it is another kind of class
int v3_class(v3_lib *lib, int i, char *cid, char *name, char *vendor, char *categories, int cap);

// a plugin of the class: its component and its controller, connected, active and processing; blocks of up to maxFrames
v3_inst *v3_new(v3_lib *lib, const char *cid, double sampleRate, uint32_t maxFrames, char *err, int errcap);
void v3_free(v3_inst *inst);
// the signal that crashed the plugin (SIGSEGV, SIGABRT ...), 0 when it is well
int v3_dead(v3_inst *inst);

// the audio buses: how many of each, the channels and the name of one
void v3_bus_counts(v3_inst *inst, int *in, int *out);
int v3_bus_info(v3_inst *inst, int is_input, int i, int *channels, char *name, int cap);

// the parameters: the id, title, unit, how many steps (0 continuous), the default and the flags (1 can automate, 2 read only, 16 hidden, 32768 program change, 65536 bypass)
int v3_param_count(v3_inst *inst);
int v3_param_info(v3_inst *inst, int i, uint32_t *id, char *title, char *units, int cap, int *steps, double *def, int *flags, int *unit);
double v3_param_value(v3_inst *inst, uint32_t id);
// the text the plugin shows for a value of a parameter (0 to 1); 0 when it has none
int v3_param_text(v3_inst *inst, uint32_t id, double value, char *out, int cap);

// events for the next block: a note (pitch a MIDI key, the tuning in cents above it) and a parameter value (0 to 1) at a sample of the block
void v3_note(v3_inst *inst, int on, uint32_t offset, int pitch, double velocity, int32_t noteID, double tuningCents);
void v3_param(v3_inst *inst, uint32_t id, double value, uint32_t offset);
// a value set on the controller now, on the main thread
void v3_param_set_now(v3_inst *inst, uint32_t id, double value);

// where the host is: the tempo, the position in beats, whether it plays
void v3_transport(v3_inst *inst, double tempo, double beats, int playing);
uint32_t v3_latency(v3_inst *inst);

// processes a block: the main input (NULL for an instrument) and the side chain, two channels each. The sound of every output bus is kept in the instance: v3_out gives a
// channel of it (valid until the next block)
int v3_process(v3_inst *inst, float *in0, float *in1, float *sc0, float *sc1, uint32_t frames);
float *v3_out(v3_inst *inst, int bus, int channel);

// what the plugin did (the controller, a knob turned in its window) since this was last asked: the ids and the values, up to max; returns how many
int v3_take_changes(v3_inst *inst, uint32_t *ids, double *values, int max);
// the flags of what the plugin asked to restart (parameter values, titles, latency changed ...) since this was last asked
int v3_take_restart(v3_inst *inst);

// the state: the component's and the controller's, made with malloc (free them with v3_free_bytes)
int v3_state_save(v3_inst *inst, uint8_t **comp, size_t *complen, uint8_t **ctrl, size_t *ctrllen);
int v3_state_load(v3_inst *inst, const uint8_t *comp, size_t complen, const uint8_t *ctrl, size_t ctrllen);
void v3_free_bytes(void *p);

// the name of a unit (a group of parameters) with the units it is in, "Oscillators / Osc 1"; 0 for the root and for a unit it does not know
int v3_unit_name(v3_inst *inst, int unit, char *name, int cap);

// the programs of the plugin (its own list of sounds): how many, the parameter that selects one and its steps; the name of one (0 when it has none)
int v3_programs(v3_inst *inst, uint32_t *param, int *steps);
int v3_program_name(v3_inst *inst, int index, char *name, int cap);

// the window of the plugin (X11, embedded in a window of ours), shown and hidden on the main thread
int v3_gui_available(v3_inst *inst);
int v3_gui_show(v3_inst *inst);
void v3_gui_hide(v3_inst *inst);
int v3_gui_visible(v3_inst *inst);

#ifdef __cplusplus
}
#endif

#endif
