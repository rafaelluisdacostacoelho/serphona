// Package tts provides Text-to-Speech provider implementations.
package tts

import (
	"context"
	"fmt"
	"io"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"go.uber.org/zap"
	"google.golang.org/api/option"
)

// GoogleProviderV2 implements TTS using Google Cloud Text-to-Speech SDK.
type GoogleProviderV2 struct {
	client    *texttospeech.Client
	projectID string
	logger    *zap.Logger
}

// NewGoogleProviderV2 creates a new Google TTS provider using official SDK.
func NewGoogleProviderV2(projectID, credentialsPath string, logger *zap.Logger) (*GoogleProviderV2, error) {
	ctx := context.Background()

	var opts []option.ClientOption
	if credentialsPath != "" {
		opts = append(opts, option.WithCredentialsFile(credentialsPath))
	}

	client, err := texttospeech.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create text-to-speech client: %w", err)
	}

	logger.Info("google tts provider v2 initialized",
		zap.String("project_id", projectID),
	)

	return &GoogleProviderV2{
		client:    client,
		projectID: projectID,
		logger:    logger,
	}, nil
}

// Synthesize converts text to audio using Google TTS.
func (p *GoogleProviderV2) Synthesize(ctx context.Context, text string, config SynthesizeConfig) (io.Reader, error) {
	// Build synthesis input
	var input *texttospeechpb.SynthesisInput
	if config.SSML {
		input = &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Ssml{
				Ssml: text,
			},
		}
	} else {
		input = &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Text{
				Text: text,
			},
		}
	}

	// Build voice selection
	voice := &texttospeechpb.VoiceSelectionParams{
		LanguageCode: config.Language,
		Name:         config.VoiceID,
		SsmlGender:   p.getGender(config.Gender),
	}

	// Build audio config
	audioConfig := &texttospeechpb.AudioConfig{
		AudioEncoding:   p.getAudioEncoding(config.AudioEncoding),
		SampleRateHertz: int32(config.SampleRate),
		SpeakingRate:    float64(config.SpeakingRate),
		Pitch:           float64(config.Pitch),
		VolumeGainDb:    float64(config.VolumeGainDb),
	}

	// Add effects profile if specified
	if len(config.EffectsProfile) > 0 {
		audioConfig.EffectsProfileId = config.EffectsProfile
	}

	// Perform synthesis
	req := &texttospeechpb.SynthesizeSpeechRequest{
		Input:       input,
		Voice:       voice,
		AudioConfig: audioConfig,
	}

	resp, err := p.client.SynthesizeSpeech(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to synthesize speech: %w", err)
	}

	p.logger.Debug("speech synthesized",
		zap.String("text", text),
		zap.String("voice_id", config.VoiceID),
		zap.Int("audio_size", len(resp.AudioContent)),
	)

	return &readerWrapper{data: resp.AudioContent}, nil
}

// StreamSynthesize converts text to audio stream using Google TTS.
// Note: Google TTS doesn't support true streaming, so we use regular synthesis.
func (p *GoogleProviderV2) StreamSynthesize(ctx context.Context, text string, config SynthesizeConfig) (io.ReadCloser, error) {
	reader, err := p.Synthesize(ctx, text, config)
	if err != nil {
		return nil, err
	}

	return io.NopCloser(reader), nil
}

// ListVoices lists available voices for a language.
func (p *GoogleProviderV2) ListVoices(ctx context.Context, languageCode string) ([]*VoiceInfo, error) {
	req := &texttospeechpb.ListVoicesRequest{
		LanguageCode: languageCode,
	}

	resp, err := p.client.ListVoices(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list voices: %w", err)
	}

	voices := make([]*VoiceInfo, 0, len(resp.Voices))
	for _, v := range resp.Voices {
		voices = append(voices, &VoiceInfo{
			Name:          v.Name,
			LanguageCodes: v.LanguageCodes,
			Gender:        p.genderToString(v.SsmlGender),
			SampleRate:    int(v.NaturalSampleRateHertz),
		})
	}

	return voices, nil
}

