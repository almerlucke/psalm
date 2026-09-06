package timing

const (
	Second = 1000.0
	Minute = Second * 60
	Hour   = Minute * 60
)

func MsToSamps(ms float64, sr float64) int64 {
	return int64(MsToSampsf(ms, sr))
}

func MsToSampsf(ms float64, sr float64) float64 {
	return ms * 0.001 * sr
}

func SecToSamps(sec float64, sr float64) int64 {
	return int64(SecToSampsf(sec, sr))
}

func SecToSampsf(sec float64, sr float64) float64 {
	return sec * sr
}

func SampsToSec(samps int64, sr float64) float64 {
	return float64(samps) / sr
}

func SampsToMs(samps int64, sr float64) float64 {
	return (float64(samps) / sr) * 1000.0
}

func BPMToMilli(bpm int) float64 {
	return 60000.0 / float64(bpm)
}
