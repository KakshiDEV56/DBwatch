package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// LongTransactionThreshold is the minimum transaction age reported by the
// transactions collector.
const LongTransactionThreshold = 30 * time.Second

// LongTransaction describes a transaction that has been open longer than
// LongTransactionThreshold.
type LongTransaction struct {
	PID         int32
	Application string
	State       string
	Duration    time.Duration
	Query       string
}

type TransactionsCollector struct {
	pool *pgxpool.Pool
}

func NewTransactionsCollector(pool *pgxpool.Pool) *TransactionsCollector {
	return &TransactionsCollector{pool: pool}
}

const transactionsQuery = `
SELECT pid, coalesce(application_name, ''), state, now() - xact_start, query
FROM pg_stat_activity
WHERE xact_start IS NOT NULL
	AND pid <> pg_backend_pid()
	AND datname = current_database()
	AND now() - xact_start > $1
ORDER BY xact_start ASC
`

func (c *TransactionsCollector) Collect(ctx context.Context) ([]LongTransaction, error) {
	rows, err := c.pool.Query(ctx, transactionsQuery, LongTransactionThreshold)
	if err != nil {
		return nil, fmt.Errorf("query long-running transactions: %w", err)
	}
	defer rows.Close()

	var txs []LongTransaction
	for rows.Next() {
		var t LongTransaction
		if err := rows.Scan(&t.PID, &t.Application, &t.State, &t.Duration, &t.Query); err != nil {
			return nil, fmt.Errorf("scan long-running transaction row: %w", err)
		}
		txs = append(txs, t)
	}
	return txs, rows.Err()
}
