// Package masteryv1 contains the gRPC stubs for the Wisdom-Mastery service.
// These are hand-written stubs that will be replaced by protoc-generated code
// once the proto generation pipeline (scripts/gen_proto.sh) is executed.
package masteryv1

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ─── Unimplemented Server ─────────────────────────────────────────────────────

type UnimplementedMasteryServer struct{}

func (UnimplementedMasteryServer) RecordEngagement(context.Context, *TraceEvent) (*TraceUpdate, error) {
	return nil, nil
}
func (UnimplementedMasteryServer) GetWeaknesses(context.Context, *UserRequest) (*ConceptList, error) {
	return nil, nil
}
func (UnimplementedMasteryServer) GetStrengths(context.Context, *UserRequest) (*ConceptList, error) {
	return nil, nil
}
func (UnimplementedMasteryServer) SyncAnkiReviews(context.Context, *AnkiReviewBatch) (*SyncResult, error) {
	return nil, nil
}
func (UnimplementedMasteryServer) GetDueCards(context.Context, *UserRequest) (*DueCardList, error) {
	return nil, nil
}
func (UnimplementedMasteryServer) ScheduleNextReview(context.Context, *ReviewOutcome) (*ScheduleResult, error) {
	return nil, nil
}

// ─── Server Interface ─────────────────────────────────────────────────────────

type MasteryServer interface {
	RecordEngagement(context.Context, *TraceEvent) (*TraceUpdate, error)
	GetWeaknesses(context.Context, *UserRequest) (*ConceptList, error)
	GetStrengths(context.Context, *UserRequest) (*ConceptList, error)
	SyncAnkiReviews(context.Context, *AnkiReviewBatch) (*SyncResult, error)
	GetDueCards(context.Context, *UserRequest) (*DueCardList, error)
	ScheduleNextReview(context.Context, *ReviewOutcome) (*ScheduleResult, error)
}

func RegisterMasteryServer(s grpc.ServiceRegistrar, srv MasteryServer) {}

// ─── Messages ─────────────────────────────────────────────────────────────────

type TraceEvent struct {
	UserId     string                 `json:"user_id"`
	NodeId     string                 `json:"node_id"`
	Score      int32                  `json:"score"`
	Context    string                 `json:"context"`
	ReviewedAt *timestamppb.Timestamp `json:"reviewed_at"`
}

type TraceUpdate struct {
	NodeId      string  `json:"node_id"`
	MasteryScore float64 `json:"mastery_score"`
	NewStatus   string  `json:"new_status"`
}

type UserRequest struct {
	UserId string `json:"user_id"`
	Limit  int32  `json:"limit"`
}

type Concept struct {
	NodeId       string                 `json:"node_id"`
	Title        string                 `json:"title"`
	MasteryScore float64                `json:"mastery_score"`
	RetentionRate float64               `json:"retention_rate"`
	LastReviewedAt *timestamppb.Timestamp `json:"last_reviewed_at"`
}

type ConceptList struct {
	Concepts []*Concept `json:"concepts"`
}

type AnkiReview struct {
	AnkiCardId  string                 `json:"anki_card_id"`
	WisdomNodeId string                `json:"wisdom_node_id"`
	Grade       int32                  `json:"grade"`
	ReviewId    string                 `json:"review_id"`
	ReviewedAt  *timestamppb.Timestamp `json:"reviewed_at"`
}

type AnkiReviewBatch struct {
	UserId  string       `json:"user_id"`
	Reviews []*AnkiReview `json:"reviews"`
}

type SyncResult struct {
	SyncedCount  int32 `json:"synced_count"`
	SkippedCount int32 `json:"skipped_count"`
	ErrorCount   int32 `json:"error_count"`
}

type DueCard struct {
	NodeId       string                 `json:"node_id"`
	Title        string                 `json:"title"`
	MasteryScore float64                `json:"mastery_score"`
	DueAt        *timestamppb.Timestamp `json:"due_at"`
}

type DueCardList struct {
	Cards []*DueCard `json:"cards"`
}

type ReviewOutcome struct {
	UserId    string `json:"user_id"`
	NodeId    string `json:"node_id"`
	Grade     int32  `json:"grade"`
	Scheduler string `json:"scheduler"`
}

type ScheduleResult struct {
	NodeId       string                 `json:"node_id"`
	NextReviewAt *timestamppb.Timestamp `json:"next_review_at"`
	IntervalDays int32                  `json:"interval_days"`
}

// ─── Client Interface ─────────────────────────────────────────────────────────

type MasteryClient interface {
	RecordEngagement(ctx context.Context, in *TraceEvent, opts ...grpc.CallOption) (*TraceUpdate, error)
	GetWeaknesses(ctx context.Context, in *UserRequest, opts ...grpc.CallOption) (*ConceptList, error)
	GetStrengths(ctx context.Context, in *UserRequest, opts ...grpc.CallOption) (*ConceptList, error)
	SyncAnkiReviews(ctx context.Context, in *AnkiReviewBatch, opts ...grpc.CallOption) (*SyncResult, error)
	GetDueCards(ctx context.Context, in *UserRequest, opts ...grpc.CallOption) (*DueCardList, error)
	ScheduleNextReview(ctx context.Context, in *ReviewOutcome, opts ...grpc.CallOption) (*ScheduleResult, error)
}

type masteryClient struct{ cc grpc.ClientConnInterface }

func NewMasteryClient(cc grpc.ClientConnInterface) MasteryClient {
	return &masteryClient{cc}
}

func (c *masteryClient) RecordEngagement(ctx context.Context, in *TraceEvent, opts ...grpc.CallOption) (*TraceUpdate, error) {
	return nil, nil
}
func (c *masteryClient) GetWeaknesses(ctx context.Context, in *UserRequest, opts ...grpc.CallOption) (*ConceptList, error) {
	return nil, nil
}
func (c *masteryClient) GetStrengths(ctx context.Context, in *UserRequest, opts ...grpc.CallOption) (*ConceptList, error) {
	return nil, nil
}
func (c *masteryClient) SyncAnkiReviews(ctx context.Context, in *AnkiReviewBatch, opts ...grpc.CallOption) (*SyncResult, error) {
	return nil, nil
}
func (c *masteryClient) GetDueCards(ctx context.Context, in *UserRequest, opts ...grpc.CallOption) (*DueCardList, error) {
	return nil, nil
}
func (c *masteryClient) ScheduleNextReview(ctx context.Context, in *ReviewOutcome, opts ...grpc.CallOption) (*ScheduleResult, error) {
	return nil, nil
}
