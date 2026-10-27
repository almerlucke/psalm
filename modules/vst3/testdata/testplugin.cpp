// two small VST3 plugins for the tests: a sine instrument (a gain parameter; the tuning of a note moves its pitch; three outputs) and a gain effect (a side chain,
// a latency, the tempo added to the sound). The component and the controller are classes of their own, as in most plugins.
#include <math.h>
#include <string.h>

#include <atomic>
#include <string>
#include <vector>

#include "pluginterfaces/base/ibstream.h"
#include "pluginterfaces/base/ipluginbase.h"
#include "pluginterfaces/vst/ivstaudioprocessor.h"
#include "pluginterfaces/vst/ivstcomponent.h"
#include "pluginterfaces/vst/ivsteditcontroller.h"
#include "pluginterfaces/vst/ivstevents.h"
#include "pluginterfaces/vst/ivstmessage.h"
#include "pluginterfaces/vst/ivstparameterchanges.h"
#include "pluginterfaces/vst/ivstprocesscontext.h"
#include "pluginterfaces/vst/ivstunits.h"
#include "pluginterfaces/vst/vstspeaker.h"

using namespace Steinberg;
using namespace Steinberg::Vst;

template <class T, class I>
static bool give(T *self, const TUID iid, void **obj) {
	if (FUnknownPrivate::iidEqual(iid, I::iid.toTUID())) {
		*obj = static_cast<I *>(self);
		self->addRef();
		return true;
	}
	return false;
}

template <class First, class... Rest>
class Obj : public First, public Rest... {
public:
	virtual ~Obj() {}
	tresult PLUGIN_API queryInterface(const TUID iid, void **obj) override {
		if (FUnknownPrivate::iidEqual(iid, FUnknown::iid.toTUID())) {
			*obj = static_cast<FUnknown *>(static_cast<First *>(this));
			addRef();
			return kResultOk;
		}
		if (give<Obj, First>(this, iid, obj) || (give<Obj, Rest>(this, iid, obj) || ...)) return kResultOk;
		*obj = nullptr;
		return kNoInterface;
	}
	uint32 PLUGIN_API addRef() override { return ++refs_; }
	uint32 PLUGIN_API release() override {
		uint32 n = --refs_;
		if (n == 0) delete this;
		return n;
	}

private:
	std::atomic<uint32> refs_{1};
};

static const FUID sineComp(0x50534C4D, 0x56535433, 0x53494E45, 0x434F4D50), sineCtrl(0x50534C4D, 0x56535433, 0x53494E45, 0x43545244);
static const FUID gainComp(0x50534C4D, 0x56535433, 0x4741494E, 0x434F4D50), gainCtrl(0x50534C4D, 0x56535433, 0x4741494E, 0x43545244);

// the gain is the normalized value of the parameter 0 (times two for the effect)
static void read_gain(IBStream *s, double &gain) {
	char buf[64] = {0};
	int32 n = 0;
	s->read(buf, sizeof buf - 1, &n);
	double g;
	if (n > 0 && sscanf(buf, "gain=%lf", &g) == 1) gain = g;
}

