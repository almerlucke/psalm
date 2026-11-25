// The CLAP side of Genny: a note effect that plays what the generators of genny made, in time with the transport of
// the host. All it does is turn beats into sample offsets and keep track of the notes that sound; the notes, the
// parameters and the state come from Go (see main.go)
#include <math.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include <clap/clap.h>

#include "shim.h"
#include "_cgo_export.h"

#define MAX_EVENTS 512
#define MAX_ACTIVE 512

typedef struct {
	int id;
	int key;
	int channel;
} active_note;

typedef struct {
	clap_plugin_t plugin;
	const clap_host_t *host;
	int handle;
	double sample_rate;
	double next_beat; // where the last block ended, to find a jump of the transport
	int has_position;
	active_note active[MAX_ACTIVE];
	int num_active;
} genny_plugin;

static const clap_plugin_descriptor_t descriptor = {
	.clap_version = CLAP_VERSION_INIT,
	.id = "nl.almerlucke.psalm.genny",
	.name = "Genny",
	.vendor = "psalm",
	.url = "",
	.manual_url = "",
	.support_url = "",
	.version = "0.2.0",
	.description = "Notes from the generators of genny, from a spec file",
	.features = (const char *[]){CLAP_PLUGIN_FEATURE_NOTE_EFFECT, NULL},
};

static void push_note(const clap_output_events_t *out, uint32_t time, int on, int id, int key, int channel, double velocity) {
	clap_event_note_t e;
	memset(&e, 0, sizeof e);
	e.header.size = sizeof e;
	e.header.time = time;
	e.header.space_id = CLAP_CORE_EVENT_SPACE_ID;
	e.header.type = on ? CLAP_EVENT_NOTE_ON : CLAP_EVENT_NOTE_OFF;
	e.header.flags = 0;
	e.note_id = id;
	e.port_index = 0;
	e.channel = channel;
	e.key = key;
	e.velocity = velocity;
	out->try_push(out, &e.header);
}

static void release_all(genny_plugin *p, const clap_output_events_t *out, uint32_t time) {
	for (int i = 0; i < p->num_active; i++) {
		push_note(out, time, 0, p->active[i].id, p->active[i].key, p->active[i].channel, 0);
	}
	p->num_active = 0;
}

static void apply_param_events(genny_plugin *p, const clap_input_events_t *in) {
	if (!in) {
		return;
	}
	uint32_t n = in->size(in);
	for (uint32_t i = 0; i < n; i++) {
		const clap_event_header_t *h = in->get(in, i);
		if (h->space_id == CLAP_CORE_EVENT_SPACE_ID && h->type == CLAP_EVENT_PARAM_VALUE) {
			const clap_event_param_value_t *e = (const clap_event_param_value_t *)h;
			genny_param_set(p->handle, (int)e->param_id, e->value);
		}
	}
}

static bool plugin_init(const clap_plugin_t *plugin) {
	genny_plugin *p = plugin->plugin_data;
	p->handle = genny_new();
	return p->handle >= 0;
}

static void plugin_destroy(const clap_plugin_t *plugin) {
	genny_plugin *p = plugin->plugin_data;
	genny_free(p->handle);
	free(p);
}

static bool plugin_activate(const clap_plugin_t *plugin, double sample_rate, uint32_t min_frames, uint32_t max_frames) {
	genny_plugin *p = plugin->plugin_data;
	p->sample_rate = sample_rate;
	p->has_position = 0;
	genny_start(p->handle);
	return true;
}

static void plugin_deactivate(const clap_plugin_t *plugin) {
	genny_plugin *p = plugin->plugin_data;
	genny_stop(p->handle);
}

static bool plugin_start_processing(const clap_plugin_t *plugin) { return true; }
static void plugin_stop_processing(const clap_plugin_t *plugin) {}

static void plugin_reset(const clap_plugin_t *plugin) {
	genny_plugin *p = plugin->plugin_data;
	p->has_position = 0;
	p->num_active = 0;
}

