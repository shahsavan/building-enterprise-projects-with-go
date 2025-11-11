package ports

import (
	"context"

	"github.com/yourname/transport/ride/internal/core/domain"
)

type AssignmentService interface {
	Create(ctx context.Context, a domain.Assignment) (domain.Assignment, error)
	GetByID(ctx context.Context, id string) (domain.Assignment, error)
	List(ctx context.Context, status *string) ([]domain.Assignment, error)
}
