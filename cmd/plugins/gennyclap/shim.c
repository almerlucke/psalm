// The CLAP side of Genny: a note effect that plays what the generators of genny made, in time with the transport of
// the host, and takes the notes played into it. All it does is turn beats into sample offsets, keep track of the notes
// that sound and put the events in order; the notes, the parameters and the state come from Go (see main.go)
#include <math.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include <clap/clap.h>

#include "shim.h"
#include "_cgo_export.h"

#define MAX_EVENTS 512
#define MAX_ACTIVE 512
#define MAX_QUEUE 2048

// the numbers of the parameters (see engine.go) that the audio thread looks at
#define PARAM_THRU 7
#define PARAM_KEYS 8

typedef struct {
	int id;
	int key;
	int channel;
	int voice; // that made the note (see genny_event), 0 for the piece of the transport
} active_note;

// every kind of event the plugin sends, in one place so they can be put in order of time before they go
typedef union {
	clap_event_header_t header;
	clap_event_note_t note;
	clap_event_note_expression_t expression;
} queued_event;

// the host of each instance, by handle, for genny_host_log (called from the goroutine that renders)
#define MAX_HOSTS 64
static const clap_host_t *hosts[MAX_HOSTS];

void genny_host_log(int handle, int severity, const char *msg) {
	if (handle < 0 || handle >= MAX_HOSTS) {
		return;
	}
	const clap_host_t *host = __atomic_load_n(&hosts[handle], __ATOMIC_ACQUIRE);
	if (!host || !host->get_extension) {
		return;
	}
	const clap_host_log_t *log = host->get_extension(host, CLAP_EXT_LOG);
	if (log && log->log) {
		log->log(host, severity, msg);
	}
}

typedef struct {
	clap_plugin_t plugin;
	const clap_host_t *host;
	int handle;
	double sample_rate;
	double next_beat; // where the last block ended, to find a jump of the transport
	int has_position;
	active_note active[MAX_ACTIVE];
	int num_active;
	queued_event queue[MAX_QUEUE]; // what goes out in this block
	int num_queued;
	long long epoch; // of the piece that is playing (see genny_epoch)
	double tempo;      // the last tempo the host gave, for the clock that runs when the transport does not
	double free_beat;  // that clock: where the block begins, in beats from when the transport stopped
	int clock;         // what the beats come from: 0 nothing plays, 1 the transport, 2 the free clock (key trigger, transport stopped)
} genny_plugin;

static const clap_plugin_descriptor_t descriptor = {
	.clap_version = CLAP_VERSION_INIT,
	.id = "nl.almerlucke.psalm.genny",
	.name = "Genny",
	.vendor = "psalm",
	.url = "",
	.manual_url = "",
	.support_url = "",
	.version = "0.3.0",
	.description = "Notes from the generators of genny, from a spec or a script, that can follow the notes played into it",
	.features = (const char *[]){CLAP_PLUGIN_FEATURE_NOTE_EFFECT, NULL},
};

static queued_event *next_event(genny_plugin *p) {
	if (p->num_queued >= MAX_QUEUE) {
		return NULL;
	}
	queued_event *e = &p->queue[p->num_queued++];
	memset(e, 0, sizeof *e);
	return e;
}

static void queue_note(genny_plugin *p, uint32_t time, int on, int id, int key, int channel, double velocity) {
	queued_event *q = next_event(p);
	if (!q) {
		return;
	}
	clap_event_note_t *e = &q->note;
	e->header.size = sizeof *e;
	e->header.time = time;
	e->header.space_id = CLAP_CORE_EVENT_SPACE_ID;
	e->header.type = on ? CLAP_EVENT_NOTE_ON : CLAP_EVENT_NOTE_OFF;
	e->header.flags = 0;
	e->note_id = id;
	e->port_index = 0;
	e->channel = channel;
	e->key = key;
	e->velocity = velocity;
}

// queue_tuning retunes a note that is on: semitones from its key (a pitch between two keys, see genny_event.tune)
static void queue_tuning(genny_plugin *p, uint32_t time, int id, int key, int channel, double semitones) {
	queued_event *q = next_event(p);
	if (!q) {
		return;
	}
	clap_event_note_expression_t *e = &q->expression;
	e->header.size = sizeof *e;
	e->header.time = time;
	e->header.space_id = CLAP_CORE_EVENT_SPACE_ID;
	e->header.type = CLAP_EVENT_NOTE_EXPRESSION;
	e->header.flags = 0;
	e->expression_id = CLAP_NOTE_EXPRESSION_TUNING;
	e->note_id = id;
	e->port_index = 0;
	e->channel = channel;
	e->key = key;
	e->value = semitones;
}

