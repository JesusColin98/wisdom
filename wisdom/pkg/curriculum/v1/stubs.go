// Package curriculumv1 contains the gRPC stubs for the Wisdom-Curriculum service.
// These are hand-written stubs, replaced by protoc-generated code once
// scripts/gen_proto.sh is executed.
package curriculumv1

import (
	"context"

	"google.golang.org/grpc"
)

// ─── Unimplemented Server ─────────────────────────────────────────────────────

type UnimplementedCurriculumServer struct{}

func (UnimplementedCurriculumServer) GeneratePath(context.Context, *PathRequest) (*LearningPath, error) {
	return nil, nil
}
func (UnimplementedCurriculumServer) MapDependencies(context.Context, *NodeList) (*DependencyGraph, error) {
	return nil, nil
}
func (UnimplementedCurriculumServer) AssignDifficulty(context.Context, *DifficultyRequest) (*DifficultyResult, error) {
	return nil, nil
}
func (UnimplementedCurriculumServer) ReprioritizeForUser(context.Context, *ReprioritizeRequest) (*LearningPath, error) {
	return nil, nil
}

// ─── Server Interface ─────────────────────────────────────────────────────────

type CurriculumServer interface {
	GeneratePath(context.Context, *PathRequest) (*LearningPath, error)
	MapDependencies(context.Context, *NodeList) (*DependencyGraph, error)
	AssignDifficulty(context.Context, *DifficultyRequest) (*DifficultyResult, error)
	ReprioritizeForUser(context.Context, *ReprioritizeRequest) (*LearningPath, error)
}

func RegisterCurriculumServer(s grpc.ServiceRegistrar, srv CurriculumServer) {}

// ─── Messages ─────────────────────────────────────────────────────────────────

type PathRequest struct {
	Topic       string `json:"topic"`
	Domain      string `json:"domain"`
	TargetLevel int32  `json:"target_level"`
	UserId      string `json:"user_id"`
}

type Module struct {
	Id              string   `json:"id"`
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	Difficulty      int32    `json:"difficulty"`
	PrerequisiteIds []string `json:"prerequisite_ids"`
	ConceptNodeIds  []string `json:"concept_node_ids"`
}

type LearningPath struct {
	TopicId    string    `json:"topic_id"`
	TopicTitle string    `json:"topic_title"`
	Domain     string    `json:"domain"`
	Modules    []*Module `json:"modules"`
}

type NodeList struct {
	NodeIds []string `json:"node_ids"`
}

type Dependency struct {
	FromNodeId string `json:"from_node_id"`
	ToNodeId   string `json:"to_node_id"`
	Relation   string `json:"relation"`
}

type DependencyGraph struct {
	Edges []*Dependency `json:"edges"`
}

type DifficultyRequest struct {
	NodeId  string `json:"node_id"`
	Content string `json:"content"`
}

type DifficultyResult struct {
	NodeId    string `json:"node_id"`
	Tier      int32  `json:"tier"`
	Rationale string `json:"rationale"`
}

type ReprioritizeRequest struct {
	UserId          string   `json:"user_id"`
	StruggleNodeIds []string `json:"struggle_node_ids"`
}

// ─── Client Interface ─────────────────────────────────────────────────────────

type CurriculumClient interface {
	GeneratePath(ctx context.Context, in *PathRequest, opts ...grpc.CallOption) (*LearningPath, error)
	MapDependencies(ctx context.Context, in *NodeList, opts ...grpc.CallOption) (*DependencyGraph, error)
	AssignDifficulty(ctx context.Context, in *DifficultyRequest, opts ...grpc.CallOption) (*DifficultyResult, error)
	ReprioritizeForUser(ctx context.Context, in *ReprioritizeRequest, opts ...grpc.CallOption) (*LearningPath, error)
}

type curriculumClient struct{ cc grpc.ClientConnInterface }

func NewCurriculumClient(cc grpc.ClientConnInterface) CurriculumClient {
	return &curriculumClient{cc}
}

func (c *curriculumClient) GeneratePath(ctx context.Context, in *PathRequest, opts ...grpc.CallOption) (*LearningPath, error) {
	return nil, nil
}
func (c *curriculumClient) MapDependencies(ctx context.Context, in *NodeList, opts ...grpc.CallOption) (*DependencyGraph, error) {
	return nil, nil
}
func (c *curriculumClient) AssignDifficulty(ctx context.Context, in *DifficultyRequest, opts ...grpc.CallOption) (*DifficultyResult, error) {
	return nil, nil
}
func (c *curriculumClient) ReprioritizeForUser(ctx context.Context, in *ReprioritizeRequest, opts ...grpc.CallOption) (*LearningPath, error) {
	return nil, nil
}
