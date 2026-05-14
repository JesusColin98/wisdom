// Package stt implements the Google Cloud STT V2 middleware for Wisdom-Thalamus.
// It receives audio bytes from the client, transcribes them via Cloud STT V2,
// and publishes the transcription as a wisdom.voice.transcribed Pub/Sub event
// for the ADK Router to consume.
//
// Architecture:
//   Client → Thalamus HTTP /transcribe → Cloud STT V2 → Pub/Sub → ADK Router
package stt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	speech "cloud.google.com/go/speech/apiv2"
	"cloud.google.com/go/speech/apiv2/speechpb"
	"cloud.google.com/go/pubsub"
	"github.com/google/uuid"
)

// TranscribeRequest is the inbound request from the Portal or mobile client.
type TranscribeRequest struct {
	AudioBytes  []byte `json:"audio_bytes"`  // Raw audio (LINEAR16 or WEBM_OPUS)
	LanguageCode string `json:"language_code"` // BCP-47 code, e.g. "en-US", "es-MX"
	UserID      string `json:"user_id"`
	SessionID   string `json:"session_id"`
	SampleRate  int    `json:"sample_rate"` // Hz, e.g. 16000
	Encoding    string `json:"encoding"`    // "LINEAR16" or "WEBM_OPUS"
}

// TranscribeResult is returned synchronously to the caller.
type TranscribeResult struct {
	Text       string  `json:"text"`
	Confidence float32 `json:"confidence"`
	Language   string  `json:"language"`
	SessionID  string  `json:"session_id"`
	EventID    string  `json:"event_id"` // Pub/Sub message ID for tracing.
}

// VoiceTranscribedEvent is published to Pub/Sub for the ADK Router.
type VoiceTranscribedEvent struct {
	Type        string  `json:"type"`         // "wisdom.voice.transcribed"
	Text        string  `json:"text"`
	UserID      string  `json:"user_id"`
	SessionID   string  `json:"session_id"`
	Confidence  float32 `json:"confidence"`
	LanguageCode string `json:"language_code"`
	Timestamp   string  `json:"timestamp"`
}

// STTMiddleware handles audio transcription via Cloud STT V2 and event publishing.
type STTMiddleware struct {
	sttClient   *speech.Client
	pubsubCli   *pubsub.Client
	voiceTopic  *pubsub.Topic
	gcpProject  string
	recognizer  string // Cloud STT V2 recognizer resource name.
}

// NewSTTMiddleware creates a new STT middleware instance.
//
// recognizerID: The Cloud STT V2 recognizer ID created in your GCP project,
//   e.g. "wisdom-voice-recognizer". If empty, uses an ad-hoc recognizer.
func NewSTTMiddleware(gcpProject, recognizerID string) (*STTMiddleware, error) {
	ctx := context.Background()

	sttCli, err := speech.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("speech.NewClient: %w", err)
	}

	psCli, err := pubsub.NewClient(ctx, gcpProject)
	if err != nil {
		return nil, fmt.Errorf("pubsub.NewClient: %w", err)
	}

	recognizerName := fmt.Sprintf("projects/%s/locations/global/recognizers/%s", gcpProject, recognizerID)
	if recognizerID == "" {
		// Use the wildcard recognizer with inline config (no pre-created recognizer needed).
		recognizerName = fmt.Sprintf("projects/%s/locations/global/recognizers/_", gcpProject)
	}

	return &STTMiddleware{
		sttClient:  sttCli,
		pubsubCli:  psCli,
		voiceTopic: psCli.Topic("wisdom.voice.transcribed"),
		gcpProject: gcpProject,
		recognizer: recognizerName,
	}, nil
}

