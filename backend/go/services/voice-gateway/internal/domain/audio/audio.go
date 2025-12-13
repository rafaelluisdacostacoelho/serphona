// Package audio contains audio domain value objects.
package audio

import (
	"fmt"
	"time"
)

// Format represents an audio format specification.
type Format struct {
	Codec      Codec `json:"codec"`
	SampleRate int   `json:"sample_rate"`
	Channels   int   `json:"channels"`
	BitDepth   int   `json:"bit_depth"`
	Bitrate    int   `json:"bitrate,omitempty"`
}

// Codec represents audio codec types.
type Codec string

const (
	CodecPCM     Codec = "pcm"
	CodecPCMU    Codec = "pcmu" // μ-law
	CodecPCMA    Codec = "pcma" // A-law
	CodecOpus    Codec = "opus"
	CodecMP3     Codec = "mp3"
	CodecAAC     Codec = "aac"
	CodecFLAC    Codec = "flac"
	CodecVorbis  Codec = "vorbis"
	CodecG722    Codec = "g722"
	CodecG729    Codec = "g729"
	CodecSpeex   Codec = "speex"
	CodecAMR     Codec = "amr"
	CodecAMRWB   Codec = "amr-wb"
	CodecUnknown Codec = "unknown"
)

// Common audio formats
var (
	// FormatTelephony represents standard telephony audio (8kHz, mono, PCM)
	FormatTelephony = Format{
		Codec:      CodecPCM,
		SampleRate: 8000,
		Channels:   1,
		BitDepth:   16,
	}

	// FormatWideband represents wideband telephony (16kHz, mono, PCM)
	FormatWideband = Format{
		Codec:      CodecPCM,
		SampleRate: 16000,
		Channels:   1,
		BitDepth:   16,
	}

	// FormatCD represents CD quality audio (44.1kHz, stereo, PCM)
	FormatCD = Format{
		Codec:      CodecPCM,
		SampleRate: 44100,
		Channels:   2,
		BitDepth:   16,
	}

	// FormatStudio represents studio quality audio (48kHz, stereo, PCM)
	FormatStudio = Format{
		Codec:      CodecPCM,
		SampleRate: 48000,
		Channels:   2,
		BitDepth:   24,
	}
)

// NewFormat creates a new audio format.
func NewFormat(codec Codec, sampleRate, channels, bitDepth int) Format {
	return Format{
		Codec:      codec,
		SampleRate: sampleRate,
		Channels:   channels,
		BitDepth:   bitDepth,
	}
}

// String returns a string representation of the format.
func (f Format) String() string {
	return fmt.Sprintf("%s/%dHz/%dch/%dbit", f.Codec, f.SampleRate, f.Channels, f.BitDepth)
}

// IsValid validates the audio format.
func (f Format) IsValid() bool {
	return f.Codec != CodecUnknown &&
		f.SampleRate > 0 &&
		f.Channels > 0 &&
		f.BitDepth > 0
}

// IsTelephony returns true if this is telephony-grade audio.
func (f Format) IsTelephony() bool {
	return f.SampleRate <= 8000 && f.Channels == 1
}

// IsWideband returns true if this is wideband audio.
func (f Format) IsWideband() bool {
	return f.SampleRate >= 16000 && f.SampleRate < 32000
}

// IsHighQuality returns true if this is high-quality audio.
func (f Format) IsHighQuality() bool {
	return f.SampleRate >= 44100
}

// BytesPerSecond calculates bytes per second for this format.
func (f Format) BytesPerSecond() int {
	return f.SampleRate * f.Channels * (f.BitDepth / 8)
}

// Chunk represents a chunk of audio data with metadata.
type Chunk struct {
	Data      []byte    `json:"data"`
	Format    Format    `json:"format"`
	Sequence  uint64    `json:"sequence"`
	Timestamp time.Time `json:"timestamp"`
	Duration  int       `json:"duration_ms"` // Duration in milliseconds

	// Optional metadata
	IsSilence bool                   `json:"is_silence,omitempty"`
	Energy    float64                `json:"energy,omitempty"` // Audio energy/volume
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewChunk creates a new audio chunk.
func NewChunk(data []byte, format Format, sequence uint64) *Chunk {
	return &Chunk{
		Data:      data,
		Format:    format,
		Sequence:  sequence,
		Timestamp: time.Now().UTC(),
		Metadata:  make(map[string]interface{}),
	}
}

// Size returns the size of the audio data in bytes.
func (c *Chunk) Size() int {
	return len(c.Data)
}

// IsValid validates the chunk.
func (c *Chunk) IsValid() bool {
	return len(c.Data) > 0 && c.Format.IsValid()
}

// CalculateDuration calculates the duration based on data size and format.
func (c *Chunk) CalculateDuration() {
	if !c.Format.IsValid() || len(c.Data) == 0 {
		c.Duration = 0
		return
	}

	bytesPerMs := c.Format.BytesPerSecond() / 1000
	if bytesPerMs > 0 {
		c.Duration = len(c.Data) / bytesPerMs
	}
}

// Stream represents a stream of audio chunks.
type Stream struct {
	ID        string     `json:"id"`
	Format    Format     `json:"format"`
	StartTime time.Time  `json:"start_time"`
	EndTime   *time.Time `json:"end_time,omitempty"`

	// Stream state
	Active   bool   `json:"active"`
	Sequence uint64 `json:"sequence"`

	// Statistics
	TotalChunks   uint64 `json:"total_chunks"`
	TotalBytes    uint64 `json:"total_bytes"`
	DroppedChunks uint64 `json:"dropped_chunks"`

	// Configuration
	BufferSize int `json:"buffer_size"` // Buffer size in chunks

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewStream creates a new audio stream.
func NewStream(id string, format Format, bufferSize int) *Stream {
	return &Stream{
		ID:         id,
		Format:     format,
		StartTime:  time.Now().UTC(),
		Active:     true,
		Sequence:   0,
		BufferSize: bufferSize,
		Metadata:   make(map[string]interface{}),
	}
}

// AddChunk increments counters when a chunk is added.
func (s *Stream) AddChunk(chunk *Chunk) {
	s.TotalChunks++
	s.TotalBytes += uint64(len(chunk.Data))
	s.Sequence = chunk.Sequence
}

// DropChunk increments the dropped counter.
func (s *Stream) DropChunk() {
	s.DroppedChunks++
}

// Close closes the stream.
func (s *Stream) Close() {
	now := time.Now().UTC()
	s.EndTime = &now
	s.Active = false
}

// Duration returns the stream duration.
func (s *Stream) Duration() time.Duration {
	if s.EndTime != nil {
		return s.EndTime.Sub(s.StartTime)
	}
	return time.Since(s.StartTime)
}

// IsActive returns true if the stream is active.
func (s *Stream) IsActive() bool {
	return s.Active && s.EndTime == nil
}

// GetDropRate returns the percentage of dropped chunks.
func (s *Stream) GetDropRate() float64 {
	if s.TotalChunks == 0 {
		return 0
	}
	return float64(s.DroppedChunks) / float64(s.TotalChunks) * 100
}

// GetAverageBitrate calculates average bitrate in kbps.
func (s *Stream) GetAverageBitrate() float64 {
	duration := s.Duration().Seconds()
	if duration == 0 {
		return 0
	}
	return float64(s.TotalBytes*8) / duration / 1000
}