static clap_process_status plugin_process(const clap_plugin_t *plugin, const clap_process_t *process) {
	genny_plugin *p = plugin->plugin_data;
	const clap_event_transport_t *tr = process->transport;
	uint32_t frames = process->frames_count;

	apply_param_events(p, process->in_events);

	int playing = tr && (tr->flags & CLAP_TRANSPORT_IS_PLAYING) && (tr->flags & CLAP_TRANSPORT_HAS_BEATS_TIMELINE) &&
	              (tr->flags & CLAP_TRANSPORT_HAS_TEMPO);
	if (!playing) {
		release_all(p, process->out_events, 0);
		p->has_position = 0;
		return CLAP_PROCESS_CONTINUE;
	}

	double from = (double)tr->song_pos_beats / (double)CLAP_BEATTIME_FACTOR;
	double per_sample = tr->tempo / (60.0 * p->sample_rate); // beats per sample
	double to = from + per_sample * frames;

	// the transport jumped (a loop, a click in the ruler): what sounds is let go
	if (p->has_position && fabs(from - p->next_beat) > 1e-4) {
		release_all(p, process->out_events, 0);
	}
	p->next_beat = to;
	p->has_position = 1;

	genny_event events[MAX_EVENTS];
	int n = genny_collect(p->handle, from, to, events, MAX_EVENTS);
	for (int i = 0; i < n; i++) {
		double offset = (events[i].beat - from) / per_sample;
		uint32_t time = offset < 0 ? 0 : (offset >= frames ? frames - 1 : (uint32_t)offset);
		if (events[i].on) {
			if (p->num_active < MAX_ACTIVE) {
				p->active[p->num_active++] = (active_note){events[i].id, events[i].key, events[i].channel};
				push_note(process->out_events, time, 1, events[i].id, events[i].key, events[i].channel, events[i].velocity);
			}
			continue;
		}
		for (int j = 0; j < p->num_active; j++) {
			if (p->active[j].id == events[i].id) {
				push_note(process->out_events, time, 0, p->active[j].id, p->active[j].key, p->active[j].channel, 0);
				p->active[j] = p->active[--p->num_active];
				break;
			}
		}
	}
	return CLAP_PROCESS_CONTINUE;
}

// the note port: one output

static uint32_t note_ports_count(const clap_plugin_t *plugin, bool is_input) { return is_input ? 0 : 1; }

static bool note_ports_get(const clap_plugin_t *plugin, uint32_t index, bool is_input, clap_note_port_info_t *info) {
	if (is_input || index != 0) {
		return false;
	}
	info->id = 0;
	info->supported_dialects = CLAP_NOTE_DIALECT_CLAP;
	info->preferred_dialect = CLAP_NOTE_DIALECT_CLAP;
	strncpy(info->name, "Notes", sizeof info->name);
	return true;
}

static const clap_plugin_note_ports_t note_ports = {.count = note_ports_count, .get = note_ports_get};

// the parameters

static uint32_t params_count(const clap_plugin_t *plugin) { return (uint32_t)genny_param_count(); }

static bool params_get_info(const clap_plugin_t *plugin, uint32_t index, clap_param_info_t *info) {
	genny_param g;
	if (!genny_param_info((int)index, &g)) {
		return false;
	}
	memset(info, 0, sizeof *info);
	info->id = (clap_id)g.id;
	info->flags = CLAP_PARAM_IS_AUTOMATABLE | (g.stepped ? CLAP_PARAM_IS_STEPPED : 0);
	info->cookie = NULL;
	strncpy(info->name, g.name, sizeof info->name - 1);
	info->module[0] = 0;
	info->min_value = g.min;
	info->max_value = g.max;
	info->default_value = g.def;
	return true;
}

static bool params_get_value(const clap_plugin_t *plugin, clap_id id, double *value) {
	genny_plugin *p = plugin->plugin_data;
	*value = genny_param_get(p->handle, (int)id);
	return true;
}

static bool params_value_to_text(const clap_plugin_t *plugin, clap_id id, double value, char *out, uint32_t size) {
	genny_param g;
	if (!genny_param_info((int)id, &g)) {
		return false;
	}
	if (g.stepped) {
		snprintf(out, size, "%d", (int)lround(value));
	} else {
		snprintf(out, size, "%.2f", value);
	}
	return true;
}

static bool params_text_to_value(const clap_plugin_t *plugin, clap_id id, const char *text, double *value) {
	char *end;
	double v = strtod(text, &end);
	if (end == text) {
		return false;
	}
	*value = v;
	return true;
}

