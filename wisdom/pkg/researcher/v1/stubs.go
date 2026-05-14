// Package researcherv1 contains the gRPC stubs for the Wisdom-Researcher service.
// These are hand-written stubs, replaced by protoc-generated code once
// scripts/gen_proto.sh is executed.
package researcherv1

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ─── Unimplemented Server ─────────────────────────────────────────────────────

type UnimplementedResearcherServer struct{}

func (UnimplementedResearcherServer) Investigate(*InvestigateRequest, Researcher_InvestigateServer) error {
	return nil
}
func (UnimplementedResearcherServer) SubscribeFeed(context.Context, *FeedRequest) (*FeedAck, error) {
	return nil, nil
}
func (UnimplementedResearcherServer) IngestBook(context.Context, *BookRequest) (*BookAck, error) {
	return nil, nil
}
func (UnimplementedResearcherServer) GetJobStatus(context.Context, *JobStatusRequest) (*JobStatus, error) {
	return nil, nil
}

// ─── Server Interface ─────────────────────────────────────────────────────────

type ResearcherServer interface {
	Investigate(*InvestigateRequest, Researcher_InvestigateServer) error
	SubscribeFeed(context.Context, *FeedRequest) (*FeedAck, error)
	IngestBook(context.Context, *BookRequest) (*BookAck, error)
	GetJobStatus(context.Context, *JobStatusRequest) (*JobStatus, error)
}

// Researcher_InvestigateServer is the streaming server interface for Investigate.
type Researcher_InvestigateServer interface {
	Send(*ResearchSignal) error
	Context() context.Context
	grpc.ServerStream
}

func RegisterResearcherServer(s grpc.ServiceRegistrar, srv ResearcherServer) {}

// ─── Messages ─────────────────────────────────────────────────────────────────

type InvestigateRequest struct {
	Topic  string `json:"topic"`
	Domain string `json:"domain"`
	Depth  int32  `json:"depth"`
	UserId string `json:"user_id"`
}

type ResearchSignal struct {
	JobId     string                 `json:"job_id"`
	Source    string                 `json:"source"`
	Content   string                 `json:"content"`
	NodeType  string                 `json:"node_type"`
	Tags      []string               `json:"tags"`
	ScrapedAt *timestamppb.Timestamp `json:"scraped_at"`
}

type FeedRequest struct {
	Url    string `json:"url"`
	Domain string `json:"domain"`
	UserId string `json:"user_id"`
}

type FeedAck struct {
	FeedId  string `json:"feed_id"`
	Success bool   `json:"success"`
}

type BookRequest struct {
	SourceUri string `json:"source_uri"`
	Domain    string `json:"domain"`
	UserId    string `json:"user_id"`
}

type BookAck struct {
	JobId  string `json:"job_id"`
	Queued bool   `json:"queued"`
}

type JobStatusRequest struct {
	JobId string `json:"job_id"`
}

type JobStatus struct {
	JobId        string                 `json:"job_id"`
	Status       string                 `json:"status"`
	Progress     int32                  `json:"progress_pct"`
	EtaSeconds   int32                  `json:"eta_seconds"`
	ErrorMessage string                 `json:"error_message"`
	StartedAt    *timestamppb.Timestamp `json:"started_at"`
	FinishedAt   *timestamppb.Timestamp `json:"finished_at"`
}