// queue_copy passes an event on as it came
static void queue_copy(genny_plugin *p, const clap_event_header_t *h) {
	if (h->size > sizeof(queued_event)) {
		return;
	}
	queued_event *q = next_event(p);
	if (q) {
		memcpy(q, h, h->size);
	}
}

// flush_queue sends the events of the block in order of time (the order they were made in when the time is the same: a note on
// goes before its tuning, a note off before the next note on)
static void flush_queue(genny_plugin *p, const clap_output_events_t *out) {
	for (int i = 1; i < p->num_queued; i++) {
		queued_event tmp = p->queue[i];
		int j = i - 1;
		while (j >= 0 && p->queue[j].header.time > tmp.header.time) {
			p->queue[j + 1] = p->queue[j];
			j--;
		}
		p->queue[j + 1] = tmp;
	}
	for (int i = 0; i < p->num_queued; i++) {
		out->try_push(out, &p->queue[i].header);
	}
	p->num_queued = 0;
}

static void release_all(genny_plugin *p, uint32_t time) {
	for (int i = 0; i < p->num_active; i++) {
		queue_note(p, time, 0, p->active[i].id, p->active[i].key, p->active[i].channel, 0);
	}
	p->num_active = 0;
}

// read_input takes what the host sends: the parameters, and the notes played into the plugin (passed on when "pass input" is on)
static void read_input(genny_plugin *p, const clap_input_events_t *in, double from, double per_sample) {
	if (!in) {
		return;
	}
	uint32_t n = in->size(in);
	int thru = -1; // looked at when the first note comes
	for (uint32_t i = 0; i < n; i++) {
		const clap_event_header_t *h = in->get(in, i);
		if (h->space_id != CLAP_CORE_EVENT_SPACE_ID) {
			continue;
		}
		switch (h->type) {
		case CLAP_EVENT_PARAM_VALUE: {
			const clap_event_param_value_t *e = (const clap_event_param_value_t *)h;
			genny_param_set(p->handle, (int)e->param_id, e->value);
			break;
		}
		case CLAP_EVENT_NOTE_ON:
		case CLAP_EVENT_NOTE_OFF:
		case CLAP_EVENT_NOTE_CHOKE:
		case CLAP_EVENT_NOTE_END: {
			const clap_event_note_t *e = (const clap_event_note_t *)h;
			if (h->type != CLAP_EVENT_NOTE_END) {
				genny_input(p->handle, e->key, e->velocity, h->type == CLAP_EVENT_NOTE_ON, from + h->time * per_sample);
			}
			if (thru < 0) {
				thru = genny_param_get(p->handle, PARAM_THRU) > 0.5;
			}
			if (thru && h->type != CLAP_EVENT_NOTE_END) {
				queue_copy(p, h);
			}
			break;
		}
		case CLAP_EVENT_MIDI: { // a host that sends the notes as MIDI
			const clap_event_midi_t *e = (const clap_event_midi_t *)h;
			int status = e->data[0] & 0xF0, key = e->data[1];
			if (status == 0x90 || status == 0x80) {
				genny_input(p->handle, key, e->data[2] / 127.0, status == 0x90 && e->data[2] > 0, from + h->time * per_sample);
			}
			break;
		}
		}
	}
}

static bool plugin_init(const clap_plugin_t *plugin) {
	genny_plugin *p = plugin->plugin_data;
	p->handle = genny_new();
	if (p->handle >= 0 && p->handle < MAX_HOSTS) {
		__atomic_store_n(&hosts[p->handle], p->host, __ATOMIC_RELEASE);
	}
	return p->handle >= 0;
}

static void plugin_destroy(const clap_plugin_t *plugin) {
	genny_plugin *p = plugin->plugin_data;
	genny_stop(p->handle); // its goroutine must be done before the host is let go
	if (p->handle >= 0 && p->handle < MAX_HOSTS) {
		__atomic_store_n(&hosts[p->handle], NULL, __ATOMIC_RELEASE);
	}
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
	p->clock = 0;
	p->num_active = 0;
	p->num_queued = 0;
	genny_reset(p->handle);
}

