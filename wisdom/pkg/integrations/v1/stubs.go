// Package integrationsv1 contains the gRPC stubs for the Wisdom-Integrations service.
// These are hand-written stubs, replaced by protoc-generated code once
// scripts/gen_proto.sh is executed.
package integrationsv1

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ─── Unimplemented Server ─────────────────────────────────────────────────────

type UnimplementedIntegrationsServer struct{}

func (UnimplementedIntegrationsServer) CreateNote(context.Context, *NoteRequest) (*IntegrationResult, error) {
	return nil, nil
}
func (UnimplementedIntegrationsServer) CreateCard(context.Context, *CardRequest) (*IntegrationResult, error) {
	return nil, nil
}
func (UnimplementedIntegrationsServer) GetPendingQueue(context.Context, *QueueRequest) (*PendingQueue, error) {
	return nil, nil
}
func (UnimplementedIntegrationsServer) RetryPendingSync(context.Context, *QueueRequest) (*RetryResult, error) {
	return nil, nil
}
func (UnimplementedIntegrationsServer) ProcessAnkiReviews(context.Context, *AnkiReviewBatch) (*AnkiSyncResult, error) {
	return nil, nil
}

// ─── Server Interface ─────────────────────────────────────────────────────────

type IntegrationsServer interface {
	CreateNote(context.Context, *NoteRequest) (*IntegrationResult, error)
	CreateCard(context.Context, *CardRequest) (*IntegrationResult, error)
	GetPendingQueue(context.Context, *QueueRequest) (*PendingQueue, error)
	RetryPendingSync(context.Context, *QueueRequest) (*RetryResult, error)
	ProcessAnkiReviews(context.Context, *AnkiReviewBatch) (*AnkiSyncResult, error)
}

func RegisterIntegrationsServer(s grpc.ServiceRegistrar, srv IntegrationsServer) {}

// ─── Enums ────────────────────────────────────────────────────────────────────

type AnkiCardType int32

const (
	AnkiCardType_BASIC  AnkiCardType = 0
	AnkiCardType_CLOZE  AnkiCardType = 1
)

// ─── Messages ─────────────────────────────────────────────────────────────────

type NoteMetadata struct {
	Title        string   `json:"title"`
	Tags         []string `json:"tags"`
	Aliases      []string `json:"aliases"`
	MasteryScore float64  `json:"mastery_score"`
	Domain       string   `json:"domain"`
}

func (m *NoteMetadata) GetTitle() string        { return m.Title }
func (m *NoteMetadata) GetTags() []string       { return m.Tags }
func (m *NoteMetadata) GetMasteryScore() float64 { return m.MasteryScore }

type NoteRequest struct {
	AgentName     string        `json:"agent_name"`
	UserId        string        `json:"user_id"`
	Metadata      *NoteMetadata `json:"metadata"`
	Content       string        `json:"content"`
	Relationships []string      `json:"relationships"`
	TargetPath    string        `json:"target_path"`
}

type CardRequest struct {
	AgentName    string       `json:"agent_name"`
	UserId       string       `json:"user_id"`
	DeckName     string       `json:"deck_name"`
	CardType     AnkiCardType `json:"card_type"`
	Front        string       `json:"front"`
	Back         string       `json:"back"`
	ClozeText    string       `json:"cloze_text"`
	Extra        string       `json:"extra"`
	Tags         []string     `json:"tags"`
	WisdomNodeId string       `json:"wisdom_node_id"`
}

type IntegrationResult struct {
	Success      bool   `json:"success"`
	ExternalId   string `json:"external_id"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message"`
}

type QueueRequest struct {
	UserId string `json:"user_id"`
}

type PendingItem struct {
	ItemId      string                 `json:"item_id"`
	ItemType    string                 `json:"item_type"`
	TargetApp   string                 `json:"target_app"`
	PayloadJson string                 `json:"payload_json"`
	RetryCount  int32                  `json:"retry_count"`
	QueuedAt    *timestamppb.Timestamp `json:"queued_at"`
}

type PendingQueue struct {
	Items      []*PendingItem `json:"items"`
	TotalCount int32          `json:"total_count"`
}

type RetryResult struct {
	Succeeded    int32 `json:"succeeded"`
	Failed       int32 `json:"failed"`
	StillPending int32 `json:"still_pending"`
}

type AnkiReviewEntry struct {
	AnkiCardId   string                 `json:"anki_card_id"`
	WisdomNodeId string                 `json:"wisdom_node_id"`
	Grade        int32                  `json:"grade"`
	ReviewId     string                 `json:"review_id"`
	ReviewedAt   *timestamppb.Timestamp `json:"reviewed_at"`
}

type AnkiReviewBatch struct {
	UserId  string             `json:"user_id"`
	Reviews []*AnkiReviewEntry `json:"reviews"`
}

type AnkiSyncResult struct {
	SyncedCount  int32 `json:"synced_count"`
	SkippedCount int32 `json:"skipped_count"`
	ErrorCount   int32 `json:"error_count"`
}

// ─── Client Interface ─────────────────────────────────────────────────────────

type IntegrationsClient interface {
	CreateNote(ctx context.Context, in *NoteRequest, opts ...grpc.CallOption) (*IntegrationResult, error)
	CreateCard(ctx context.Context, in *CardRequest, opts ...grpc.CallOption) (*IntegrationResult, error)
	GetPendingQueue(ctx context.Context, in *QueueRequest, opts ...grpc.CallOption) (*PendingQueue, error)
	RetryPendingSync(ctx context.Context, in *QueueRequest, opts ...grpc.CallOption) (*RetryResult, error)
	ProcessAnkiReviews(ctx context.Context, in *AnkiReviewBatch, opts ...grpc.CallOption) (*AnkiSyncResult, error)
}

type integrationsClient struct{ cc grpc.ClientConnInterface }

func NewIntegrationsClient(cc grpc.ClientConnInterface) IntegrationsClient {
	return &integrationsClient{cc}
}
func (c *integrationsClient) CreateNote(ctx context.Context, in *NoteRequest, opts ...grpc.CallOption) (*IntegrationResult, error) {
	return nil, nil
}
func (c *integrationsClient) CreateCard(ctx context.Context, in *CardRequest, opts ...grpc.CallOption) (*IntegrationResult, error) {
	return nil, nil
}
func (c *integrationsClient) GetPendingQueue(ctx context.Context, in *QueueRequest, opts ...grpc.CallOption) (*PendingQueue, error) {
	return nil, nil
}
func (c *integrationsClient) RetryPendingSync(ctx context.Context, in *QueueRequest, opts ...grpc.CallOption) (*RetryResult, error) {
	return nil, nil
}
func (c *integrationsClient) ProcessAnkiReviews(ctx context.Context, in *AnkiReviewBatch, opts ...grpc.CallOption) (*AnkiSyncResult, error) {
	return nil, nil
}
