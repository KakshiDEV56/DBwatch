package testkit

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SeedCounts controls how much data to generate. Zero fields are skipped
// entirely rather than defaulted, so a caller can seed just one table.
type SeedCounts struct {
	Users      int
	Products   int
	Orders     int
	OrderItems int
	Payments   int
	Sessions   int
	Events     int
	AuditLogs  int
}

// DefaultSeedCounts is small enough to seed in a few seconds on a laptop.
// Pass larger numbers explicitly for a real stress run -- the spec's
// suggested "100,000+ users" scale works fine, it's just not the default.
func DefaultSeedCounts() SeedCounts {
	return SeedCounts{
		Users:      20_000,
		Products:   5_000,
		Orders:     100_000,
		OrderItems: 300_000,
		Payments:   100_000,
		Sessions:   40_000,
		Events:     300_000,
		AuditLogs:  300_000,
	}
}

// Seed populates the production-shaped schema (see
// test/postgres/init/002_schema.sql). It's additive/idempotent: it only
// tops up each table up to the requested count rather than truncating, so
// running it twice with the same counts is a cheap no-op.
func Seed(ctx context.Context, pool *pgxpool.Pool, n *Narrator, counts SeedCounts) error {
	steps := []struct {
		label string
		fn    func(context.Context, *pgxpool.Pool, int) error
	}{
		{"users", seedUsers},
		{"products", seedProducts},
		{"orders", seedOrders},
		{"order_items", seedOrderItems},
		{"payments", seedPayments},
		{"sessions", seedSessions},
		{"events", seedEvents},
		{"audit_logs", seedAuditLogs},
	}
	counts_ := []int{counts.Users, counts.Products, counts.Orders, counts.OrderItems, counts.Payments, counts.Sessions, counts.Events, counts.AuditLogs}

	for i, step := range steps {
		target := counts_[i]
		if target <= 0 {
			continue
		}
		start := time.Now()
		if err := step.fn(ctx, pool, target); err != nil {
			return fmt.Errorf("seed %s: %w", step.label, err)
		}
		n.OK("%-12s topped up to %d rows (%s)", step.label, target, time.Since(start).Round(time.Millisecond))
	}
	return nil
}

func tableCount(ctx context.Context, pool *pgxpool.Pool, table string) (int, error) {
	var n int
	err := pool.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&n)
	return n, err
}

func seedUsers(ctx context.Context, pool *pgxpool.Pool, target int) error {
	have, err := tableCount(ctx, pool, "users")
	if err != nil || have >= target {
		return err
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO users (email)
		SELECT 'user' || g || '@example.test'
		FROM generate_series($1::int, $2::int) AS g
		ON CONFLICT (email) DO NOTHING`, have+1, target)
	return err
}

func seedProducts(ctx context.Context, pool *pgxpool.Pool, target int) error {
	have, err := tableCount(ctx, pool, "products")
	if err != nil || have >= target {
		return err
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO products (name, price_cents)
		SELECT 'product-' || g, (random() * 9900 + 100)::int
		FROM generate_series($1::int, $2::int) AS g`, have+1, target)
	return err
}

func seedOrders(ctx context.Context, pool *pgxpool.Pool, target int) error {
	have, err := tableCount(ctx, pool, "orders")
	if err != nil || have >= target {
		return err
	}
	userCount, err := tableCount(ctx, pool, "users")
	if err != nil || userCount == 0 {
		return fmt.Errorf("seed orders: no users to reference (seed users first)")
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO orders (user_id, total_cents, status, created_at)
		SELECT
			(random() * ($3 - 1) + 1)::int,
			(random() * 20000 + 500)::int,
			(ARRAY['pending', 'paid', 'shipped', 'refunded'])[(random() * 3 + 1)::int],
			now() - (random() * interval '180 days')
		FROM generate_series(1, $1::int - $2::int) AS g`, target, have, userCount)
	return err
}

func seedOrderItems(ctx context.Context, pool *pgxpool.Pool, target int) error {
	have, err := tableCount(ctx, pool, "order_items")
	if err != nil || have >= target {
		return err
	}
	orderCount, productCount := 0, 0
	if orderCount, err = tableCount(ctx, pool, "orders"); err != nil || orderCount == 0 {
		return fmt.Errorf("seed order_items: no orders to reference")
	}
	if productCount, err = tableCount(ctx, pool, "products"); err != nil || productCount == 0 {
		return fmt.Errorf("seed order_items: no products to reference")
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO order_items (order_id, product_id, quantity, unit_price_cents)
		SELECT
			(random() * ($3 - 1) + 1)::int,
			(random() * ($4 - 1) + 1)::int,
			(random() * 4 + 1)::int,
			(random() * 9900 + 100)::int
		FROM generate_series(1, $1::int - $2::int) AS g`, target, have, orderCount, productCount)
	return err
}

func seedPayments(ctx context.Context, pool *pgxpool.Pool, target int) error {
	have, err := tableCount(ctx, pool, "payments")
	if err != nil || have >= target {
		return err
	}
	orderCount, err := tableCount(ctx, pool, "orders")
	if err != nil || orderCount == 0 {
		return fmt.Errorf("seed payments: no orders to reference")
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO payments (order_id, amount_cents, status, created_at)
		SELECT
			(random() * ($3 - 1) + 1)::int,
			(random() * 20000 + 500)::int,
			(ARRAY['pending', 'succeeded', 'failed'])[(random() * 2 + 1)::int],
			now() - (random() * interval '180 days')
		FROM generate_series(1, $1::int - $2::int) AS g`, target, have, orderCount)
	return err
}

func seedSessions(ctx context.Context, pool *pgxpool.Pool, target int) error {
	have, err := tableCount(ctx, pool, "sessions")
	if err != nil || have >= target {
		return err
	}
	userCount, err := tableCount(ctx, pool, "users")
	if err != nil || userCount == 0 {
		return fmt.Errorf("seed sessions: no users to reference")
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO sessions (user_id, created_at, expires_at)
		SELECT
			(random() * ($3 - 1) + 1)::int,
			now() - (random() * interval '30 days'),
			now() + interval '7 days'
		FROM generate_series(1, $1::int - $2::int) AS g`, target, have, userCount)
	return err
}

func seedEvents(ctx context.Context, pool *pgxpool.Pool, target int) error {
	have, err := tableCount(ctx, pool, "events")
	if err != nil || have >= target {
		return err
	}
	userCount, err := tableCount(ctx, pool, "users")
	if err != nil || userCount == 0 {
		return fmt.Errorf("seed events: no users to reference")
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO events (user_id, name, occurred_at)
		SELECT
			(random() * ($3 - 1) + 1)::int,
			(ARRAY['page_view', 'signup', 'purchase', 'click', 'logout'])[(random() * 4 + 1)::int],
			now() - (random() * interval '30 days')
		FROM generate_series(1, $1::int - $2::int) AS g`, target, have, userCount)
	return err
}

func seedAuditLogs(ctx context.Context, pool *pgxpool.Pool, target int) error {
	have, err := tableCount(ctx, pool, "audit_logs")
	if err != nil || have >= target {
		return err
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO audit_logs (actor, action, occurred_at)
		SELECT
			'user' || (random() * 9999 + 1)::int,
			(ARRAY['login', 'update_profile', 'delete_item', 'export_data'])[(random() * 3 + 1)::int],
			now() - (random() * interval '90 days')
		FROM generate_series(1, $1::int - $2::int) AS g`, target, have)
	return err
}
