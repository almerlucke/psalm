// two small CLAP plugins for the tests: a sine instrument (a gain parameter; the tuning of a note moves its pitch) and a gain effect
#include <math.h>
#include <stdlib.h>
#include <string.h>

#include "clap/clap.h"

#define VOICES 16

typedef struct {
	int on;
	int32_t id;
	double freq, phase;
} voice;

typedef struct {
	clap_plugin_t plugin;
	const clap_host_t *host;
	int instrument;
	double sr, gain;
	voice v[VOICES];
} test;

static const char *inst_features[] = {CLAP_PLUGIN_FEATURE_INSTRUMENT, CLAP_PLUGIN_FEATURE_SYNTHESIZER, NULL};
static const char *fx_features[] = {CLAP_PLUGIN_FEATURE_AUDIO_EFFECT, NULL};
static const clap_plugin_descriptor_t descs[2] = {
    {CLAP_VERSION_INIT, "psalm.test.sine", "Psalm Test Sine", "psalm", "", "", "", "1", "a sine", inst_features},
    {CLAP_VERSION_INIT, "psalm.test.gain", "Psalm Test Gain", "psalm", "", "", "", "1", "a gain", fx_features},
};

static void handle(test *t, const clap_event_header_t *h) {
	if (h->space_id != CLAP_CORE_EVENT_SPACE_ID) {
		return;
	}
	switch (h->type) {
	case CLAP_EVENT_NOTE_ON: {
		const clap_event_note_t *n = (const clap_event_note_t *)h;
		for (int i = 0; i < VOICES; i++) {
			if (!t->v[i].on) {
				t->v[i] = (voice){1, n->note_id, 440.0 * pow(2.0, (n->key - 69) / 12.0), 0};
				break;
			}
		}
		break;
	}
	case CLAP_EVENT_NOTE_OFF: {
		const clap_event_note_t *n = (const clap_event_note_t *)h;
		for (int i = 0; i < VOICES; i++) {
			if (t->v[i].on && t->v[i].id == n->note_id) {
				t->v[i].on = 0;
			}
		}
		break;
	}
	case CLAP_EVENT_NOTE_EXPRESSION: {
		const clap_event_note_expression_t *e = (const clap_event_note_expression_t *)h;
		if (e->expression_id == CLAP_NOTE_EXPRESSION_TUNING) {
			for (int i = 0; i < VOICES; i++) {
				if (t->v[i].on && t->v[i].id == e->note_id) {
					t->v[i].freq *= pow(2.0, e->value / 12.0);
				}
			}
		}
		break;
	}
	case CLAP_EVENT_PARAM_VALUE:
		t->gain = ((const clap_event_param_value_t *)h)->value;
		break;
	}
}

static bool t_init(const clap_plugin_t *p) {
	test *t = p->plugin_data;
	t->gain = t->instrument ? 0.5 : 1.0;
	return true;
}
static void t_destroy(const clap_plugin_t *p) { free(p->plugin_data); }
static bool t_activate(const clap_plugin_t *p, double sr, uint32_t min, uint32_t max) {
	((test *)p->plugin_data)->sr = sr;
	return true;
}
static void t_deactivate(const clap_plugin_t *p) {}
static bool t_start(const clap_plugin_t *p) { return true; }
static void t_stop(const clap_plugin_t *p) {}
static void t_reset(const clap_plugin_t *p) {}
static void t_on_main(const clap_plugin_t *p) {}

static clap_process_status t_process(const clap_plugin_t *p, const clap_process_t *pr) {
	test *t = p->plugin_data;
	uint32_t n = pr->in_events->size(pr->in_events), e = 0;
	float *o0 = pr->audio_outputs[0].data32[0], *o1 = pr->audio_outputs[0].data32[1];
	for (uint32_t i = 0; i < pr->frames_count; i++) {
		while (e < n && pr->in_events->get(pr->in_events, e)->time <= i) {
			handle(t, pr->in_events->get(pr->in_events, e++));
		}
		float s = 0;
		if (t->instrument) {
			for (int k = 0; k < VOICES; k++) {
				if (t->v[k].on) {
					s += (float)(sin(t->v[k].phase) * t->gain);
					t->v[k].phase += 2 * M_PI * t->v[k].freq / t->sr;
				}
			}
			o0[i] = o1[i] = s;
		} else {
			o0[i] = pr->audio_inputs[0].data32[0][i] * (float)t->gain;
			o1[i] = pr->audio_inputs[0].data32[1][i] * (float)t->gain;
		}
	}
	for (; e < n; e++) {
		handle(t, pr->in_events->get(pr->in_events, e));
	}
	return CLAP_PROCESS_CONTINUE;
}

