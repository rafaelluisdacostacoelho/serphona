// Package stt provides Speech-to-Text provider implementations.
package stt

import (
	"context"
	"fmt"
	"io"

	speech "cloud.google.com/go/speech/apiv1"
	"cloud.google.com/go/speech/apiv1/speechpb"
	"go.uber.org/zap"
	"google.golang.org/api/option"
)

// GoogleProviderV2 implements STT using Google Cloud Speech-to-Text SDK.
type GoogleProviderV2 struct {
	client    *speech.Client
	projectID string
	logger    *zap.Logger
}

// NewGoogleProviderV2 creates a new Google STT provider using official SDK.
func NewGoogleProviderV2(projectID, credentialsPath string, logger *zap.Logger) (*GoogleProviderV2, error) {
	ctx := context.Background()

	var opts []option.ClientOption
	if credentialsPath != "" {
		opts = append(opts, option.WithCredentialsFile(credentialsPath))
	}

	client, err := speech.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create speech client: %w", err)
	}

	logger.Info("google stt provider v2 initialized",
		zap.String("project_id", projectID),
	)

	return &GoogleProviderV2{
		client:    client,
		projectID: projectID,
		logger:    logger,
	}, nil
}

// StreamTranscribe transcribes audio stream using Google STT streaming API.
func (p *GoogleProviderV2) StreamTranscribe(ctx context.Context, audioStream io.Reader, config StreamConfig) (<-chan Result, error) {
	results := make(chan Result, 10)

	// Create streaming recognize client
	stream, err := p.client.StreamingRecognize(ctx)
	if err != nil {
		close(results)
		return nil, fmt.Errorf("failed to create streaming recognize: %w", err)
	}

	// Build recognition config
	recognitionConfig := &speechpb.RecognitionConfig{
		Encoding:                   p.getEncoding(config.Encoding),
		SampleRateHertz:            int32(config.SampleRate),
		AudioChannelCount:          int32(config.Channels),
		LanguageCode:               config.Language,
		EnableAutomaticPunctuation: config.EnablePunctuation,
		Model:                      config.Model,
		UseEnhanced:                config.UseEnhanced,
		MaxAlternatives:            int32(config.MaxAlternatives),
	}

	// Add speech contexts if provided
	if len(config.PhraseHints) > 0 {
		recognitionConfig.SpeechContexts = []*speechpb.SpeechContext{
			{
				Phrases: config.PhraseHints,
				Boost:   config.PhraseBoost,
			},
		}
	}

	// Build streaming config
	streamingConfig := &speechpb.StreamingRecognitionConfig{
		Config:          recognitionConfig,
		InterimResults:  config.InterimResults,
		SingleUtterance: config.SingleUtterance,
	}

	// Send initial config
	if err := stream.Send(&speechpb.StreamingRecognizeRequest{
		StreamingRequest: &speechpb.StreamingRecognizeRequest_StreamingConfig{
			StreamingConfig: streamingConfig,
		},
	}); err != nil {
		close(results)
		return nil, fmt.Errorf("failed to send streaming config: %w", err)
	}

	// Start goroutine to send audio data
	go func() {
		defer func() {
			if err := stream.CloseSend(); err != nil {
				p.logger.Error("failed to close send stream", zap.Error(err))
			}
		}()

		buffer := make([]byte, 8192)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := audioStream.Read(buffer)
				if err != nil {
					if err != io.EOF {
						p.logger.Error("error reading audio stream", zap.Error(err))
					}
					return
				}

				if n > 0 {
					if err := stream.Send(&speechpb.StreamingRecognizeRequest{
						StreamingRequest: &speechpb.StreamingRecognizeRequest_AudioContent{
							AudioContent: buffer[:n],
						},
					}); err != nil {
						p.logger.Error("failed to send audio content", zap.Error(err))
						return
					}
				}
			}
		}
	}()

	// Start goroutine to receive results
	go func() {
		defer close(results)

		for {
			resp, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					p.logger.Debug("streaming recognize completed")
					return
				}
				p.logger.Error("streaming recognize error", zap.Error(err))
				results <- Result{
					Error: err,
				}
				return
			}

			// Process results
			for _, result := range resp.Results {
				if len(result.Alternatives) == 0 {
					continue
				}

				// Get best alternative
				alt := result.Alternatives[0]

				results <- Result{
					Transcript: alt.Transcript,
					Confidence: alt.Confidence,
					IsFinal:    result.IsFinal,
					Words:      p.convertWords(alt.Words),
				}
			}
		}
	}()

	return results, nil
}

// RecognizeOnce performs one-shot speech recognition.
func (p *GoogleProviderV2) RecognizeOnce(ctx context.Context, audioData []byte, config StreamConfig) (*Result, error) {
	// Build recognition config
	recognitionConfig := &speechpb.RecognitionConfig{
		Encoding:                   p.getEncoding(config.Encoding),
		SampleRateHertz:            int32(config.SampleRate),
		AudioChannelCount:          int32(config.Channels),
		LanguageCode:               config.Language,
		EnableAutomaticPunctuation: config.EnablePunctuation,
		Model:                      config.Model,
		UseEnhanced:                config.UseEnhanced,
		MaxAlternatives:            int32(config.MaxAlternatives),
	}

	// Add speech contexts if provided
	if len(config.PhraseHints) > 0 {
		recognitionConfig.SpeechContexts = []*speechpb.SpeechContext{
			{
				Phrases: config.PhraseHints,
				Boost:   config.PhraseBoost,
			},
		}
	}

	// Build request
	req := &speechpb.RecognizeRequest{
		Config: recognitionConfig,
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{
				Content: audioData,
			},
		},
	}

	// Perform recognition
	resp, err := p.client.Recognize(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to recognize: %w", err)
	}

	// Extract best result
	if len(resp.Results) == 0 || len(resp.Results[0].Alternatives) == 0 {
		return &Result{
			Transcript: "",
			Confidence: 0.0,
			IsFinal:    true,
		}, nil
	}

	alt := resp.Results[0].Alternatives[0]
	return &Result{
		Transcript: alt.Transcript,
		Confidence: alt.Confidence,
		IsFinal:    true,
		Words:      p.convertWords(alt.Words),
	}, nil
}