// Close closes the Google TTS client.
func (p *GoogleProviderV2) Close() error {
	if err := p.client.Close(); err != nil {
		return fmt.Errorf("failed to close text-to-speech client: %w", err)
	}
	p.logger.Info("google tts client closed")
	return nil
}

// Name returns the provider name.
func (p *GoogleProviderV2) Name() string {
	return "google-v2"
}

// getGender converts string gender to Google gender enum.
func (p *GoogleProviderV2) getGender(gender string) texttospeechpb.SsmlVoiceGender {
	switch gender {
	case "MALE":
		return texttospeechpb.SsmlVoiceGender_MALE
	case "FEMALE":
		return texttospeechpb.SsmlVoiceGender_FEMALE
	case "NEUTRAL":
		return texttospeechpb.SsmlVoiceGender_NEUTRAL
	default:
		return texttospeechpb.SsmlVoiceGender_SSML_VOICE_GENDER_UNSPECIFIED
	}
}

// genderToString converts Google gender enum to string.
func (p *GoogleProviderV2) genderToString(gender texttospeechpb.SsmlVoiceGender) string {
	switch gender {
	case texttospeechpb.SsmlVoiceGender_MALE:
		return "MALE"
	case texttospeechpb.SsmlVoiceGender_FEMALE:
		return "FEMALE"
	case texttospeechpb.SsmlVoiceGender_NEUTRAL:
		return "NEUTRAL"
	default:
		return "UNSPECIFIED"
	}
}

// getAudioEncoding converts string encoding to Google audio encoding enum.
func (p *GoogleProviderV2) getAudioEncoding(encoding string) texttospeechpb.AudioEncoding {
	switch encoding {
	case "LINEAR16", "PCM":
		return texttospeechpb.AudioEncoding_LINEAR16
	case "MP3":
		return texttospeechpb.AudioEncoding_MP3
	case "OGG_OPUS":
		return texttospeechpb.AudioEncoding_OGG_OPUS
	case "MULAW":
		return texttospeechpb.AudioEncoding_MULAW
	case "ALAW":
		return texttospeechpb.AudioEncoding_ALAW
	default:
		return texttospeechpb.AudioEncoding_AUDIO_ENCODING_UNSPECIFIED
	}
}

// GetSupportedLanguages returns list of supported language codes.
func (p *GoogleProviderV2) GetSupportedLanguages() []string {
	return []string{
		"pt-BR", "pt-PT", // Portuguese
		"en-US", "en-GB", "en-AU", "en-IN", // English
		"es-ES", "es-MX", "es-US", // Spanish
		"fr-FR", "fr-CA", // French
		"de-DE",          // German
		"it-IT",          // Italian
		"ja-JP",          // Japanese
		"ko-KR",          // Korean
		"zh-CN", "zh-TW", // Chinese
		"nl-NL", // Dutch
		"ru-RU", // Russian
		"pl-PL", // Polish
		"tr-TR", // Turkish
		"sv-SE", // Swedish
		"da-DK", // Danish
		"fi-FI", // Finnish
		"no-NO", // Norwegian
		"ar-XA", // Arabic
		"hi-IN", // Hindi
		"id-ID", // Indonesian
		"th-TH", // Thai
		"vi-VN", // Vietnamese
	}
}

// GetSupportedVoiceTypes returns available voice types.
func (p *GoogleProviderV2) GetSupportedVoiceTypes() []string {
	return []string{
		"Standard",
		"WaveNet",
		"Neural2",
		"Studio",
		"News",
		"Journey",
	}
}

// readerWrapper wraps a byte slice to implement io.Reader.
type readerWrapper struct {
	data   []byte
	offset int
}

func (r *readerWrapper) Read(p []byte) (n int, err error) {
	if r.offset >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.offset:])
	r.offset += n
	return n, nil
}