static clap_process_status plugin_process(const clap_plugin_t *plugin, const clap_process_t *process) {
	genny_plugin *p = plugin->plugin_data;
	const clap_event_transport_t *tr = process->transport;
	uint32_t frames = process->frames_count;

	p->num_queued = 0;

	// where the block is, in beats: the transport when it plays; the clock of the plugin when it does not but the key trigger is on (the
	// keys are played with the transport stopped as well); otherwise nothing plays
	if (tr && (tr->flags & CLAP_TRANSPORT_HAS_TEMPO) && tr->tempo > 0) {
		p->tempo = tr->tempo;
	}
	int playing = tr && (tr->flags & CLAP_TRANSPORT_IS_PLAYING) && (tr->flags & CLAP_TRANSPORT_HAS_BEATS_TIMELINE) &&
	              (tr->flags & CLAP_TRANSPORT_HAS_TEMPO);
	int keys = genny_param_get(p->handle, PARAM_KEYS) > 0.5;
	int clock = playing ? 1 : (keys ? 2 : 0);
	if (clock != p->clock) { // the beats come from somewhere else: what sounds is let go, the voices forgotten
		release_all(p, 0);
		genny_reset(p->handle);
		p->has_position = 0;
		p->free_beat = 0;
		p->clock = clock;
	}
	double per_sample = (p->tempo > 0 ? p->tempo : 120.0) / (60.0 * p->sample_rate); // beats per sample
	double from = clock == 1 ? (double)tr->song_pos_beats / (double)CLAP_BEATTIME_FACTOR : p->free_beat;
	double to = from + per_sample * frames;

	read_input(p, process->in_events, from, per_sample);

	if (clock == 0) {
		flush_queue(p, process->out_events);
		return CLAP_PROCESS_CONTINUE;
	}
	if (clock == 2) {
		p->free_beat = to;
	}

	// the transport jumped (a loop, a click in the ruler), or the piece was replaced (the new one does not know the notes that sound):
	// what sounds is let go
	long long epoch = genny_epoch(p->handle);
	if ((p->has_position && fabs(from - p->next_beat) > 1e-4) || epoch != p->epoch) {
		release_all(p, 0);
		if (clock == 1 && p->has_position && fabs(from - p->next_beat) > 1e-4) {
			genny_reset(p->handle);
		}
	}
	p->epoch = epoch;
	p->next_beat = to;
	p->has_position = 1;

	genny_event events[MAX_EVENTS];
	int n = genny_collect(p->handle, from, to, events, MAX_EVENTS);
	for (int i = 0; i < n; i++) {
		double offset = (events[i].beat - from) / per_sample;
		uint32_t time = offset < 0 ? 0 : (offset >= frames ? frames - 1 : (uint32_t)offset);
		if (events[i].on == 2) { // the end of a voice: the notes it still has on are let go
			for (int j = 0; j < p->num_active;) {
				if (p->active[j].voice == events[i].voice) {
					queue_note(p, time, 0, p->active[j].id, p->active[j].key, p->active[j].channel, 0);
					p->active[j] = p->active[--p->num_active];
				} else {
					j++;
				}
			}
			continue;
		}
		if (events[i].on) {
			if (p->num_active < MAX_ACTIVE) {
				p->active[p->num_active++] = (active_note){events[i].id, events[i].key, events[i].channel, events[i].voice};
				queue_note(p, time, 1, events[i].id, events[i].key, events[i].channel, events[i].velocity);
				if (fabs(events[i].tune) > 1e-4) { // right after the note on, at the same time
					queue_tuning(p, time, events[i].id, events[i].key, events[i].channel, events[i].tune);
				}
			}
			continue;
		}
		for (int j = 0; j < p->num_active; j++) {
			if (p->active[j].id == events[i].id) {
				queue_note(p, time, 0, p->active[j].id, p->active[j].key, p->active[j].channel, 0);
				p->active[j] = p->active[--p->num_active];
				break;
			}
		}
	}
	flush_queue(p, process->out_events);
	return CLAP_PROCESS_CONTINUE;
}

// the note ports: one input (the notes the script can follow), one output

static uint32_t note_ports_count(const clap_plugin_t *plugin, bool is_input) { return 1; }

static bool note_ports_get(const clap_plugin_t *plugin, uint32_t index, bool is_input, clap_note_port_info_t *info) {
	if (index != 0) {
		return false;
	}
	info->id = is_input ? 1 : 0;
	info->supported_dialects = is_input ? (CLAP_NOTE_DIALECT_CLAP | CLAP_NOTE_DIALECT_MIDI) : CLAP_NOTE_DIALECT_CLAP;
	info->preferred_dialect = CLAP_NOTE_DIALECT_CLAP;
	strncpy(info->name, is_input ? "Input" : "Notes", sizeof info->name);
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
	return genny_param_text((int)id, value, out, (int)size) != 0;
}

static bool params_text_to_value(const clap_plugin_t *plugin, clap_id id, const char *text, double *value) {
	return genny_param_parse((int)id, (char *)text, value) != 0;
}

static void params_flush(const clap_plugin_t *plugin, const clap_input_events_t *in, const clap_output_events_t *out) {
	genny_plugin *p = plugin->plugin_data;
	read_input(p, in, p->next_beat, 0); // notes that come while the plugin is not processing are let through as well
	flush_queue(p, out);
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