// Transcribe performs synchronous STT and publishes the result to Pub/Sub.
// Returns immediately with the transcription — the ADK Router picks up the event
// asynchronously from Pub/Sub.
func (s *STTMiddleware) Transcribe(ctx context.Context, req *TranscribeRequest) (*TranscribeResult, error) {
	if len(req.AudioBytes) == 0 {
		return nil, fmt.Errorf("audio_bytes is required")
	}
	if req.UserID == "" {
		return nil, fmt.Errorf("user_id is required")
	}
	if req.SessionID == "" {
		req.SessionID = uuid.NewString()
	}
	if req.LanguageCode == "" {
		req.LanguageCode = "en-US"
	}

	// Map encoding string to protobuf enum.
	encoding := speechpb.AutoDetectDecodingConfig{}
	_ = encoding // Use auto-detection for now (handles LINEAR16, WEBM_OPUS, FLAC).

	// Build the STT V2 recognition request.
	sttReq := &speechpb.RecognizeRequest{
		Recognizer: s.recognizer,
		Config: &speechpb.RecognitionConfig{
			DecodingConfig: &speechpb.RecognitionConfig_AutoDecodingConfig{
				AutoDecodingConfig: &speechpb.AutoDetectDecodingConfig{},
			},
			LanguageCodes:      []string{req.LanguageCode},
			Model:              "long",        // "long" for dictation, "latest_short" for commands.
			EnableWordTimeOffsets: false,
			Features: &speechpb.RecognitionFeatures{
				EnableAutomaticPunctuation: true,
				ProfanityFilter:            false,
			},
		},
		AudioSource: &speechpb.RecognizeRequest_Content{
			Content: req.AudioBytes,
		},
	}

	resp, err := s.sttClient.Recognize(ctx, sttReq)
	if err != nil {
		return nil, fmt.Errorf("STT Recognize: %w", err)
	}

	// Extract best transcript.
	var transcript string
	var confidence float32

	for _, result := range resp.Results {
		if len(result.Alternatives) == 0 {
			continue
		}
		best := result.Alternatives[0]
		if best.Confidence > confidence {
			transcript += best.Transcript + " "
			confidence = best.Confidence
		}
	}
	transcript = trimSpace(transcript)

	if transcript == "" {
		return nil, fmt.Errorf("STT returned no transcript (audio may be too short or silent)")
	}

	log.Printf("STT: transcribed '%s' (%.2f confidence, lang=%s)", transcript, confidence, req.LanguageCode)

	// Publish voice.transcribed event to Pub/Sub for ADK Router.
	eventID, err := s.publishVoiceEvent(ctx, VoiceTranscribedEvent{
		Type:         "wisdom.voice.transcribed",
		Text:         transcript,
		UserID:       req.UserID,
		SessionID:    req.SessionID,
		Confidence:   confidence,
		LanguageCode: req.LanguageCode,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		// Non-fatal: return transcript even if Pub/Sub fails.
		log.Printf("WARN: failed to publish voice event: %v", err)
	}

	return &TranscribeResult{
		Text:       transcript,
		Confidence: confidence,
		Language:   req.LanguageCode,
		SessionID:  req.SessionID,
		EventID:    eventID,
	}, nil
}

// publishVoiceEvent serializes and publishes a VoiceTranscribedEvent to Pub/Sub.
func (s *STTMiddleware) publishVoiceEvent(ctx context.Context, event VoiceTranscribedEvent) (string, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return "", fmt.Errorf("json.Marshal: %w", err)
	}

	result := s.voiceTopic.Publish(ctx, &pubsub.Message{
		Data: data,
		Attributes: map[string]string{
			"user_id":    event.UserID,
			"session_id": event.SessionID,
			"language":   event.LanguageCode,
		},
	})

	msgID, err := result.Get(ctx)
	if err != nil {
		return "", fmt.Errorf("pubsub.Publish: %w", err)
	}

	log.Printf("STT: published voice.transcribed event (msg_id=%s, session=%s)", msgID, event.SessionID)
	return msgID, nil
}

// Close releases resources held by the STT middleware.
func (s *STTMiddleware) Close() {
	s.sttClient.Close()
	s.pubsubCli.Close()
}

func trimSpace(s string) string {
	result := bytes.TrimSpace([]byte(s))
	return string(result)
}
