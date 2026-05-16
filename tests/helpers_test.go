package tests

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type testPool struct {
	pool *pgxpool.Pool
}

func (tp testPool) Pool() *pgxpool.Pool {
	return tp.pool
}

func getTestPool(t *testing.T) testPool {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://waf:waf@127.0.0.1:5432/waf_test?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("skipping integration test: database not available: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		t.Skipf("skipping integration test: database not reachable: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	return testPool{pool: pool}
}

func TestDatabaseConnection(t *testing.T) {
	pool := getTestPool(t)
	ctx := context.Background()

	var result int
	err := pool.Pool().QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		t.Fatalf("database query failed: %v", err)
	}
	if result != 1 {
		t.Fatalf("expected 1, got %d", result)
	}
	fmt.Printf("database connection OK\n")
}
