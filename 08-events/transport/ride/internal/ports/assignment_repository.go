package ports

import (
	"context"

	"github.com/yourname/transport/ride/internal/models"
)

type AssignmentRepository interface {
	Save(ctx context.Context, a models.Assignment) (bool, error)
	FindByID(ctx context.Context, id string) (models.Assignment, error)
	FindAll(ctx context.Context, status *string) ([]models.Assignment, error)
}
