package converter

import (
	"github.com/yourname/transport/ride/internal/adapters/http/api"
	"github.com/yourname/transport/ride/internal/domain"
)

// API -> Domain
func AssignmentToDomain(r api.Assignment) domain.Assignment {
	return domain.Assignment{
		ID:        *r.Id,
		VehicleID: r.VehicleId,
		RouteID:   r.RouteId,
		StartsAt:  r.StartsAt,
		Status:    string(r.Status),
	}
}

// Domain -> API
func AssignmentFromDomain(r domain.Assignment) api.Assignment {
	return api.Assignment{
		Id:        &r.ID,
		VehicleId: r.VehicleID,
		RouteId:   r.RouteID,
		StartsAt:  r.StartsAt,
		Status:    api.AssignmentStatus(r.Status),
	}
}