static void params_flush(const clap_plugin_t *plugin, const clap_input_events_t *in, const clap_output_events_t *out) {
	apply_param_events(plugin->plugin_data, in);
}

static const clap_plugin_params_t params = {
	.count = params_count,
	.get_info = params_get_info,
	.get_value = params_get_value,
	.value_to_text = params_value_to_text,
	.text_to_value = params_text_to_value,
	.flush = params_flush,
};

// the state: the parameters and the spec, in JSON

static bool state_save(const clap_plugin_t *plugin, const clap_ostream_t *stream) {
	genny_plugin *p = plugin->plugin_data;
	int size = genny_state_write(p->handle, NULL, 0);
	char *buf = malloc(size + 1);
	if (!buf) {
		return false;
	}
	size = genny_state_write(p->handle, buf, size);
	int written = 0;
	while (written < size) {
		int64_t w = stream->write(stream, buf + written, (uint64_t)(size - written));
		if (w <= 0) {
			free(buf);
			return false;
		}
		written += (int)w;
	}
	free(buf);
	return true;
}

static bool state_load(const clap_plugin_t *plugin, const clap_istream_t *stream) {
	genny_plugin *p = plugin->plugin_data;
	int cap = 1 << 16, len = 0;
	char *buf = malloc(cap);
	if (!buf) {
		return false;
	}
	for (;;) {
		if (len == cap) {
			cap *= 2;
			char *bigger = realloc(buf, cap);
			if (!bigger) {
				free(buf);
				return false;
			}
			buf = bigger;
		}
		int64_t r = stream->read(stream, buf + len, (uint64_t)(cap - len));
		if (r < 0) {
			free(buf);
			return false;
		}
		if (r == 0) {
			break;
		}
		len += (int)r;
	}
	genny_state_read(p->handle, buf, len);
	free(buf);
	return true;
}

static const clap_plugin_state_t state = {.save = state_save, .load = state_load};

static const void *plugin_get_extension(const clap_plugin_t *plugin, const char *id) {
	if (!strcmp(id, CLAP_EXT_NOTE_PORTS)) {
		return &note_ports;
	}
	if (!strcmp(id, CLAP_EXT_PARAMS)) {
		return &params;
	}
	if (!strcmp(id, CLAP_EXT_STATE)) {
		return &state;
	}
	return NULL;
}

static void plugin_on_main_thread(const clap_plugin_t *plugin) {}

static uint32_t factory_count(const clap_plugin_factory_t *factory) { return 1; }

static const clap_plugin_descriptor_t *factory_descriptor(const clap_plugin_factory_t *factory, uint32_t index) {
	return index == 0 ? &descriptor : NULL;
}

static const clap_plugin_t *factory_create(const clap_plugin_factory_t *factory, const clap_host_t *host, const char *plugin_id) {
	if (strcmp(plugin_id, descriptor.id)) {
		return NULL;
	}
	genny_plugin *p = calloc(1, sizeof *p);
	p->host = host;
	p->plugin = (clap_plugin_t){
		.desc = &descriptor,
		.plugin_data = p,
		.init = plugin_init,
		.destroy = plugin_destroy,
		.activate = plugin_activate,
		.deactivate = plugin_deactivate,
		.start_processing = plugin_start_processing,
		.stop_processing = plugin_stop_processing,
		.reset = plugin_reset,
		.process = plugin_process,
		.get_extension = plugin_get_extension,
		.on_main_thread = plugin_on_main_thread,
	};
	return &p->plugin;
}

static const clap_plugin_factory_t factory = {
	.get_plugin_count = factory_count,
	.get_plugin_descriptor = factory_descriptor,
	.create_plugin = factory_create,
};

static bool entry_init(const char *plugin_path) { return true; }
static void entry_deinit(void) {}
static const void *entry_get_factory(const char *factory_id) {
	return strcmp(factory_id, CLAP_PLUGIN_FACTORY_ID) ? NULL : &factory;
}

CLAP_EXPORT const clap_plugin_entry_t clap_entry = {
	.clap_version = CLAP_VERSION_INIT,
	.init = entry_init,
	.deinit = entry_deinit,
	.get_factory = entry_get_factory,
};
