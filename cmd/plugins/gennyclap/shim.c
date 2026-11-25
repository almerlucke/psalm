// The CLAP side of the Genny spike: a note effect that plays what the generators of genny made, in time with
// the transport of the host. All it does is turn beats into sample offsets; the notes come from Go (genny_collect)
#include <math.h>
#include <stdlib.h>
#include <string.h>

#include <clap/clap.h>

#include "shim.h"
#include "_cgo_export.h"

#define MAX_EVENTS 256

typedef struct {
	clap_plugin_t plugin;
	const clap_host_t *host;
	double sample_rate;
	double next_beat; // where the last block ended, to find a jump of the transport
	int has_position;
	unsigned char sounding[128];
} genny_plugin;

static const clap_plugin_descriptor_t descriptor = {
	.clap_version = CLAP_VERSION_INIT,
	.id = "nl.almerlucke.psalm.genny-spike",
	.name = "Genny (spike)",
	.vendor = "psalm",
	.url = "",
	.manual_url = "",
	.support_url = "",
	.version = "0.0.1",
	.description = "Notes from the generators of genny",
	.features = (const char *[]){CLAP_PLUGIN_FEATURE_NOTE_EFFECT, NULL},
};

static void push_note(const clap_output_events_t *out, uint32_t time, int on, int key, double velocity) {
	clap_event_note_t e;
	memset(&e, 0, sizeof e);
	e.header.size = sizeof e;
	e.header.time = time;
	e.header.space_id = CLAP_CORE_EVENT_SPACE_ID;
	e.header.type = on ? CLAP_EVENT_NOTE_ON : CLAP_EVENT_NOTE_OFF;
	e.header.flags = 0;
	e.note_id = -1;
	e.port_index = 0;
	e.channel = 0;
	e.key = key;
	e.velocity = velocity;
	out->try_push(out, &e.header);
}

static void release_all(genny_plugin *p, const clap_output_events_t *out, uint32_t time) {
	for (int k = 0; k < 128; k++) {
		if (p->sounding[k]) {
			push_note(out, time, 0, k, 0);
			p->sounding[k] = 0;
		}
	}
}

static bool plugin_init(const clap_plugin_t *plugin) { return true; }
static void plugin_destroy(const clap_plugin_t *plugin) { free(plugin->plugin_data); }

static bool plugin_activate(const clap_plugin_t *plugin, double sample_rate, uint32_t min_frames, uint32_t max_frames) {
	genny_plugin *p = plugin->plugin_data;
	p->sample_rate = sample_rate;
	p->has_position = 0;
	genny_prepare();
	return true;
}

static void plugin_deactivate(const clap_plugin_t *plugin) {}
static bool plugin_start_processing(const clap_plugin_t *plugin) { return true; }
static void plugin_stop_processing(const clap_plugin_t *plugin) {}
static void plugin_reset(const clap_plugin_t *plugin) {
	genny_plugin *p = plugin->plugin_data;
	p->has_position = 0;
}

static clap_process_status plugin_process(const clap_plugin_t *plugin, const clap_process_t *process) {
	genny_plugin *p = plugin->plugin_data;
	const clap_event_transport_t *tr = process->transport;
	uint32_t frames = process->frames_count;

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
	int n = genny_collect(from, to, events, MAX_EVENTS);
	for (int i = 0; i < n; i++) {
		double offset = (events[i].beat - from) / per_sample;
		uint32_t time = offset < 0 ? 0 : (offset >= frames ? frames - 1 : (uint32_t)offset);
		int key = events[i].key;
		if (key < 0 || key > 127) {
			continue;
		}
		if (events[i].on) {
			push_note(process->out_events, time, 1, key, events[i].velocity);
			p->sounding[key] = 1;
		} else if (p->sounding[key]) {
			push_note(process->out_events, time, 0, key, 0);
			p->sounding[key] = 0;
		}
	}
	return CLAP_PROCESS_CONTINUE;
}

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

static const void *plugin_get_extension(const clap_plugin_t *plugin, const char *id) {
	if (!strcmp(id, CLAP_EXT_NOTE_PORTS)) {
		return &note_ports;
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