class TestComponent : public Obj<IComponent, IAudioProcessor> {
public:
	bool instrument;
	double sr = 44100, gain;
	struct Voice {
		bool on = false;
		int32 id = 0;
		double freq = 0, phase = 0;
	} voices[16];
	explicit TestComponent(bool inst) : instrument(inst), gain(inst ? 0.5 : 1.0) {}
	tresult PLUGIN_API initialize(FUnknown *) override { return kResultOk; }
	tresult PLUGIN_API terminate() override { return kResultOk; }
	tresult PLUGIN_API getControllerClassId(TUID id) override {
		(instrument ? sineCtrl : gainCtrl).toTUID(id);
		return kResultOk;
	}
	tresult PLUGIN_API setIoMode(IoMode) override { return kResultOk; }
	int32 PLUGIN_API getBusCount(MediaType type, BusDirection dir) override {
		if (type == kAudio) return instrument ? (dir == kInput ? 0 : 3) : (dir == kInput ? 2 : 1);
		return instrument && dir == kInput ? 1 : 0;
	}
	tresult PLUGIN_API getBusInfo(MediaType type, BusDirection dir, int32 index, BusInfo &bus) override {
		if (index < 0 || index >= getBusCount(type, dir)) return kResultFalse;
		bus.mediaType = type;
		bus.direction = dir;
		bus.channelCount = type == kAudio ? 2 : 1;
		const char *n = index == 0 ? "main" : "aux";
		for (int i = 0; i < 5; i++) bus.name[i] = (char16)n[i < 4 ? i : 0] * (i < 4 ? 1 : 0);
		bus.name[4] = 0;
		bus.busType = index == 0 ? kMain : kAux;
		bus.flags = index == 0 ? BusInfo::kDefaultActive : 0;
		return kResultOk;
	}
	tresult PLUGIN_API getRoutingInfo(RoutingInfo &, RoutingInfo &) override { return kResultFalse; }
	tresult PLUGIN_API activateBus(MediaType, BusDirection, int32, TBool) override { return kResultOk; }
	tresult PLUGIN_API setActive(TBool) override { return kResultOk; }
	tresult PLUGIN_API setState(IBStream *s) override {
		read_gain(s, gain);
		return kResultOk;
	}
	tresult PLUGIN_API getState(IBStream *s) override {
		char buf[64];
		int n = snprintf(buf, sizeof buf, "gain=%.6f", gain);
		return s->write(buf, n, nullptr);
	}
	tresult PLUGIN_API setBusArrangements(SpeakerArrangement *, int32, SpeakerArrangement *, int32) override { return kResultOk; }
	tresult PLUGIN_API getBusArrangement(BusDirection, int32, SpeakerArrangement &arr) override {
		arr = SpeakerArr::kStereo;
		return kResultOk;
	}
	tresult PLUGIN_API canProcessSampleSize(int32 s) override { return s == kSample32 ? kResultOk : kResultFalse; }
	uint32 PLUGIN_API getLatencySamples() override { return instrument ? 0 : 64; }
	tresult PLUGIN_API setupProcessing(ProcessSetup &s) override {
		sr = s.sampleRate;
		return kResultOk;
	}
	tresult PLUGIN_API setProcessing(TBool) override { return kResultOk; }
	uint32 PLUGIN_API getTailSamples() override { return 0; }
	tresult PLUGIN_API process(ProcessData &d) override {
		if (d.inputParameterChanges) {
			for (int32 i = 0; i < d.inputParameterChanges->getParameterCount(); i++) {
				IParamValueQueue *q = d.inputParameterChanges->getParameterData(i);
				int32 n = q->getPointCount(), off;
				ParamValue v;
				if (q->getParameterId() == 0 && n > 0 && q->getPoint(n - 1, off, v) == kResultOk) gain = instrument ? v : v * 2;
				if (q->getParameterId() == 1 && n > 0 && q->getPoint(n - 1, off, v) == kResultOk) { // a program: its gain
					static const double programs[3] = {0.1, 0.5, 0.9};
					int index = (int)lround(v * 2);
					gain = (instrument ? 1 : 2) * programs[index < 0 ? 0 : index > 2 ? 2 : index];
				}
			}
		}
		if (d.numOutputs < 1) return kResultOk;
		int32 frames = d.numSamples;
		float *o0 = d.outputs[0].channelBuffers32[0], *o1 = d.outputs[0].channelBuffers32[1];
		int32 next = 0, count = d.inputEvents ? d.inputEvents->getEventCount() : 0;
		for (int32 s = 0; s < frames; s++) {
			while (next < count) {
				Event e;
				d.inputEvents->getEvent(next, e);
				if (e.sampleOffset > s) break;
				next++;
				if (e.type == Event::kNoteOnEvent) {
					if (e.noteOn.pitch == 126) *(volatile int *)0 = 1; // a plugin that crashes
					for (auto &v : voices) {
						if (!v.on) {
							v = Voice{true, e.noteOn.noteId, 440.0 * pow(2.0, (e.noteOn.pitch - 69 + e.noteOn.tuning / 100.0) / 12.0), 0};
							break;
						}
					}
				} else if (e.type == Event::kNoteOffEvent) {
					for (auto &v : voices)
						if (v.on && v.id == e.noteOff.noteId) v.on = false;
				}
			}
			float a = 0, b = 0;
			if (instrument) {
				for (auto &v : voices) {
					if (v.on) {
						a += (float)(sin(v.phase) * gain);
						v.phase += 2 * M_PI * v.freq / sr;
					}
				}
				b = a;
				if (d.numOutputs > 1) d.outputs[1].channelBuffers32[0][s] = d.outputs[1].channelBuffers32[1][s] = a * 0.5f; // the second output has it at half level
			} else {
				float side0 = 0, side1 = 0, tempo = 0;
				if (d.numInputs > 1 && d.inputs[1].numChannels > 1) {
					side0 = d.inputs[1].channelBuffers32[0][s] * 0.25f;
					side1 = d.inputs[1].channelBuffers32[1][s] * 0.25f;
				}
				if (d.processContext && (d.processContext->state & ProcessContext::kTempoValid)) tempo = (float)d.processContext->tempo * 0.0001f;
				a = d.inputs[0].channelBuffers32[0][s] * (float)gain + side0 + tempo;
				b = d.inputs[0].channelBuffers32[1][s] * (float)gain + side1 + tempo;
			}
			o0[s] = a;
			o1[s] = b;
		}
		return kResultOk;
	}
};

