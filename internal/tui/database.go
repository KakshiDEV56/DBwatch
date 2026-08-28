package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/KakshiDEV56/DBwatch/internal/collector"
	"github.com/KakshiDEV56/DBwatch/internal/config"
)

// DBState is the live, per-database runtime state the dashboard renders:
// connection pool, collectors, latest snapshots, and each panel's log
// stream. One exists per configured database and all are polled every
// interval regardless of which one is currently selected.
type DBState struct {
	Name   string
	Region string
	DSN    string

	Pool         *pgxpool.Pool
	Connections  *collector.ConnectionsCollector
	Cache        *collector.CacheCollector
	Queries      *collector.QueriesCollector
	Locks        *collector.LocksCollector
	Transactions *collector.TransactionsCollector
	Activity     *collector.ActivityCollector

	ConnectErr    error
	Bootstrapped  bool
	QueryExtWarn  string
	Version       string
	Host          string
	Port          uint16
	Database      string
	User          string
	PostmasterUp  time.Time
	HasPostmaster bool

	ConnStats   collector.ConnectionStats
	ConnErr     error
	ConnLoaded  bool
	ConnHistory []float64 // recent UtilizationPercent readings, oldest first

	CacheStats   collector.CacheStats
	CacheErr     error
	CacheLoaded  bool
	CacheHistory []float64 // recent HitRatio readings, oldest first

	QueryStats  []collector.QueryStat
	QueryErr    error
	QueryLoaded bool

	LockStats  []collector.BlockedLock
	LockErr    error
	LockLoaded bool

	TxStats  []collector.LongTransaction
	TxErr    error
	TxLoaded bool

	// Load: derived rates from consecutive ActivityStats snapshots. Not
	// meaningful until two snapshots exist (LoadLoaded).
	ActivityErr    error
	LoadLoaded     bool
	XactPerSec     float64
	TuplesPerSec   float64
	TempBytesRate  float64
	Deadlocks      int64
	prevActivity   collector.ActivityStats
	prevActivityAt time.Time

	Logs map[PanelKind]*LogBuffer

	// change-detection state, used to decide what's worth logging rather
	// than re-logging an unchanged "healthy" reading every tick.
	connLogged     bool
	connLastSev    Severity
	cacheLogged    bool
	cacheLastSev   Severity
	queryLastCall  map[string]int64
	queryBaseline  bool
	queryErrLogged bool
	queryLastSev   Severity
	locksLogged    bool
	locksLastState string
	txLogged       bool
	txLastState    string
}

// NewDBState creates the runtime state for one configured database. The
// caller is responsible for setting Pool and the five collectors once
// connected.
func NewDBState(target config.Database) *DBState {
	logs := make(map[PanelKind]*LogBuffer, len(allPanels))
	for _, p := range allPanels {
		logs[p] = &LogBuffer{}
	}
	return &DBState{
		Name:          target.Name,
		Region:        target.Region,
		DSN:           target.DSN,
		Logs:          logs,
		queryLastCall: make(map[string]int64),
	}
}

// ConnectDatabase builds the runtime state for one configured database
// and opens its connection pool. Used identically whether the database
// came from startup config or was just typed into the add-database
// overlay at runtime -- there's only one code path for "start watching a
// database," so the two can never drift apart.
//
// pgxpool.New only parses the DSN -- it does not connect eagerly, so it
// failing means a malformed DSN, not an unreachable server. Reachability
// (and recovery from unreachability) is handled by DBState.Bootstrap,
// retried every tick from the TUI's poll loop -- a database that's down
// now, or down at add-time, is picked up automatically once it responds.
func ConnectDatabase(ctx context.Context, target config.Database) *DBState {
	db := NewDBState(target)

	pool, err := pgxpool.New(ctx, target.DSN)
	if err != nil {
		db.ConnectErr = fmt.Errorf("connect: %w", err)
		return db
	}

	db.Pool = pool
	db.Connections = collector.NewConnectionsCollector(pool)
	db.Cache = collector.NewCacheCollector(pool)
	db.Locks = collector.NewLocksCollector(pool)
	db.Transactions = collector.NewTransactionsCollector(pool)
	db.Queries = collector.NewQueriesCollector(pool, 5)
	db.Activity = collector.NewActivityCollector(pool)
	return db
}

