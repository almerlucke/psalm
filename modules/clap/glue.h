// glue between Go and a CLAP plugin: loading a library, making an instance, its parameters, the events of a block and the processing of it. The Go side
// holds only the opaque pointers; buffers are passed in each call and never kept
#ifndef PSALM_CLAP_GLUE_H
#define PSALM_CLAP_GLUE_H

#include <stddef.h>
#include <stdint.h>

typedef struct pc_lib pc_lib;
typedef struct pc_inst pc_inst;

// a library: opened (and its entry initialised) once; err gets the reason when it fails
pc_lib *pc_open(const char *path, char *err, int errcap);
int pc_count(pc_lib *lib);
// the descriptor of plugin i: id, name, vendor and the features joined by spaces
int pc_desc(pc_lib *lib, int i, char *id, char *name, char *vendor, char *features, int cap);

// an instance of the plugin with this id, active and processing; blocks of up to maxFrames
pc_inst *pi_new(pc_lib *lib, const char *id, double sampleRate, uint32_t maxFrames, char *err, int errcap);
void pi_free(pc_inst *inst);
// the channels of the first audio input and output port (0 when it has none)
void pi_ports(pc_inst *inst, int *in, int *out);

int pi_param_count(pc_inst *inst);
int pi_param_info(pc_inst *inst, int i, uint32_t *id, char *name, char *module, int cap, double *min, double *max, double *def, uint32_t *flags);
double pi_param_value(pc_inst *inst, uint32_t id);
// the text the plugin shows for a value of a parameter ("440 Hz", "Classic"); 0 when it has none
int pi_param_text(pc_inst *inst, uint32_t id, double value, char *out, int cap);

// events for the next block, in the order they come (they are sorted by time when it is processed)
void pi_note(pc_inst *inst, int on, uint32_t time, int key, double velocity, int32_t noteID, double tuning);
void pi_param(pc_inst *inst, uint32_t id, double value, uint32_t time);

// the audio ports: how many of each, the channels and the name of one
void pi_port_counts(pc_inst *inst, int *in, int *out);
int pi_port_info(pc_inst *inst, int is_input, int i, int *channels, char *name, int cap);

// where the host is: the tempo, the position in beats, whether it plays (for the next blocks, until it is set again)
void pi_transport(pc_inst *inst, double tempo, double beats, int playing);
// the latency the plugin says it has, in samples
uint32_t pi_latency(pc_inst *inst);

// processes a block: the main input (NULL for an instrument) and the side chain (the plugin's second input; NULL when there is none), two channels each. The sound of
// every output port is kept in the instance: pi_out gives a channel of it (valid until the next block); a port of one channel is also the second
int pi_process(pc_inst *inst, float *in0, float *in1, float *sc0, float *sc1, uint32_t frames);
float *pi_out(pc_inst *inst, int port, int channel);

// the presets of a library, found by its preset discovery (the plugins' own: files it declares places for, and presets that are in the plugin itself)
typedef struct {
	char name[256];
	char plugin[256]; // the id of the plugin it is for ("" for any of the library)
	char creator[256];
	char features[256]; // separated by 0x1f
	char description[512];
	uint32_t kind;       // 0 a file, 1 in the plugin
	char location[1024]; // the file, or empty
	char loadkey[256];
	uint32_t flags;
} pc_preset;

// crawls the places the library declares; returns how many presets it found (0 when it has no preset discovery)
int pc_scan_presets(pc_lib *lib);
int pc_preset_get(pc_lib *lib, int i, pc_preset *out);
// loads one into the instance; 0 when the plugin has no preset loading or says it failed
int pi_load_preset(pc_inst *inst, uint32_t kind, const char *location, const char *loadkey);

// the state of an instance (what the plugin keeps of its sound), to keep and to give back; the data is made with malloc: free it with pc_free
int pi_state_save(pc_inst *inst, uint8_t **out, size_t *len);
int pi_state_load(pc_inst *inst, const uint8_t *data, size_t len);
void pc_free(void *p);

#endif