class TestController : public Obj<IEditController, IUnitInfo> {
public:
	bool instrument;
	double value, program = 0;
	IComponentHandler *handler = nullptr;
	explicit TestController(bool inst) : instrument(inst), value(0.5) {}
	tresult PLUGIN_API initialize(FUnknown *) override { return kResultOk; }
	tresult PLUGIN_API terminate() override { return kResultOk; }
	tresult PLUGIN_API setComponentState(IBStream *s) override {
		double g = instrument ? 0.5 : 1.0;
		read_gain(s, g);
		value = instrument ? g : g / 2;
		return kResultOk;
	}
	tresult PLUGIN_API setState(IBStream *) override { return kResultOk; }
	tresult PLUGIN_API getState(IBStream *) override { return kResultOk; }
	int32 PLUGIN_API getParameterCount() override { return 2; }
	tresult PLUGIN_API getParameterInfo(int32 i, ParameterInfo &info) override {
		if (i == 1) { // the program
			memset(&info, 0, sizeof info);
			info.id = 1;
			const char *t = "Program";
			for (int k = 0; t[k]; k++) info.title[k] = (char16)t[k];
			info.stepCount = 2;
			info.unitId = 1;
			info.flags = ParameterInfo::kCanAutomate | ParameterInfo::kIsProgramChange | ParameterInfo::kIsList;
			return kResultOk;
		}
		if (i != 0) return kResultFalse;
		memset(&info, 0, sizeof info);
		info.id = 0;
		const char *t = instrument && value > 0.75 ? "Loudness" : "Gain"; // above three quarters the parameter is called Loudness: the host is told
		for (int k = 0; t[k]; k++) info.title[k] = (char16)t[k];
		info.defaultNormalizedValue = 0.5;
		info.flags = ParameterInfo::kCanAutomate;
		return kResultOk;
	}
	tresult PLUGIN_API getParamStringByValue(ParamID, ParamValue v, String128 s) override {
		char buf[32];
		snprintf(buf, sizeof buf, "%.2f", v * (instrument ? 1 : 2));
		for (int k = 0; k < 32; k++) s[k] = (char16)buf[k];
		return kResultOk;
	}
	tresult PLUGIN_API getParamValueByString(ParamID, TChar *, ParamValue &) override { return kResultFalse; }
	ParamValue PLUGIN_API normalizedParamToPlain(ParamID, ParamValue v) override { return v; }
	ParamValue PLUGIN_API plainParamToNormalized(ParamID, ParamValue v) override { return v; }
	ParamValue PLUGIN_API getParamNormalized(ParamID id) override { return id == 1 ? program : value; }
	tresult PLUGIN_API setParamNormalized(ParamID id, ParamValue v) override {
		if (id == 1) { // a program: the gain it has
			program = v;
			static const double programs[3] = {0.1, 0.5, 0.9};
			int index = (int)lround(v * 2);
			value = programs[index < 0 ? 0 : index > 2 ? 2 : index] * (instrument ? 1 : 1);
			return kResultOk;
		}
		bool was = value > 0.75;
		value = v;
		if (instrument && handler && was != (v > 0.75)) handler->restartComponent(kParamTitlesChanged);
		return kResultOk;
	}
	tresult PLUGIN_API setComponentHandler(IComponentHandler *h) override {
		handler = h;
		return kResultOk;
	}
	IPlugView *PLUGIN_API createView(FIDString) override { return nullptr; }
	// the programs: one list of three in the unit 1
	int32 PLUGIN_API getUnitCount() override { return 2; }
	tresult PLUGIN_API getUnitInfo(int32 i, UnitInfo &info) override {
		memset(&info, 0, sizeof info);
		info.id = i == 0 ? kRootUnitId : 1;
		info.parentUnitId = i == 0 ? kNoParentUnitId : kRootUnitId;
		info.programListId = i == 0 ? kNoProgramListId : 7;
		const char *n = "Programs";
		if (i == 1) for (int k = 0; n[k]; k++) info.name[k] = (char16)n[k];
		return i < 2 ? kResultOk : kResultFalse;
	}
	int32 PLUGIN_API getProgramListCount() override { return 1; }
	tresult PLUGIN_API getProgramListInfo(int32 i, ProgramListInfo &info) override {
		if (i != 0) return kResultFalse;
		memset(&info, 0, sizeof info);
		info.id = 7;
		info.programCount = 3;
		return kResultOk;
	}
	tresult PLUGIN_API getProgramName(ProgramListID list, int32 index, String128 name) override {
		static const char *names[3] = {"Quiet", "Middle", "Loud"};
		if (list != 7 || index < 0 || index > 2) return kResultFalse;
		memset(name, 0, sizeof(String128));
		for (int k = 0; names[index][k]; k++) name[k] = (char16)names[index][k];
		return kResultOk;
	}
	tresult PLUGIN_API getProgramInfo(ProgramListID, int32, Vst::CString, String128) override { return kResultFalse; }
	tresult PLUGIN_API hasProgramPitchNames(ProgramListID, int32) override { return kResultFalse; }
	tresult PLUGIN_API getProgramPitchName(ProgramListID, int32, int16, String128) override { return kResultFalse; }
	UnitID PLUGIN_API getSelectedUnit() override { return kRootUnitId; }
	tresult PLUGIN_API selectUnit(UnitID) override { return kResultOk; }
	tresult PLUGIN_API getUnitByBus(MediaType, BusDirection, int32, int32, UnitID &) override { return kResultFalse; }
	tresult PLUGIN_API setUnitProgramData(int32, int32, IBStream *) override { return kResultFalse; }
};