// Uptime returns how long the Postgres server has been up, if known.
func (d *DBState) Uptime() (time.Duration, bool) {
	if !d.HasPostmaster {
		return 0, false
	}
	return time.Since(d.PostmasterUp), true
}

// OverallSeverity is the worst severity across every panel — drives the
// sidebar database list's status dot.
func (d *DBState) OverallSeverity() Severity {
	if d.ConnectErr != nil {
		return SeverityError
	}
	worst := SeverityOK
	for _, p := range allPanels {
		if sev := d.PanelSeverity(p); sev > worst {
			worst = sev
		}
	}
	return worst
}

func (d *DBState) PanelSeverity(k PanelKind) Severity {
	switch k {
	case PanelConnections:
		if d.ConnErr != nil {
			return SeverityError
		}
		if !d.ConnLoaded {
			return SeverityInfo
		}
		pct := d.ConnStats.UtilizationPercent()
		switch {
		case pct >= 90:
			return SeverityCritical // near exhaustion -- genuinely severe
		case pct >= 80:
			return SeverityDegraded
		case pct >= 70:
			return SeverityWarning
		default:
			return SeverityOK
		}
	case PanelCache:
		if d.CacheErr != nil {
			return SeverityError
		}
		if !d.CacheLoaded {
			return SeverityInfo
		}
		pct := d.CacheStats.HitRatio()
		switch {
		case pct >= 99:
			return SeverityOK
		case pct >= 95:
			return SeverityWarning
		case pct >= 85:
			return SeverityDegraded
		default:
			// A cache miss rate this high is a real problem, but it's
			// not "database unavailable" -- red is reserved for that,
			// per dbwatch's color semantics.
			return SeverityDegraded
		}
	case PanelQueries:
		if d.QueryExtWarn != "" {
			return SeverityWarning
		}
		if d.QueryErr != nil {
			return SeverityError
		}
		return SeverityInfo
	case PanelLocks:
		if d.LockErr != nil {
			return SeverityError
		}
		if !d.LockLoaded {
			return SeverityInfo
		}
		if len(d.LockStats) > 0 {
			// Blocking is worth flagging, not alarming over -- red is
			// reserved for the database actually being in trouble
			// (unreachable, near connection exhaustion), not for one
			// query waiting on another.
			return SeverityWarning
		}
		return SeverityOK
	case PanelTransactions:
		if d.TxErr != nil {
			return SeverityError
		}
		if !d.TxLoaded {
			return SeverityInfo
		}
		if len(d.TxStats) > 0 {
			return SeverityWarning
		}
		return SeverityOK
	}
	return SeverityInfo
}

const versionQuery = `SELECT current_setting('server_version'), pg_postmaster_start_time()`

// Bootstrap fetches server identity (version, uptime) and enables
// pg_stat_statements. It is retried every tick until it succeeds, rather
// than running once at startup — a database that's unreachable when
// dbwatch launches must still be picked up automatically once it comes
// back, exactly like a database that goes down mid-session.
func (d *DBState) Bootstrap(ctx context.Context) error {
	if d.Pool == nil {
		return fmt.Errorf("no pool")
	}
	cfg := d.Pool.Config().ConnConfig
	d.Host, d.Port, d.Database, d.User = cfg.Host, cfg.Port, cfg.Database, cfg.User

	var version string
	var start time.Time
	if err := d.Pool.QueryRow(ctx, versionQuery).Scan(&version, &start); err != nil {
		return fmt.Errorf("identity query: %w", err)
	}
	d.Version = version
	d.PostmasterUp = start
	d.HasPostmaster = true

	if d.Queries != nil {
		if err := d.Queries.EnsureExtension(ctx); err != nil {
			d.QueryExtWarn = err.Error()
		} else {
			d.QueryExtWarn = ""
		}
	}
	return nil
}
