package ports

import (
	"context"

	"github.com/yourname/transport/ride/internal/core/domain"
)

type AssignmentRepository interface {
	Save(ctx context.Context, a domain.Assignment) (domain.Assignment, error)
	FindByID(ctx context.Context, id string) (domain.Assignment, error)
	FindAll(ctx context.Context, status *string) ([]domain.Assignment, error)
}