class TestFactory : public Obj<IPluginFactory3> {
public:
	tresult PLUGIN_API queryInterface(const TUID iid, void **obj) override { // the interfaces it extends too
		if (FUnknownPrivate::iidEqual(iid, IPluginFactory2::iid.toTUID())) {
			*obj = static_cast<IPluginFactory2 *>(this);
			addRef();
			return kResultOk;
		}
		if (FUnknownPrivate::iidEqual(iid, IPluginFactory::iid.toTUID())) {
			*obj = static_cast<IPluginFactory *>(this);
			addRef();
			return kResultOk;
		}
		return Obj<IPluginFactory3>::queryInterface(iid, obj);
	}
	tresult PLUGIN_API getFactoryInfo(PFactoryInfo *info) override {
		strcpy(info->vendor, "psalm");
		strcpy(info->url, "");
		strcpy(info->email, "");
		info->flags = 16; // unicode
		return kResultOk;
	}
	int32 PLUGIN_API countClasses() override { return 4; }
	static const FUID &cid(int32 i) {
		static const FUID *all[4] = {&sineComp, &sineCtrl, &gainComp, &gainCtrl};
		return *all[i];
	}
	tresult PLUGIN_API getClassInfo(int32 i, PClassInfo *info) override {
		if (i < 0 || i > 3) return kInvalidArgument;
		cid(i).toTUID(info->cid);
		info->cardinality = PClassInfo::kManyInstances;
		strcpy(info->category, i % 2 == 0 ? kVstAudioEffectClass : kVstComponentControllerClass);
		strcpy(info->name, i < 2 ? "Psalm VST3 Sine" : "Psalm VST3 Gain");
		return kResultOk;
	}
	tresult PLUGIN_API createInstance(FIDString c, FIDString iid, void **obj) override {
		for (int i = 0; i < 4; i++) {
			if (FUnknownPrivate::iidEqual(c, cid(i).toTUID())) {
				FUnknown *o = nullptr;
				if (i % 2 == 0) o = static_cast<IComponent *>(new TestComponent(i == 0));
				else o = static_cast<IEditController *>(new TestController(i == 1));
				tresult r = o->queryInterface(iid, obj);
				o->release();
				return r;
			}
		}
		*obj = nullptr;
		return kNoInterface;
	}
	tresult PLUGIN_API getClassInfo2(int32 i, PClassInfo2 *info) override {
		if (i < 0 || i > 3) return kInvalidArgument;
		cid(i).toTUID(info->cid);
		info->cardinality = PClassInfo::kManyInstances;
		strcpy(info->category, i % 2 == 0 ? kVstAudioEffectClass : kVstComponentControllerClass);
		strcpy(info->name, i < 2 ? "Psalm VST3 Sine" : "Psalm VST3 Gain");
		info->classFlags = 0;
		strcpy(info->subCategories, i < 2 ? "Instrument|Synth" : "Fx");
		strcpy(info->vendor, "psalm");
		strcpy(info->version, "1");
		strcpy(info->sdkVersion, "VST 3.8.0");
		return kResultOk;
	}
	tresult PLUGIN_API getClassInfoUnicode(int32, PClassInfoW *) override { return kNotImplemented; }
	tresult PLUGIN_API setHostContext(FUnknown *) override { return kResultOk; }
};

extern "C" {
__attribute__((visibility("default"))) bool ModuleEntry(void *) { return true; }
__attribute__((visibility("default"))) bool ModuleExit() { return true; }
__attribute__((visibility("default"))) IPluginFactory *GetPluginFactory() { return new TestFactory; }
}