// LongRunningRecognize performs long-running speech recognition.
func (p *GoogleProviderV2) LongRunningRecognize(ctx context.Context, audioURI string, config StreamConfig) (<-chan Result, error) {
	results := make(chan Result, 10)

	// Build recognition config
	recognitionConfig := &speechpb.RecognitionConfig{
		Encoding:                   p.getEncoding(config.Encoding),
		SampleRateHertz:            int32(config.SampleRate),
		AudioChannelCount:          int32(config.Channels),
		LanguageCode:               config.Language,
		EnableAutomaticPunctuation: config.EnablePunctuation,
		Model:                      config.Model,
		UseEnhanced:                config.UseEnhanced,
		MaxAlternatives:            int32(config.MaxAlternatives),
		EnableWordTimeOffsets:      true,
	}

	// Add speech contexts if provided
	if len(config.PhraseHints) > 0 {
		recognitionConfig.SpeechContexts = []*speechpb.SpeechContext{
			{
				Phrases: config.PhraseHints,
				Boost:   config.PhraseBoost,
			},
		}
	}

	// Build request
	req := &speechpb.LongRunningRecognizeRequest{
		Config: recognitionConfig,
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Uri{
				Uri: audioURI,
			},
		},
	}

	// Start long-running operation
	op, err := p.client.LongRunningRecognize(ctx, req)
	if err != nil {
		close(results)
		return nil, fmt.Errorf("failed to start long running recognize: %w", err)
	}

	// Wait for operation in background
	go func() {
		defer close(results)

		resp, err := op.Wait(ctx)
		if err != nil {
			results <- Result{
				Error: fmt.Errorf("long running recognize failed: %w", err),
			}
			return
		}

		// Process results
		for _, result := range resp.Results {
			if len(result.Alternatives) == 0 {
				continue
			}

			alt := result.Alternatives[0]
			results <- Result{
				Transcript: alt.Transcript,
				Confidence: alt.Confidence,
				IsFinal:    true,
				Words:      p.convertWords(alt.Words),
			}
		}
	}()

	return results, nil
}

// Close closes the Google STT client.
func (p *GoogleProviderV2) Close() error {
	if err := p.client.Close(); err != nil {
		return fmt.Errorf("failed to close speech client: %w", err)
	}
	p.logger.Info("google stt client closed")
	return nil
}

// Name returns the provider name.
func (p *GoogleProviderV2) Name() string {
	return "google-v2"
}

// getEncoding converts string encoding to Google encoding enum.
func (p *GoogleProviderV2) getEncoding(encoding string) speechpb.RecognitionConfig_AudioEncoding {
	switch encoding {
	case "LINEAR16", "PCM":
		return speechpb.RecognitionConfig_LINEAR16
	case "FLAC":
		return speechpb.RecognitionConfig_FLAC
	case "MULAW":
		return speechpb.RecognitionConfig_MULAW
	case "AMR":
		return speechpb.RecognitionConfig_AMR
	case "AMR_WB":
		return speechpb.RecognitionConfig_AMR_WB
	case "OGG_OPUS":
		return speechpb.RecognitionConfig_OGG_OPUS
	case "SPEEX_WITH_HEADER_BYTE":
		return speechpb.RecognitionConfig_SPEEX_WITH_HEADER_BYTE
	case "WEBM_OPUS":
		return speechpb.RecognitionConfig_WEBM_OPUS
	case "MP3":
		return speechpb.RecognitionConfig_MP3
	default:
		return speechpb.RecognitionConfig_ENCODING_UNSPECIFIED
	}
}

// convertWords converts Google word info to our Word structure.
func (p *GoogleProviderV2) convertWords(words []*speechpb.WordInfo) []Word {
	result := make([]Word, len(words))
	for i, w := range words {
		result[i] = Word{
			Word:       w.Word,
			Confidence: w.Confidence,
			StartTime:  w.StartTime.AsDuration(),
			EndTime:    w.EndTime.AsDuration(),
		}
	}
	return result
}

// GetSupportedLanguages returns list of supported language codes.
func (p *GoogleProviderV2) GetSupportedLanguages() []string {
	return []string{
		"pt-BR", "pt-PT", // Portuguese
		"en-US", "en-GB", // English
		"es-ES", "es-MX", // Spanish
		"fr-FR",          // French
		"de-DE",          // German
		"it-IT",          // Italian
		"ja-JP",          // Japanese
		"ko-KR",          // Korean
		"zh-CN", "zh-TW", // Chinese
		// Add more as needed
	}
}

// GetSupportedModels returns list of supported models.
func (p *GoogleProviderV2) GetSupportedModels() []string {
	return []string{
		"default",
		"command_and_search",
		"phone_call",
		"video",
		"medical_conversation",
		"medical_dictation",
	}
}
