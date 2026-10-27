#include "glue.h"

#include <dlfcn.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "clap/clap.h"

#define MAX_EVENTS 4096

struct pc_lib {
	void *handle;
	const clap_plugin_entry_t *entry;
	const clap_plugin_factory_t *factory;
};

typedef union {
	clap_event_header_t h;
	clap_event_note_t note;
	clap_event_note_expression_t expr;
	clap_event_param_value_t param;
} pc_event;

struct pc_inst {
	clap_host_t host;
	pc_lib *lib;
	const clap_plugin_t *plugin;
	const clap_plugin_params_t *params;
	int in_ch, out_ch;
	clap_input_events_t in_events;
	clap_output_events_t out_events;
	int n;
	pc_event ev[MAX_EVENTS];
	int started, activated;
};

static void seterr(char *err, int cap, const char *msg) {
	if (err && cap > 0) {
		snprintf(err, cap, "%s", msg);
	}
}

// --- the host the plugins see: it offers nothing but its name ---

static const void *host_get_extension(const clap_host_t *host, const char *id) { return NULL; }
static void host_nothing(const clap_host_t *host) {}

static uint32_t in_size(const clap_input_events_t *list) { return ((pc_inst *)list->ctx)->n; }
static const clap_event_header_t *in_get(const clap_input_events_t *list, uint32_t i) {
	pc_inst *inst = list->ctx;
	return i < (uint32_t)inst->n ? &inst->ev[i].h : NULL;
}
static bool out_push(const clap_output_events_t *list, const clap_event_header_t *event) { return true; }

pc_lib *pc_open(const char *path, char *err, int errcap) {
	void *h = dlopen(path, RTLD_NOW | RTLD_LOCAL);
	if (!h) {
		seterr(err, errcap, dlerror());
		return NULL;
	}
	const clap_plugin_entry_t *entry = dlsym(h, "clap_entry");
	if (!entry) {
		seterr(err, errcap, "no clap_entry");
		dlclose(h);
		return NULL;
	}
	if (!entry->init(path)) {
		seterr(err, errcap, "the entry would not initialise");
		dlclose(h);
		return NULL;
	}
	const clap_plugin_factory_t *f = entry->get_factory(CLAP_PLUGIN_FACTORY_ID);
	if (!f) {
		seterr(err, errcap, "no plugin factory");
		entry->deinit();
		dlclose(h);
		return NULL;
	}
	pc_lib *lib = calloc(1, sizeof *lib);
	lib->handle = h;
	lib->entry = entry;
	lib->factory = f;
	return lib;
}

int pc_count(pc_lib *lib) { return (int)lib->factory->get_plugin_count(lib->factory); }

int pc_desc(pc_lib *lib, int i, char *id, char *name, char *vendor, char *features, int cap) {
	const clap_plugin_descriptor_t *d = lib->factory->get_plugin_descriptor(lib->factory, (uint32_t)i);
	if (!d) {
		return 0;
	}
	snprintf(id, cap, "%s", d->id ? d->id : "");
	snprintf(name, cap, "%s", d->name ? d->name : "");
	snprintf(vendor, cap, "%s", d->vendor ? d->vendor : "");
	features[0] = 0;
	if (d->features) {
		for (const char *const *f = d->features; *f; f++) {
			size_t used = strlen(features);
			snprintf(features + used, cap - used, "%s%s", used ? "\x1f" : "", *f);
		}
	}
	return 1;
}

static int port_channels(pc_inst *inst, int is_input) {
	const clap_plugin_audio_ports_t *ports = inst->plugin->get_extension(inst->plugin, CLAP_EXT_AUDIO_PORTS);
	if (!ports || ports->count(inst->plugin, is_input) == 0) {
		return 0;
	}
	clap_audio_port_info_t info;
	if (!ports->get(inst->plugin, 0, is_input, &info)) {
		return 0;
	}
	return (int)info.channel_count;
}

pc_inst *pi_new(pc_lib *lib, const char *id, double sampleRate, uint32_t maxFrames, char *err, int errcap) {
	pc_inst *inst = calloc(1, sizeof *inst);
	inst->lib = lib;
	inst->host = (clap_host_t){.clap_version = CLAP_VERSION_INIT,
	                           .host_data = inst,
	                           .name = "psalm",
	                           .vendor = "psalm",
	                           .url = "",
	                           .version = "1",
	                           .get_extension = host_get_extension,
	                           .request_restart = host_nothing,
	                           .request_process = host_nothing,
	                           .request_callback = host_nothing};
	inst->in_events = (clap_input_events_t){.ctx = inst, .size = in_size, .get = in_get};
	inst->out_events = (clap_output_events_t){.ctx = inst, .try_push = out_push};
	inst->plugin = lib->factory->create_plugin(lib->factory, &inst->host, id);
	if (!inst->plugin) {
		seterr(err, errcap, "the factory would not make the plugin");
		free(inst);
		return NULL;
	}
	if (!inst->plugin->init(inst->plugin)) {
		seterr(err, errcap, "the plugin would not initialise");
		inst->plugin->destroy(inst->plugin);
		free(inst);
		return NULL;
	}
	inst->params = inst->plugin->get_extension(inst->plugin, CLAP_EXT_PARAMS);
	inst->in_ch = port_channels(inst, 1);
	inst->out_ch = port_channels(inst, 0);
	if (inst->in_ch > 2 || inst->out_ch > 2 || inst->out_ch == 0) {
		seterr(err, errcap, "the plugin has no stereo or mono output");
		inst->plugin->destroy(inst->plugin);
		free(inst);
		return NULL;
	}
	if (!inst->plugin->activate(inst->plugin, sampleRate, 1, maxFrames)) {
		seterr(err, errcap, "the plugin would not activate");
		inst->plugin->destroy(inst->plugin);
		free(inst);
		return NULL;
	}
	inst->activated = 1;
	inst->started = inst->plugin->start_processing(inst->plugin);
	if (!inst->started) {
		seterr(err, errcap, "the plugin would not start processing");
		inst->plugin->deactivate(inst->plugin);
		inst->plugin->destroy(inst->plugin);
		free(inst);
		return NULL;
	}
	return inst;
}

