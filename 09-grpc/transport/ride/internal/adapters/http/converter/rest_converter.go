package converter

import (
	"github.com/yourname/transport/ride/internal/adapters/http/api"
	"github.com/yourname/transport/ride/internal/models"
)

// API -> Domain
func AssignmentToDomain(r api.Assignment) models.Assignment {
	return models.Assignment{
		ID:        *r.Metadata.Id,
		VehicleID: r.VehicleId,
		RouteID:   r.RouteId,
		StartsAt:  r.StartsAt,
		Status:    string(r.Status),
	}
}

// Domain -> API
func AssignmentFromDomain(r models.Assignment) api.Assignment {
	return api.Assignment{
		Metadata: &api.EntityMetadata{
			Id: &r.ID,
		},
		VehicleId: r.VehicleID,
		RouteId:   r.RouteID,
		StartsAt:  r.StartsAt,
		Status:    api.AssignmentStatus(r.Status),
	}
}
