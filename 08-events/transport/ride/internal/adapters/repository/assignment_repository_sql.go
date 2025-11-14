package repository

import (
	"context"
	"database/sql"

	"github.com/yourname/transport/ride/internal/ports"
)

type sqlAssignmentRepository struct {
	db *sql.DB
}

func NewSQLAssignmentRepository(db *sql.DB) ports.AssignmentRepository {
	return &sqlAssignmentRepository{db: db}
}

func (r *sqlAssignmentRepository) Save(ctx context.Context, a domain.Assignment) (domain.Assignment, error) {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO assignments (id, vehicle_id, route_id, starts_at, status)
		VALUES (?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
		    vehicle_id = VALUES(vehicle_id),
		    route_id   = VALUES(route_id),
		    starts_at  = VALUES(starts_at),
		    status     = VALUES(status)`,
		a.ID, a.VehicleID, a.RouteID, a.StartsAt, a.Status,
	)
	if err != nil {
		return domain.Assignment{}, err
	}
	return a, nil
}

func (r *sqlAssignmentRepository) FindByID(ctx context.Context, id string) (domain.Assignment, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, vehicle_id, route_id, starts_at, status
		FROM assignments WHERE id = ?`, id,
	)

	var a domain.Assignment
	err := row.Scan(&a.ID, &a.VehicleID, &a.RouteID, &a.StartsAt, &a.Status)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.Assignment{}, nil // caller decides how to handle "not found"
		}
		return domain.Assignment{}, err
	}
	return a, nil
}

func (r *sqlAssignmentRepository) FindAll(ctx context.Context, status *string) ([]domain.Assignment, error) {
	var rows *sql.Rows
	var err error

	if status != nil {
		rows, err = r.db.QueryContext(ctx, `
			SELECT id, vehicle_id, route_id, starts_at, status
			FROM assignments WHERE status = ?`, *status,
		)
	} else {
		rows, err = r.db.QueryContext(ctx, `
			SELECT id, vehicle_id, route_id, starts_at, status
			FROM assignments`,
		)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assignments []domain.Assignment
	for rows.Next() {
		var a domain.Assignment
		if err := rows.Scan(&a.ID, &a.VehicleID, &a.RouteID, &a.StartsAt, &a.Status); err != nil {
			return nil, err
		}
		assignments = append(assignments, a)
	}
	return assignments, rows.Err()
}
