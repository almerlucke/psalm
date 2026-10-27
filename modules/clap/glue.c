#include "glue.h"

#include <ctype.h>
#include <dirent.h>
#include <dlfcn.h>
#include <sys/stat.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "clap/clap.h"

#define MAX_EVENTS 4096

typedef struct {
	uint32_t flags, kind;
	char name[256], location[1024];
} pc_location;

struct pc_lib {
	void *handle;
	const clap_plugin_entry_t *entry;
	const clap_plugin_factory_t *factory;

	// the preset discovery: what the providers declared, and the presets found
	char (*exts)[32];
	int n_exts;
	pc_location *locs;
	int n_locs;
	pc_preset *presets;
	int n_presets, cap_presets;
	pc_preset cur; // the preset being told by a provider
	int cur_open;
	pc_location *cur_loc;
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
	volatile int callback; // the plugin asked for a call on the main thread
};

static void seterr(char *err, int cap, const char *msg) {
	if (err && cap > 0) {
		snprintf(err, cap, "%s", msg);
	}
}

// --- the host the plugins see: it offers nothing but its name ---

static const void *host_get_extension(const clap_host_t *host, const char *id) { return NULL; }
static void host_nothing(const clap_host_t *host) {}
static void host_callback(const clap_host_t *host) { ((pc_inst *)host->host_data)->callback = 1; }

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
	                           .request_callback = host_callback};
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

