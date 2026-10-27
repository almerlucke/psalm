// glue between Go and a CLAP plugin: loading a library, making an instance, its parameters, the events of a block and the processing of it. The Go side
// holds only the opaque pointers; buffers are passed in each call and never kept
#ifndef PSALM_CLAP_GLUE_H
#define PSALM_CLAP_GLUE_H

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

// events for the next block, in the order they come (they are sorted by time when it is processed)
void pi_note(pc_inst *inst, int on, uint32_t time, int key, double velocity, int32_t noteID, double tuning);
void pi_param(pc_inst *inst, uint32_t id, double value, uint32_t time);

// processes a block: the input channels (NULL for an instrument), the output channels (the second is filled from the first when the plugin has one channel)
int pi_process(pc_inst *inst, float *in0, float *in1, float *out0, float *out1, uint32_t frames);

#endif
