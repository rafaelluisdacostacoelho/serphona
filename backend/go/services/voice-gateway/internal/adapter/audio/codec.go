// Package audio provides audio codec and processing utilities.
package audio

// CodecType represents audio codec types.
type CodecType string

const (
	CodecPCM   CodecType = "pcm"
	CodecOpus  CodecType = "opus"
	CodecMP3   CodecType = "mp3"
	CodecMulaw CodecType = "mulaw"
	CodecAlaw  CodecType = "alaw"
)

// AudioFormat represents audio format parameters.
type AudioFormat struct {
	Codec      CodecType
	SampleRate int
	Channels   int
	BitRate    int
}

// DefaultFormat returns default audio format (PCM 16kHz mono).
func DefaultFormat() AudioFormat {
	return AudioFormat{
		Codec:      CodecPCM,
		SampleRate: 16000,
		Channels:   1,
		BitRate:    256000,
	}
}

// ConvertPCMToInt16 converts PCM bytes to int16 samples.
func ConvertPCMToInt16(pcmData []byte) []int16 {
	samples := make([]int16, len(pcmData)/2)
	for i := 0; i < len(samples); i++ {
		samples[i] = int16(pcmData[i*2]) | int16(pcmData[i*2+1])<<8
	}
	return samples
}

// ConvertInt16ToPCM converts int16 samples to PCM bytes.
func ConvertInt16ToPCM(samples []int16) []byte {
	pcmData := make([]byte, len(samples)*2)
	for i, sample := range samples {
		pcmData[i*2] = byte(sample)
		pcmData[i*2+1] = byte(sample >> 8)
	}
	return pcmData
}

// ConvertPCMToFloat32 converts PCM int16 to float32 (-1.0 to 1.0).
func ConvertPCMToFloat32(samples []int16) []float32 {
	floats := make([]float32, len(samples))
	for i, sample := range samples {
		floats[i] = float32(sample) / 32768.0
	}
	return floats
}

// ConvertFloat32ToPCM converts float32 (-1.0 to 1.0) to PCM int16.
func ConvertFloat32ToPCM(floats []float32) []int16 {
	samples := make([]int16, len(floats))
	for i, f := range floats {
		// Clamp to valid range
		if f > 1.0 {
			f = 1.0
		} else if f < -1.0 {
			f = -1.0
		}
		samples[i] = int16(f * 32767.0)
	}
	return samples
}

// ResamplePCM resamples PCM audio (simple linear interpolation).
func ResamplePCM(input []int16, inputRate, outputRate int) []int16 {
	if inputRate == outputRate {
		return input
	}

	ratio := float64(inputRate) / float64(outputRate)
	outputLen := int(float64(len(input)) / ratio)
	output := make([]int16, outputLen)

	for i := 0; i < outputLen; i++ {
		srcIdx := float64(i) * ratio
		srcIdxInt := int(srcIdx)
		srcIdxFrac := srcIdx - float64(srcIdxInt)

		if srcIdxInt+1 < len(input) {
			// Linear interpolation
			output[i] = int16(float64(input[srcIdxInt])*(1.0-srcIdxFrac) +
				float64(input[srcIdxInt+1])*srcIdxFrac)
		} else {
			output[i] = input[srcIdxInt]
		}
	}

	return output
}

// MixAudio mixes two PCM audio streams.
func MixAudio(a, b []int16) []int16 {
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}

	result := make([]int16, maxLen)
	for i := 0; i < maxLen; i++ {
		var sampleA, sampleB int32
		if i < len(a) {
			sampleA = int32(a[i])
		}
		if i < len(b) {
			sampleB = int32(b[i])
		}

		// Mix with clamping
		mixed := sampleA + sampleB
		if mixed > 32767 {
			mixed = 32767
		} else if mixed < -32768 {
			mixed = -32768
		}
		result[i] = int16(mixed)
	}

	return result
}

// ApplyGain applies gain (volume adjustment) to PCM audio.
func ApplyGain(samples []int16, gainDb float64) []int16 {
	// Convert dB to linear gain
	gain := pow10(gainDb / 20.0)

	result := make([]int16, len(samples))
	for i, sample := range samples {
		scaled := float64(sample) * gain
		if scaled > 32767 {
			scaled = 32767
		} else if scaled < -32768 {
			scaled = -32768
		}
		result[i] = int16(scaled)
	}

	return result
}

// pow10 calculates 10^x
func pow10(x float64) float64 {
	// Simple approximation using exp
	// 10^x = e^(x * ln(10))
	const ln10 = 2.302585092994046
	return exp(x * ln10)
}

// exp calculates e^x using Taylor series (simple approximation)
func exp(x float64) float64 {
	sum := 1.0
	term := 1.0
	for i := 1; i < 20; i++ {
		term *= x / float64(i)
		sum += term
	}
	return sum
}

// CalculateRMS calculates the RMS (Root Mean Square) of PCM audio.
func CalculateRMS(samples []int16) float64 {
	if len(samples) == 0 {
		return 0
	}

	var sum float64
	for _, sample := range samples {
		val := float64(sample) / 32768.0
		sum += val * val
	}

	return sqrt(sum / float64(len(samples)))
}

// sqrt calculates square root using Newton's method
func sqrt(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x == 0 {
		return 0
	}

	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}

// DetectSilence detects if audio contains silence.
func DetectSilence(samples []int16, threshold float64) bool {
	rms := CalculateRMS(samples)
	return rms < threshold
}

// TrimSilence removes silence from the beginning and end of audio.
func TrimSilence(samples []int16, threshold float64, frameSize int) []int16 {
	if len(samples) == 0 {
		return samples
	}

	// Find start of speech
	start := 0
	for i := 0; i < len(samples)-frameSize; i += frameSize {
		frame := samples[i : i+frameSize]
		if !DetectSilence(frame, threshold) {
			start = i
			break
		}
	}

	// Find end of speech
	end := len(samples)
	for i := len(samples) - frameSize; i >= 0; i -= frameSize {
		endIdx := i + frameSize
		if endIdx > len(samples) {
			endIdx = len(samples)
		}
		frame := samples[i:endIdx]
		if !DetectSilence(frame, threshold) {
			end = endIdx
			break
		}
	}

	if start >= end {
		return []int16{}
	}

	return samples[start:end]
}