void pi_free(pc_inst *inst) {
	if (!inst) {
		return;
	}
	if (inst->started) {
		inst->plugin->stop_processing(inst->plugin);
	}
	if (inst->activated) {
		inst->plugin->deactivate(inst->plugin);
	}
	inst->plugin->destroy(inst->plugin);
	free(inst);
}

void pi_ports(pc_inst *inst, int *in, int *out) {
	*in = inst->in_ch;
	*out = inst->out_ch;
}

int pi_param_count(pc_inst *inst) { return inst->params ? (int)inst->params->count(inst->plugin) : 0; }

int pi_param_info(pc_inst *inst, int i, uint32_t *id, char *name, char *module, int cap, double *min, double *max, double *def, uint32_t *flags) {
	clap_param_info_t info;
	if (!inst->params || !inst->params->get_info(inst->plugin, (uint32_t)i, &info)) {
		return 0;
	}
	*id = info.id;
	snprintf(name, cap, "%s", info.name);
	snprintf(module, cap, "%s", info.module);
	*min = info.min_value;
	*max = info.max_value;
	*def = info.default_value;
	*flags = info.flags;
	return 1;
}

double pi_param_value(pc_inst *inst, uint32_t id) {
	double v = 0;
	if (inst->params && inst->params->get_value(inst->plugin, id, &v)) {
		return v;
	}
	return 0;
}

static pc_event *push(pc_inst *inst, uint16_t type, uint32_t size, uint32_t time) {
	if (inst->n >= MAX_EVENTS) {
		return NULL;
	}
	pc_event *e = &inst->ev[inst->n++];
	memset(e, 0, sizeof *e);
	e->h.size = size;
	e->h.time = time;
	e->h.space_id = CLAP_CORE_EVENT_SPACE_ID;
	e->h.type = type;
	e->h.flags = 0;
	return e;
}

void pi_note(pc_inst *inst, int on, uint32_t time, int key, double velocity, int32_t noteID, double tuning) {
	pc_event *e = push(inst, on ? CLAP_EVENT_NOTE_ON : CLAP_EVENT_NOTE_OFF, sizeof(clap_event_note_t), time);
	if (!e) {
		return;
	}
	e->note.note_id = noteID;
	e->note.port_index = 0;
	e->note.channel = 0;
	e->note.key = (int16_t)key;
	e->note.velocity = velocity;
	if (on && tuning != 0) { // a pitch between the keys: the tuning of the note, in semitones
		pc_event *t = push(inst, CLAP_EVENT_NOTE_EXPRESSION, sizeof(clap_event_note_expression_t), time);
		if (t) {
			t->expr.expression_id = CLAP_NOTE_EXPRESSION_TUNING;
			t->expr.note_id = noteID;
			t->expr.port_index = 0;
			t->expr.channel = 0;
			t->expr.key = (int16_t)key;
			t->expr.value = tuning;
		}
	}
}

void pi_param(pc_inst *inst, uint32_t id, double value, uint32_t time) {
	pc_event *e = push(inst, CLAP_EVENT_PARAM_VALUE, sizeof(clap_event_param_value_t), time);
	if (!e) {
		return;
	}
	e->param.param_id = id;
	e->param.cookie = NULL;
	e->param.note_id = -1;
	e->param.port_index = -1;
	e->param.channel = -1;
	e->param.key = -1;
	e->param.value = value;
}

// the events in the order of their time (a note and its tuning stay together: the sort is stable)
static void sort_events(pc_inst *inst) {
	for (int i = 1; i < inst->n; i++) {
		pc_event e = inst->ev[i];
		int j = i - 1;
		while (j >= 0 && inst->ev[j].h.time > e.h.time) {
			inst->ev[j + 1] = inst->ev[j];
			j--;
		}
		inst->ev[j + 1] = e;
	}
}

int pi_process(pc_inst *inst, float *in0, float *in1, float *out0, float *out1, uint32_t frames) {
	float *inch[2] = {in0, in1};
	float *outch[2] = {out0, out1};
	clap_audio_buffer_t in = {.data32 = inch, .data64 = NULL, .channel_count = (uint32_t)inst->in_ch, .latency = 0, .constant_mask = 0};
	clap_audio_buffer_t out = {.data32 = outch, .data64 = NULL, .channel_count = (uint32_t)inst->out_ch, .latency = 0, .constant_mask = 0};
	clap_process_t pr = {.steady_time = -1,
	                     .frames_count = frames,
	                     .transport = NULL,
	                     .audio_inputs = (inst->in_ch > 0 && in0) ? &in : NULL,
	                     .audio_outputs = &out,
	                     .audio_inputs_count = (inst->in_ch > 0 && in0) ? 1 : 0,
	                     .audio_outputs_count = 1,
	                     .in_events = &inst->in_events,
	                     .out_events = &inst->out_events};
	sort_events(inst);
	clap_process_status st = inst->plugin->process(inst->plugin, &pr);
	inst->n = 0;
	if (inst->out_ch == 1) {
		memcpy(out1, out0, frames * sizeof(float));
	}
	return st != CLAP_PROCESS_ERROR;
}
