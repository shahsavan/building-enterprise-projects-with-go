package ports

import (
	"context"

	"github.com/yourname/transport/ride/internal/models"
)

type AssignmentService interface {
	Create(ctx context.Context, a models.Assignment) (models.Assignment, error)
	GetByID(ctx context.Context, id string) (models.Assignment, error)
	List(ctx context.Context, status *string) ([]models.Assignment, error)
}