int pi_param_text(pc_inst *inst, uint32_t id, double value, char *out, int cap) {
	if (!inst->params || !inst->params->value_to_text) {
		return 0;
	}
	out[0] = 0;
	return inst->params->value_to_text(inst->plugin, id, value, out, (uint32_t)cap) && out[0] != 0;
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

// the cookie the plugin gave a parameter: it asks for it back in the events that set it
static void *param_cookie(pc_inst *inst, uint32_t id) {
	clap_param_info_t info;
	if (inst->params) {
		uint32_t n = inst->params->count(inst->plugin);
		for (uint32_t i = 0; i < n; i++) {
			if (inst->params->get_info(inst->plugin, i, &info) && info.id == id) {
				return info.cookie;
			}
		}
	}
	return NULL;
}

void pi_param(pc_inst *inst, uint32_t id, double value, uint32_t time) {
	pc_event *e = push(inst, CLAP_EVENT_PARAM_VALUE, sizeof(clap_event_param_value_t), time);
	if (!e) {
		return;
	}
	e->param.param_id = id;
	e->param.cookie = param_cookie(inst, id);
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
	if (inst->callback) { // what the plugin asked to do on the main thread: done here, between two blocks, as the same thread as the process
		inst->callback = 0;
		inst->plugin->on_main_thread(inst->plugin);
	}
	if (inst->out_ch == 1) {
		memcpy(out1, out0, frames * sizeof(float));
	}
	return st != CLAP_PROCESS_ERROR;
}

// --- preset discovery: the indexer the providers declare to, the receiver they tell the presets to ---

static bool ix_filetype(const clap_preset_discovery_indexer_t *ix, const clap_preset_discovery_filetype_t *ft) {
	pc_lib *lib = ix->indexer_data;
	if (!ft || !ft->file_extension) {
		return false;
	}
	lib->exts = realloc(lib->exts, (lib->n_exts + 1) * sizeof *lib->exts);
	snprintf(lib->exts[lib->n_exts++], 32, "%s", ft->file_extension);
	return true;
}

static bool ix_location(const clap_preset_discovery_indexer_t *ix, const clap_preset_discovery_location_t *l) {
	pc_lib *lib = ix->indexer_data;
	if (!l || (l->kind == CLAP_PRESET_DISCOVERY_LOCATION_FILE && !l->location)) {
		return false;
	}
	lib->locs = realloc(lib->locs, (lib->n_locs + 1) * sizeof *lib->locs);
	pc_location *p = &lib->locs[lib->n_locs++];
	memset(p, 0, sizeof *p);
	p->flags = l->flags;
	p->kind = l->kind;
	snprintf(p->name, sizeof p->name, "%s", l->name ? l->name : "");
	snprintf(p->location, sizeof p->location, "%s", l->location ? l->location : "");
	return true;
}

static bool ix_soundpack(const clap_preset_discovery_indexer_t *ix, const clap_preset_discovery_soundpack_t *sp) { return true; }
static const void *ix_extension(const clap_preset_discovery_indexer_t *ix, const char *id) { return NULL; }

static void flush_preset(pc_lib *lib) {
	if (!lib->cur_open) {
		return;
	}
	lib->cur_open = 0;
	if (lib->n_presets == lib->cap_presets) {
		lib->cap_presets = lib->cap_presets ? lib->cap_presets * 2 : 256;
		lib->presets = realloc(lib->presets, lib->cap_presets * sizeof *lib->presets);
	}
	lib->presets[lib->n_presets++] = lib->cur;
}

static void rc_error(const clap_preset_discovery_metadata_receiver_t *r, int32_t os_error, const char *msg) {}

static bool rc_begin(const clap_preset_discovery_metadata_receiver_t *r, const char *name, const char *load_key) {
	pc_lib *lib = r->receiver_data;
	flush_preset(lib);
	memset(&lib->cur, 0, sizeof lib->cur);
	snprintf(lib->cur.name, sizeof lib->cur.name, "%s", name ? name : "");
	snprintf(lib->cur.loadkey, sizeof lib->cur.loadkey, "%s", load_key ? load_key : "");
	lib->cur.kind = lib->cur_loc->kind;
	snprintf(lib->cur.location, sizeof lib->cur.location, "%s", lib->cur_loc->location);
	lib->cur.flags = lib->cur_loc->flags;
	lib->cur_open = 1;
	return true;
}

static void rc_plugin(const clap_preset_discovery_metadata_receiver_t *r, const clap_universal_plugin_id_t *id) {
	pc_lib *lib = r->receiver_data;
	if (lib->cur_open && id && id->abi && !strcmp(id->abi, "clap") && !lib->cur.plugin[0]) {
		snprintf(lib->cur.plugin, sizeof lib->cur.plugin, "%s", id->id ? id->id : "");
	}
}

static void rc_soundpack(const clap_preset_discovery_metadata_receiver_t *r, const char *id) {}
static void rc_flags(const clap_preset_discovery_metadata_receiver_t *r, uint32_t flags) {
	pc_lib *lib = r->receiver_data;
	if (lib->cur_open) {
		lib->cur.flags = flags;
	}
}
static void rc_creator(const clap_preset_discovery_metadata_receiver_t *r, const char *creator) {
	pc_lib *lib = r->receiver_data;
	if (lib->cur_open && creator && !lib->cur.creator[0]) {
		snprintf(lib->cur.creator, sizeof lib->cur.creator, "%s", creator);
	}
}
static void rc_description(const clap_preset_discovery_metadata_receiver_t *r, const char *d) {
	pc_lib *lib = r->receiver_data;
	if (lib->cur_open && d) {
		snprintf(lib->cur.description, sizeof lib->cur.description, "%s", d);
	}
}
static void rc_timestamps(const clap_preset_discovery_metadata_receiver_t *r, clap_timestamp a, clap_timestamp b) {}
static void rc_feature(const clap_preset_discovery_metadata_receiver_t *r, const char *f) {
	pc_lib *lib = r->receiver_data;
	if (lib->cur_open && f) {
		size_t used = strlen(lib->cur.features);
		snprintf(lib->cur.features + used, sizeof lib->cur.features - used, "%s%s", used ? "\x1f" : "", f);
	}
}
static void rc_extra(const clap_preset_discovery_metadata_receiver_t *r, const char *k, const char *v) {}

static int has_ext(pc_lib *lib, const char *path) {
	const char *dot = strrchr(path, '.');
	if (!dot) {
		return 0;
	}
	for (int i = 0; i < lib->n_exts; i++) {
		const char *e = lib->exts[i];
		if (e[0] == '.') {
			e++;
		}
		const char *d = dot + 1;
		size_t n = strlen(e);
		if (strlen(d) == n) {
			int same = 1;
			for (size_t k = 0; k < n; k++) {
				if (tolower((unsigned char)e[k]) != tolower((unsigned char)d[k])) {
					same = 0;
					break;
				}
			}
			if (same) {
				return 1;
			}
		}
	}
	return 0;
}

static void crawl(pc_lib *lib, const clap_preset_discovery_provider_t *prov, const clap_preset_discovery_metadata_receiver_t *rc, const char *path, int depth) {
	struct stat st;
	if (depth > 12 || stat(path, &st) != 0) {
		return;
	}
	if (S_ISDIR(st.st_mode)) {
		DIR *d = opendir(path);
		if (!d) {
			return;
		}
		struct dirent *e;
		while ((e = readdir(d))) {
			if (e->d_name[0] == '.') {
				continue;
			}
			char sub[2048];
			snprintf(sub, sizeof sub, "%s/%s", path, e->d_name);
			crawl(lib, prov, rc, sub, depth + 1);
		}
		closedir(d);
		return;
	}
	if (!has_ext(lib, path)) {
		return;
	}
	pc_location loc = {0, CLAP_PRESET_DISCOVERY_LOCATION_FILE, "", ""};
	snprintf(loc.location, sizeof loc.location, "%s", path);
	loc.flags = lib->cur_loc ? lib->cur_loc->flags : 0;
	pc_location *outer = lib->cur_loc;
	lib->cur_loc = &loc;
	prov->get_metadata(prov, CLAP_PRESET_DISCOVERY_LOCATION_FILE, path, rc);
	flush_preset(lib);
	lib->cur_loc = outer;
}

int pc_scan_presets(pc_lib *lib) {
	if (lib->n_presets > 0) {
		return lib->n_presets;
	}
	const clap_preset_discovery_factory_t *f = lib->entry->get_factory(CLAP_PRESET_DISCOVERY_FACTORY_ID);
	if (!f) {
		f = lib->entry->get_factory(CLAP_PRESET_DISCOVERY_FACTORY_ID_COMPAT);
	}
	if (!f) {
		return 0;
	}
	for (uint32_t i = 0; i < f->count(f); i++) {
		const clap_preset_discovery_provider_descriptor_t *d = f->get_descriptor(f, i);
		if (!d) {
			continue;
		}
		clap_preset_discovery_indexer_t ix = {.clap_version = CLAP_VERSION_INIT,
		                                      .name = "psalm",
		                                      .vendor = "psalm",
		                                      .url = "",
		                                      .version = "1",
		                                      .indexer_data = lib,
		                                      .declare_filetype = ix_filetype,
		                                      .declare_location = ix_location,
		                                      .declare_soundpack = ix_soundpack,
		                                      .get_extension = ix_extension};
		const clap_preset_discovery_provider_t *prov = f->create(f, &ix, d->id);
		if (!prov) {
			continue;
		}
		if (!prov->init(prov)) {
			prov->destroy(prov);
			continue;
		}
		clap_preset_discovery_metadata_receiver_t rc = {.receiver_data = lib,
		                                                .on_error = rc_error,
		                                                .begin_preset = rc_begin,
		                                                .add_plugin_id = rc_plugin,
		                                                .set_soundpack_id = rc_soundpack,
		                                                .set_flags = rc_flags,
		                                                .add_creator = rc_creator,
		                                                .set_description = rc_description,
		                                                .set_timestamps = rc_timestamps,
		                                                .add_feature = rc_feature,
		                                                .add_extra_info = rc_extra};
		// the declarations are done in init: crawl what was declared
		int n = lib->n_locs;
		pc_location *locs = malloc(n * sizeof *locs);
		memcpy(locs, lib->locs, n * sizeof *locs);
		for (int k = 0; k < n; k++) {
			lib->cur_loc = &locs[k];
			if (locs[k].kind == CLAP_PRESET_DISCOVERY_LOCATION_FILE) {
				crawl(lib, prov, &rc, locs[k].location, 0);
			} else {
				prov->get_metadata(prov, locs[k].kind, locs[k].location[0] ? locs[k].location : NULL, &rc);
				flush_preset(lib);
			}
		}
		free(locs);
		lib->cur_loc = NULL;
		prov->destroy(prov);
	}
	return lib->n_presets;
}

int pc_preset_get(pc_lib *lib, int i, pc_preset *out) {
	if (i < 0 || i >= lib->n_presets) {
		return 0;
	}
	*out = lib->presets[i];
	return 1;
}

int pi_load_preset(pc_inst *inst, uint32_t kind, const char *location, const char *loadkey) {
	const clap_plugin_preset_load_t *pl = inst->plugin->get_extension(inst->plugin, CLAP_EXT_PRESET_LOAD);
	if (!pl) {
		pl = inst->plugin->get_extension(inst->plugin, CLAP_EXT_PRESET_LOAD_COMPAT);
	}
	if (!pl || !pl->from_location) {
		return 0;
	}
	return pl->from_location(inst->plugin, kind, location && location[0] ? location : NULL, loadkey && loadkey[0] ? loadkey : NULL) ? 1 : 0;
}

// --- the state of a plugin ---

typedef struct {
	uint8_t *data;
	size_t len, cap, pos;
} membuf;

static int64_t os_write(const clap_ostream_t *s, const void *buf, uint64_t size) {
	membuf *m = s->ctx;
	if (m->len + size > m->cap) {
		m->cap = (m->len + size) * 2;
		m->data = realloc(m->data, m->cap);
	}
	memcpy(m->data + m->len, buf, size);
	m->len += size;
	return (int64_t)size;
}

static int64_t is_read(const clap_istream_t *s, void *buf, uint64_t size) {
	membuf *m = s->ctx;
	uint64_t n = m->len - m->pos;
	if (n > size) {
		n = size;
	}
	memcpy(buf, m->data + m->pos, n);
	m->pos += n;
	return (int64_t)n;
}

int pi_state_save(pc_inst *inst, uint8_t **out, size_t *len) {
	const clap_plugin_state_t *st = inst->plugin->get_extension(inst->plugin, CLAP_EXT_STATE);
	if (!st) {
		return 0;
	}
	membuf m = {0};
	clap_ostream_t os = {.ctx = &m, .write = os_write};
	if (!st->save(inst->plugin, &os)) {
		free(m.data);
		return 0;
	}
	*out = m.data;
	*len = m.len;
	return 1;
}

int pi_state_load(pc_inst *inst, const uint8_t *data, size_t len) {
	const clap_plugin_state_t *st = inst->plugin->get_extension(inst->plugin, CLAP_EXT_STATE);
	if (!st) {
		return 0;
	}
	membuf m = {.data = (uint8_t *)data, .len = len};
	clap_istream_t is = {.ctx = &m, .read = is_read};
	return st->load(inst->plugin, &is) ? 1 : 0;
}

void pc_free(void *p) { free(p); }
