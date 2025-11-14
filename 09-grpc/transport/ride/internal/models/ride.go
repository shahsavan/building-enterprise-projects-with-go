package models

import (
	"time"
)

// Defines values for AssignmentStatus.
const (
	AssignmentStatusActive    AssignmentStatus = "active"
	AssignmentStatusCompleted AssignmentStatus = "completed"
	AssignmentStatusPending   AssignmentStatus = "pending"
)

// Defines values for ListAssignmentsParamsStatus.
const (
	ListAssignmentsParamsStatusActive    ListAssignmentsParamsStatus = "active"
	ListAssignmentsParamsStatusCompleted ListAssignmentsParamsStatus = "completed"
	ListAssignmentsParamsStatusPending   ListAssignmentsParamsStatus = "pending"
)

type Assignment struct {
	ID        string
	VehicleID string
	RouteID   string
	StartsAt  time.Time
	Status    string
}

// AssignmentStatus defines model for Assignment.Status.
type AssignmentStatus string

// EntityMetadata defines model for EntityMetadata.
type EntityMetadata struct {
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	Id        *string    `json:"id,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

// NewAssignment defines model for NewAssignment.
type NewAssignment struct {
	RouteId   string    `json:"routeId"`
	StartsAt  time.Time `json:"startsAt"`
	VehicleId string    `json:"vehicleId"`
}

// BadRequest defines model for BadRequest.
type BadRequest struct {
	Details *string `json:"details,omitempty"`
	Error   *string `json:"error,omitempty"`
}

// NotFound defines model for NotFound.
type NotFound struct {
	Error *string `json:"error,omitempty"`
}

// ListAssignmentsParams defines parameters for ListAssignments.
type ListAssignmentsParams struct {
	Status *ListAssignmentsParamsStatus `form:"status,omitempty" json:"status,omitempty"`
}

// ListAssignmentsParamsStatus defines parameters for ListAssignments.
type ListAssignmentsParamsStatus string

// CreateAssignmentJSONRequestBody defines body for CreateAssignment for application/json ContentType.
type CreateAssignmentJSONRequestBody = NewAssignment
