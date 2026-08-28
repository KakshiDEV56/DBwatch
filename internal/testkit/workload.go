package testkit

import (
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type WorkloadReport struct {
	Workers    int
	TargetQPS  int
	ActualQPS  float64
	Duration   time.Duration
	Successful int64
	Failed     int64
}

type idRanges struct {
	maxUserID, maxOrderID int64
}

// workloadQuery is one entry in the mixed query load. weight is relative
// selection frequency.
type workloadQuery struct {
	category string
	weight   int
	run      func(ctx context.Context, pool *pgxpool.Pool, ids idRanges) error
}

var queryMix = []workloadQuery{
	{"read (pk)", 3, func(ctx context.Context, pool *pgxpool.Pool, ids idRanges) error {
		rows, err := pool.Query(ctx, "SELECT * FROM orders WHERE id = $1", randID(ids.maxOrderID))
		if err != nil {
			return err
		}
		rows.Close()
		return rows.Err()
	}},
	{"read (indexed fk)", 3, func(ctx context.Context, pool *pgxpool.Pool, ids idRanges) error {
		rows, err := pool.Query(ctx, "SELECT id FROM orders WHERE user_id = $1", randID(ids.maxUserID))
		if err != nil {
			return err
		}
		rows.Close()
		return rows.Err()
	}},
	{"read (unindexed fk)", 2, func(ctx context.Context, pool *pgxpool.Pool, ids idRanges) error {
		rows, err := pool.Query(ctx, "SELECT id FROM payments WHERE order_id = $1", randID(ids.maxOrderID))
		if err != nil {
			return err
		}
		rows.Close()
		return rows.Err()
	}},
	{"join", 2, func(ctx context.Context, pool *pgxpool.Pool, ids idRanges) error {
		rows, err := pool.Query(ctx, `
			SELECT o.id, u.email, count(oi.id)
			FROM orders o
			JOIN users u ON u.id = o.user_id
			JOIN order_items oi ON oi.order_id = o.id
			WHERE o.id = $1
			GROUP BY o.id, u.email`, randID(ids.maxOrderID))
		if err != nil {
			return err
		}
		rows.Close()
		return rows.Err()
	}},
	{"aggregation", 1, func(ctx context.Context, pool *pgxpool.Pool, ids idRanges) error {
		rows, err := pool.Query(ctx, `
			SELECT user_id, count(*), sum(total_cents)
			FROM orders
			GROUP BY user_id
			ORDER BY user_id
			LIMIT 200`)
		if err != nil {
			return err
		}
		rows.Close()
		return rows.Err()
	}},
	{"insert", 2, func(ctx context.Context, pool *pgxpool.Pool, ids idRanges) error {
		_, err := pool.Exec(ctx, "INSERT INTO events (user_id, name) VALUES ($1, 'workload_event')", randID(ids.maxUserID))
		return err
	}},
	{"update", 2, func(ctx context.Context, pool *pgxpool.Pool, ids idRanges) error {
		_, err := pool.Exec(ctx, "UPDATE orders SET status = status WHERE id = $1", randID(ids.maxOrderID))
		return err
	}},
	{"insert+delete", 1, func(ctx context.Context, pool *pgxpool.Pool, ids idRanges) error {
		_, err := pool.Exec(ctx, `
			WITH ins AS (
				INSERT INTO events (name) VALUES ('workload_scratch') RETURNING id
			)
			DELETE FROM events WHERE id IN (SELECT id FROM ins)`)
		return err
	}},
}

// Workload runs a continuous, realistic mixed query load (reads, writes,
// joins, aggregations, an intentionally-unindexed lookup) against the
// production-shaped schema, rate-limited across `workers` goroutines to
// approximate `qps`, for `duration`.
func Workload(ctx context.Context, cfg Config, dbName string, workers, qps int, duration time.Duration) (WorkloadReport, error) {
	n := NewNarrator("QUERY WORKLOAD")
	report := WorkloadReport{Workers: workers, TargetQPS: qps, Duration: duration}

	pool, err := Connect(ctx, cfg, dbName)
	if err != nil {
		return report, err
	}
	defer pool.Close()

	ids, err := loadRanges(ctx, pool)
	if err != nil {
		return report, err
	}
	n.Step("workers=%d target_qps=%d duration=%s against %s", workers, qps, duration, dbName)
	n.Expect("Top Queries fills in with a real mix (reads/writes/joins/aggregation).\nCache Hit Ratio should stay high. Connections shows ~%d active workers.\nA payments lookup by order_id runs an unindexed sequential scan on\npurpose -- see it show up with a higher mean time than the indexed\norders lookup.", workers)

	perWorkerInterval := time.Duration(float64(workers) / float64(qps) * float64(time.Second))
	if perWorkerInterval <= 0 {
		perWorkerInterval = time.Millisecond
	}

	deadline := time.Now().Add(duration)
	var wg sync.WaitGroup
	var ok, failed int64

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(seed int64) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(seed))
			ticker := time.NewTicker(perWorkerInterval)
			defer ticker.Stop()
			for time.Now().Before(deadline) {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					q := pickQuery(rng)
					if err := q.run(ctx, pool, ids); err != nil {
						atomic.AddInt64(&failed, 1)
					} else {
						atomic.AddInt64(&ok, 1)
					}
				}
			}
		}(int64(w) + time.Now().UnixNano())
	}
	wg.Wait()

	report.Successful, report.Failed = ok, failed
	total := ok + failed
	report.ActualQPS = float64(total) / duration.Seconds()

	n.OK("successful=%d failed=%d actual_qps=%.1f (target %d)", ok, failed, report.ActualQPS, qps)
	return report, nil
}

func loadRanges(ctx context.Context, pool *pgxpool.Pool) (idRanges, error) {
	var ids idRanges
	if err := pool.QueryRow(ctx, "SELECT coalesce(max(id), 1) FROM users").Scan(&ids.maxUserID); err != nil {
		return ids, err
	}
	if err := pool.QueryRow(ctx, "SELECT coalesce(max(id), 1) FROM orders").Scan(&ids.maxOrderID); err != nil {
		return ids, err
	}
	return ids, nil
}

func pickQuery(rng *rand.Rand) workloadQuery {
	total := 0
	for _, q := range queryMix {
		total += q.weight
	}
	r := rng.Intn(total)
	for _, q := range queryMix {
		if r < q.weight {
			return q
		}
		r -= q.weight
	}
	return queryMix[0]
}

func randID(max int64) int64 {
	if max <= 0 {
		return 1
	}
	return rand.Int63n(max) + 1
}
