package service

import (
	"context"
	"log"
	"log/slog"

	vehiclepb "github.com/yourname/transport/vehicle/internal/grpc"
	"github.com/yourname/transport/vehicle/internal/ports"
)

// VehicleService is our concrete implementation of the gRPC VehicleServiceServer interface.
// Think of this as the "adapter" layer between gRPC and your domain logic.
// It embeds the UnimplementedVehicleServiceServer to stay forward-compatible:
//   - If a new RPC is later added to the proto file,
//     your code will still compile until you explicitly implement it.
//
// This prevents breaking builds when contracts evolve.
type VehicleService struct {
	vehiclepb.UnimplementedVehicleServiceServer
	// Here you would inject dependencies: for example, a repository, a logger, or a cache.
	// Keeping dependencies as fields makes testing easier (use mocks or fakes).
	repo   ports.VehicleRepository
	logger slog.Logger
}

func NewVehicleService(repo ports.VehicleRepository, logger slog.Logger) *VehicleService {
	return &VehicleService{
		repo:   repo,
		logger: logger,
	}
}

// FindAvailableVehicle is a unary RPC: it takes a request, returns a single response.
// Notice how the parameter is a *FindRequest generated from the proto file.
// This keeps the wire format (protobuf) separate from the core domain.
// Inside, you would call your domain service (business logic) instead of hardcoding.
// For now, we return a dummy bus-123 to keep the example runnable.
func (s *VehicleService) FindAvailableVehicle(ctx context.Context, req *vehiclepb.FindRequest) (*vehiclepb.FindResponse, error) {
	log.Printf("FindAvailableVehicle called with routeId=%s", req.GetRouteId())

	// TODO: connect this to your domain or repository instead of hardcoding
	return &vehiclepb.FindResponse{
		VehicleId: "bus-123",
		Status:    "available",
	}, nil
}

// GetVehicleInfo is another unary RPC.
// This shows how to return multiple fields (id, type, status) from the domain.
// Always prefer explicit fields over maps or "any" types — protobuf enforces this contract.
func (s *VehicleService) GetVehicleInfo(ctx context.Context, req *vehiclepb.InfoRequest) (*vehiclepb.InfoResponse, error) {
	log.Printf("GetVehicleInfo called with vehicleId=%s", req.GetVehicleId())

	// TODO: call your database or cache to fetch vehicle details.
	return &vehiclepb.InfoResponse{
		VehicleId: req.GetVehicleId(),
		Type:      "bus",
		Status:    "available",
	}, nil
}

// StreamAssignments demonstrates a bidirectional streaming RPC.
// Instead of a simple request/response, client and server can keep sending messages.
// The stream.Recv() blocks until the client sends something or closes the connection.
// The stream.Send() pushes an acknowledgement back to the client.
// This pattern is common for live updates, telemetry, or chat-like scenarios.
func (s *VehicleService) StreamAssignments(stream vehiclepb.VehicleService_StreamAssignmentsServer) error {
	log.Println("StreamAssignments started")

	for {
		req, err := stream.Recv()
		if err != nil {
			// When client closes the stream, Recv() returns an EOF error.
			// Always handle this case cleanly.
			return err
		}

		log.Printf("Assignment received: %v", req)

		ack := &vehiclepb.AssignmentAck{
			AssignmentId: req.GetAssignmentId(),
			Accepted:     true,
		}
		if err := stream.Send(ack); err != nil {
			// If sending fails (e.g., client disconnected), exit the loop.
			return err
		}
	}
}
