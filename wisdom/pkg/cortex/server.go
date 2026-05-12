package cortex

import (
	"context"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/google/wisdom/pkg/cortex/v1" // Assuming this will be the path when compiled
)

// Server implements the gRPC Cortex service.
type Server struct {
	pb.UnimplementedCortexServer
	engine *PostgresEngine
}

// NewServer creates a new Cortex Server.
func NewServer(engine *PostgresEngine) *Server {
	return &Server{
		engine: engine,
	}
}

// Check implements the health check interface.
func (s *Server) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	// Simple ping to the DB to check health
	if err := s.engine.db.PingContext(ctx); err != nil {
		return &grpc_health_v1.HealthCheckResponse{
			Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
		}, nil
	}
	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}, nil
}

// Watch implements the health check watch interface.
func (s *Server) Watch(req *grpc_health_v1.HealthCheckRequest, srv grpc_health_v1.Health_WatchServer) error {
	return status.Error(codes.Unimplemented, "watch is not implemented")
}

// Memorize ingests a new Node into the Cortex.
func (s *Server) Memorize(ctx context.Context, req *pb.IngestRequest) (*pb.NodeID, error) {
	if req.Type == "" {
		return nil, status.Error(codes.InvalidArgument, "type is required")
	}

	nodeID := req.Id
	if nodeID == "" {
		nodeID = uuid.NewString()
	}

	payloadMap := make(map[string]any)
	if req.Payload != nil {
		payloadMap = req.Payload.AsMap()
	}

	var ttl *time.Time
	if req.Ttl != nil {
		t := req.Ttl.AsTime()
		ttl = &t
	}

	node := &Node{
		ID:            nodeID,
		Type:          NodeType(req.Type),
		Payload:       payloadMap,
		Confidence:    req.Confidence,
		RequiresHuman: req.RequiresHuman,
		TTL:           ttl,
	}

	if err := s.engine.Memorize(ctx, node); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to memorize node: %v", err)
	}

	return &pb.NodeID{Id: nodeID}, nil
}

// Recall retrieves a Node and its surrounding context.
func (s *Server) Recall(ctx context.Context, req *pb.RecallRequest) (*pb.CognitionResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	// Currently ignoring Depth and just doing direct neighbors
	cognition, err := s.engine.Recall(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to recall node: %v", err)
	}
	if cognition == nil || cognition.Center == nil {
		return nil, status.Error(codes.NotFound, "node not found")
	}

	centerPB, err := nodeToPB(cognition.Center)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert node: %v", err)
	}

	outEdgesPB := make([]*pb.Edge, 0, len(cognition.OutEdges))
	for _, e := range cognition.OutEdges {
		outEdgesPB = append(outEdgesPB, edgeToPB(e))
	}

	inEdgesPB := make([]*pb.Edge, 0, len(cognition.InEdges))
	for _, e := range cognition.InEdges {
		inEdgesPB = append(inEdgesPB, edgeToPB(e))
	}

	// In a real implementation, we would also fetch and convert the neighbor nodes
	neighborsPB := make([]*pb.Node, 0)

	return &pb.CognitionResponse{
		Center:    centerPB,
		OutEdges:  outEdgesPB,
		InEdges:   inEdgesPB,
		Neighbors: neighborsPB,
	}, nil
}

// QueryHechos retrieves Facts based on metadata filters.
func (s *Server) QueryHechos(ctx context.Context, req *pb.FactRequest) (*pb.FactList, error) {
	facts, err := s.engine.QueryHechos(ctx, req.MetadataFilters)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query facts: %v", err)
	}

	factsPB := make([]*pb.Node, 0, len(facts))
	for _, f := range facts {
		pbNode, err := nodeToPB(f)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert node: %v", err)
		}
		factsPB = append(factsPB, pbNode)
	}

	return &pb.FactList{
		Facts: factsPB,
	}, nil
}

func nodeToPB(n *Node) (*pb.Node, error) {
	payloadStruct, err := structpb.NewStruct(n.Payload)
	if err != nil {
		return nil, err
	}

	var ttl *timestamppb.Timestamp
	if n.TTL != nil {
		ttl = timestamppb.New(*n.TTL)
	}

	return &pb.Node{
		Id:            n.ID,
		Type:          string(n.Type),
		Payload:       payloadStruct,
		Confidence:    n.Confidence,
		RequiresHuman: n.RequiresHuman,
		Ttl:           ttl,
		CreatedAt:     timestamppb.New(n.CreatedAt),
		UpdatedAt:     timestamppb.New(n.UpdatedAt),
	}, nil
}

func edgeToPB(e *Edge) *pb.Edge {
	return &pb.Edge{
		SourceId:  e.SourceID,
		TargetId:  e.TargetID,
		Relation:  string(e.Relation),
		CreatedAt: timestamppb.New(e.CreatedAt),
	}
}
