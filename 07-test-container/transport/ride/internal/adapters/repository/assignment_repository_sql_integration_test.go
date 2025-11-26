//go:build integration_test

package repository_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/yourname/transport/ride/internal/adapters/repository"
	"github.com/yourname/transport/ride/internal/models"
	"github.com/yourname/transport/ride/test_containers"
)

func TestSQLAssignmentRepository_SaveAndFind(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	host, port := test_containers.GetMySqlContainer(ctx, "testdb", "testuser", "testpass", nil)

	dsn := fmt.Sprintf("testuser:testpass@tcp(%s:%s)/testdb?parseTime=true", host, port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	// Schema setup
	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS assignments (
	    id VARCHAR(50) PRIMARY KEY,
	    vehicle_id VARCHAR(50),
	    route_id VARCHAR(50),
	    starts_at DATETIME,
	    status VARCHAR(20)
	);`)

	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	repo := repository.NewSQLAssignmentRepository(db)

	assignment := models.Assignment{
		ID:        "A1",
		VehicleID: "V1",
		RouteID:   "R1",
		StartsAt:  time.Now(),
		Status:    "pending",
	}

	_, err = repo.Save(context.Background(), assignment)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	got, err := repo.FindByID(context.Background(), "A1")
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}

	if got.ID != assignment.ID {
		t.Errorf("expected ID %s, got %s", assignment.ID, got.ID)
	}
}