static uint32_t ap_count(const clap_plugin_t *p, bool is_input) {
	test *t = p->plugin_data;
	return (is_input && t->instrument) ? 0 : 1;
}
static bool ap_get(const clap_plugin_t *p, uint32_t i, bool is_input, clap_audio_port_info_t *info) {
	info->id = 0;
	strcpy(info->name, "main");
	info->channel_count = 2;
	info->flags = CLAP_AUDIO_PORT_IS_MAIN;
	info->port_type = CLAP_PORT_STEREO;
	info->in_place_pair = CLAP_INVALID_ID;
	return true;
}
static const clap_plugin_audio_ports_t audio_ports = {ap_count, ap_get};

static uint32_t pa_count(const clap_plugin_t *p) { return 1; }
static bool pa_info(const clap_plugin_t *p, uint32_t i, clap_param_info_t *info) {
	test *t = p->plugin_data;
	memset(info, 0, sizeof *info);
	info->id = 0;
	info->flags = CLAP_PARAM_IS_AUTOMATABLE;
	strcpy(info->name, "Gain");
	strcpy(info->module, "Output");
	info->min_value = 0;
	info->max_value = t->instrument ? 1 : 2;
	info->default_value = t->instrument ? 0.5 : 1;
	return true;
}
static bool pa_value(const clap_plugin_t *p, clap_id id, double *v) {
	*v = ((test *)p->plugin_data)->gain;
	return true;
}
static bool pa_text(const clap_plugin_t *p, clap_id id, double v, char *out, uint32_t cap) { return false; }
static bool pa_from_text(const clap_plugin_t *p, clap_id id, const char *text, double *v) { return false; }
static void pa_flush(const clap_plugin_t *p, const clap_input_events_t *in, const clap_output_events_t *out) {
	for (uint32_t i = 0; i < in->size(in); i++) {
		handle(p->plugin_data, in->get(in, i));
	}
}
static const clap_plugin_params_t params = {pa_count, pa_info, pa_value, pa_text, pa_from_text, pa_flush};

static const void *t_extension(const clap_plugin_t *p, const char *id) {
	if (!strcmp(id, CLAP_EXT_AUDIO_PORTS)) {
		return &audio_ports;
	}
	if (!strcmp(id, CLAP_EXT_PARAMS)) {
		return &params;
	}
	return NULL;
}

static uint32_t f_count(const clap_plugin_factory_t *f) { return 2; }
static const clap_plugin_descriptor_t *f_desc(const clap_plugin_factory_t *f, uint32_t i) { return i < 2 ? &descs[i] : NULL; }
static const clap_plugin_t *f_create(const clap_plugin_factory_t *f, const clap_host_t *host, const char *id) {
	for (int i = 0; i < 2; i++) {
		if (!strcmp(id, descs[i].id)) {
			test *t = calloc(1, sizeof *t);
			t->host = host;
			t->instrument = i == 0;
			t->plugin = (clap_plugin_t){&descs[i], t, t_init, t_destroy, t_activate, t_deactivate, t_start, t_stop, t_reset, t_process, t_extension, t_on_main};
			return &t->plugin;
		}
	}
	return NULL;
}
static const clap_plugin_factory_t factory = {f_count, f_desc, f_create};

static bool e_init(const char *path) { return true; }
static void e_deinit(void) {}
static const void *e_factory(const char *id) { return !strcmp(id, CLAP_PLUGIN_FACTORY_ID) ? &factory : NULL; }

CLAP_EXPORT const clap_plugin_entry_t clap_entry = {CLAP_VERSION_INIT, e_init, e_deinit, e_factory};
