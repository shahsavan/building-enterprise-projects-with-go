package service

import (
	"context"
	"errors"

	"github.com/yourname/transport/ride/internal/core/domain"
	"github.com/yourname/transport/ride/internal/core/ports"
)

type assignmentService struct {
	// Could depend on repository ports
	assignmentRepo ports.AssignmentRepository
}

func NewAssignmentService(repo ports.AssignmentRepository) ports.AssignmentService {
	return &assignmentService{assignmentRepo: repo}
}

func (s *assignmentService) Create(ctx context.Context, a domain.Assignment) (domain.Assignment, error) {
	if a.VehicleID == "" {
		return domain.Assignment{}, errors.New("vehicle ID required")
	}
	return s.assignmentRepo.Save(ctx, a)
}

func (s *assignmentService) GetByID(ctx context.Context, id string) (domain.Assignment, error) {
	return s.assignmentRepo.FindByID(ctx, id)
}

func (s *assignmentService) List(ctx context.Context, status *string) ([]domain.Assignment, error) {
	return s.assignmentRepo.FindAll(ctx, status)
}
