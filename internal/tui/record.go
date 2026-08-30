package tui

import (
	"fmt"
	"time"

	"github.com/KakshiDEV56/DBwatch/internal/collector"
)

// sparklineHistoryLimit bounds every rolling history slice -- sparklines
// only ever show real samples, and this keeps memory bounded regardless
// of session length.
const sparklineHistoryLimit = 30

func appendHistory(h []float64, v float64) []float64 {
	h = append(h, v)
	if len(h) > sparklineHistoryLimit {
		h = h[len(h)-sparklineHistoryLimit:]
	}
	return h
}

// logEvent appends e to panel's own log stream and, when it's at Warning
// severity or worse, cross-posts the same entry into PanelErrors -- so
// every real problem dbwatch already detects shows up in one merged
// timeline without a second collector re-deriving it.
func (d *DBState) logEvent(panel PanelKind, e LogEntry) {
	e.Panel = panel
	d.Logs[panel].Append(e)
	if panel != PanelErrors && e.Severity >= SeverityWarning {
		d.Logs[PanelErrors].Append(e)
	}
}

// RecordActivity derives per-second rates by diffing this snapshot
// against the previous one -- pg_stat_database's counters are cumulative
// since the last stats reset, not point-in-time, so a single snapshot
// alone can't say anything about current load.
func (d *DBState) RecordActivity(stats collector.ActivityStats, err error) {
	d.ActivityErr = err
	if err != nil {
		if !d.activityErrLogged {
			d.activityErrLogged = true
			d.logEvent(PanelErrors, LogEntry{
				Time: time.Now(), Severity: SeverityError, Panel: PanelErrors,
				Summary: "activity collection error: " + err.Error(),
			})
		}
		return
	}
	d.activityErrLogged = false

	now := time.Now()
	if !d.prevActivityAt.IsZero() {
		elapsed := now.Sub(d.prevActivityAt).Seconds()
		if elapsed > 0 {
			xactDelta := (stats.XactCommit + stats.XactRollback) - (d.prevActivity.XactCommit + d.prevActivity.XactRollback)
			tupDelta := (stats.TupReturned + stats.TupFetched + stats.TupInserted + stats.TupUpdated + stats.TupDeleted) -
				(d.prevActivity.TupReturned + d.prevActivity.TupFetched + d.prevActivity.TupInserted + d.prevActivity.TupUpdated + d.prevActivity.TupDeleted)
			tempDelta := stats.TempBytes - d.prevActivity.TempBytes

			d.XactPerSec = rateOrZero(xactDelta, elapsed)
			d.TuplesPerSec = rateOrZero(tupDelta, elapsed)
			d.TempBytesRate = rateOrZero(tempDelta, elapsed)
			d.LoadLoaded = true
		}

		// Deadlocks is an exact PostgreSQL counter (pg_stat_database), so an
		// increase is a real, discrete event worth its own log entry, not
		// just a number in the Health card.
		if deadlockDelta := stats.Deadlocks - d.prevActivity.Deadlocks; deadlockDelta > 0 {
			d.logEvent(PanelErrors, LogEntry{
				Time: now, Severity: SeverityCritical, Panel: PanelErrors,
				Summary: fmt.Sprintf("%d new deadlock(s) detected (%d total)", deadlockDelta, stats.Deadlocks),
			})
		}
	}
	d.Deadlocks = stats.Deadlocks
	d.prevActivity = stats
	d.prevActivityAt = now
}

func rateOrZero(delta int64, elapsedSeconds float64) float64 {
	if delta < 0 {
		// A counter went backwards -- pg_stat_database was reset
		// (pg_stat_reset(), or the server restarted). Report 0 rather
		// than a nonsensical negative number for this one tick.
		return 0
	}
	return float64(delta) / elapsedSeconds
}

// RecordConnections stores the latest snapshot and appends a log entry on
// first load or when severity changes (including transitioning into or
// out of a collection error), so the log stays a record of events rather
// than a spammed repeat of the same reading — or the same error — every
// tick during a sustained outage.
func (d *DBState) RecordConnections(stats collector.ConnectionStats, err error) {
	d.ConnStats, d.ConnErr, d.ConnLoaded = stats, err, true
	if err != nil {
		d.Bootstrapped = false
	} else {
		d.ConnHistory = appendHistory(d.ConnHistory, stats.UtilizationPercent())
	}

	sev := d.PanelSeverity(PanelConnections)
	if d.connLogged && sev == d.connLastSev {
		return
	}
	d.connLogged, d.connLastSev = true, sev

	if err != nil {
		d.logEvent(PanelConnections, LogEntry{
			Time: time.Now(), Severity: sev, Panel: PanelConnections,
			Summary: "collection error: " + err.Error(),
		})
		return
	}

	pct := stats.UtilizationPercent()
	d.logEvent(PanelConnections, LogEntry{
		Time: time.Now(), Severity: sev, Panel: PanelConnections,
		Summary: fmt.Sprintf("connections %d/%d (%.0f%%)", stats.Total, stats.MaxConnections, pct),
		Fields: []LogField{
			{"Active", fmt.Sprintf("%d", stats.Active)},
			{"Idle", fmt.Sprintf("%d", stats.Idle)},
			{"Idle in transaction", fmt.Sprintf("%d", stats.IdleInTransaction)},
			{"Max connections", fmt.Sprintf("%d", stats.MaxConnections)},
		},
	})
}

func (d *DBState) RecordCache(stats collector.CacheStats, err error) {
	d.CacheStats, d.CacheErr, d.CacheLoaded = stats, err, true
	if err != nil {
		d.Bootstrapped = false
	} else {
		d.CacheHistory = appendHistory(d.CacheHistory, stats.HitRatio())
	}

	sev := d.PanelSeverity(PanelCache)
	if d.cacheLogged && sev == d.cacheLastSev {
		return
	}
	d.cacheLogged, d.cacheLastSev = true, sev

	if err != nil {
		d.logEvent(PanelCache, LogEntry{
			Time: time.Now(), Severity: sev, Panel: PanelCache,
			Summary: "collection error: " + err.Error(),
		})
		return
	}

	d.logEvent(PanelCache, LogEntry{
		Time: time.Now(), Severity: sev, Panel: PanelCache,
		Summary: fmt.Sprintf("cache hit ratio %.1f%%", stats.HitRatio()),
		Fields: []LogField{
			{"Blocks hit", fmt.Sprintf("%d", stats.BlocksHit)},
			{"Blocks read", fmt.Sprintf("%d", stats.BlocksRead)},
		},
	})
}

// RecordQueries logs a query row only the first time its call count
// increases after dbwatch established a baseline — otherwise every
// historical total would replay as if it just happened. Collection errors
// dedupe the same way as the other panels.
func (d *DBState) RecordQueries(stats []collector.QueryStat, err error) {
	d.QueryStats, d.QueryErr, d.QueryLoaded = stats, err, true
	if err != nil {
		d.Bootstrapped = false
	}

	if err != nil {
		sev := d.PanelSeverity(PanelQueries)
		if d.queryErrLogged && sev == d.queryLastSev {
			return
		}
		d.queryErrLogged, d.queryLastSev = true, sev
		d.logEvent(PanelQueries, LogEntry{
			Time: time.Now(), Severity: sev, Panel: PanelQueries,
			Summary: "collection error: " + err.Error(),
		})
		return
	}
	d.queryErrLogged = false

	first := !d.queryBaseline
	d.queryBaseline = true

	for _, q := range stats {
		prev, seen := d.queryLastCall[q.Query]
		d.queryLastCall[q.Query] = q.Calls
		if first || (seen && q.Calls <= prev) {
			continue
		}
		d.logEvent(PanelQueries, LogEntry{
			Time: time.Now(), Severity: SeverityInfo, Panel: PanelQueries,
			Summary: fmt.Sprintf("calls %d · mean %s · total %s · rows %d",
				q.Calls,
				fmtDuration(time.Duration(q.MeanExecMs*float64(time.Millisecond))),
				fmtDuration(time.Duration(q.TotalExecMs*float64(time.Millisecond))),
				q.Rows),
			Query: q.Query,
			Fields: []LogField{
				{"Calls", fmt.Sprintf("%d", q.Calls)},
				{"Mean time", fmtDuration(time.Duration(q.MeanExecMs * float64(time.Millisecond)))},
				{"Total time", fmtDuration(time.Duration(q.TotalExecMs * float64(time.Millisecond)))},
				{"Rows", fmt.Sprintf("%d", q.Rows)},
			},
		})
	}
}

func (d *DBState) RecordLocks(stats []collector.BlockedLock, err error) {
	d.LockStats, d.LockErr, d.LockLoaded = stats, err, true
	if err != nil {
		d.Bootstrapped = false
	}

	if err != nil {
		sev := d.PanelSeverity(PanelLocks)
		if d.locksLogged && d.locksLastState == "error" {
			return
		}
		d.locksLogged, d.locksLastState = true, "error"
		d.logEvent(PanelLocks, LogEntry{
			Time: time.Now(), Severity: sev, Panel: PanelLocks,
			Summary: "collection error: " + err.Error(),
		})
		return
	}

	if len(stats) == 0 {
		if !d.locksLogged || d.locksLastState != "empty" {
			d.logEvent(PanelLocks, LogEntry{
				Time: time.Now(), Severity: SeverityOK, Panel: PanelLocks,
				Summary: "no lock contention detected",
			})
		}
		d.locksLogged, d.locksLastState = true, "empty"
		return
	}
	d.locksLogged, d.locksLastState = true, "nonempty"

	for _, l := range stats {
		d.logEvent(PanelLocks, LogEntry{
			Time: time.Now(), Severity: SeverityCritical, Panel: PanelLocks,
			Summary: fmt.Sprintf("pid %d blocked by pid %d", l.BlockedPID, l.BlockingPID),
			Query:   l.BlockedQuery,
			Fields: []LogField{
				{"Blocked PID", fmt.Sprintf("%d", l.BlockedPID)},
				{"Blocked waiting", fmtDuration(l.BlockedDuration)},
				{"Blocked query", l.BlockedQuery},
				{"Blocking PID", fmt.Sprintf("%d", l.BlockingPID)},
				{"Blocking holding", fmtDuration(l.BlockingDuration)},
				{"Blocking query", l.BlockingQuery},
			},
		})
	}
}

func (d *DBState) RecordTransactions(stats []collector.LongTransaction, err error) {
	d.TxStats, d.TxErr, d.TxLoaded = stats, err, true
	if err != nil {
		d.Bootstrapped = false
	}

	if err != nil {
		sev := d.PanelSeverity(PanelTransactions)
		if d.txLogged && d.txLastState == "error" {
			return
		}
		d.txLogged, d.txLastState = true, "error"
		d.logEvent(PanelTransactions, LogEntry{
			Time: time.Now(), Severity: sev, Panel: PanelTransactions,
			Summary: "collection error: " + err.Error(),
		})
		return
	}

	if len(stats) == 0 {
		if !d.txLogged || d.txLastState != "empty" {
			d.logEvent(PanelTransactions, LogEntry{
				Time: time.Now(), Severity: SeverityOK, Panel: PanelTransactions,
				Summary: "no long-running transactions",
			})
		}
		d.txLogged, d.txLastState = true, "empty"
		return
	}
	d.txLogged, d.txLastState = true, "nonempty"

	for _, t := range stats {
		app := t.Application
		if app == "" {
			app = "(unknown client)"
		}
		d.logEvent(PanelTransactions, LogEntry{
			Time: time.Now(), Severity: SeverityWarning, Panel: PanelTransactions,
			Summary: fmt.Sprintf("pid %d · %s · running %s", t.PID, t.State, fmtDuration(t.Duration)),
			Query:   t.Query,
			Fields: []LogField{
				{"PID", fmt.Sprintf("%d", t.PID)},
				{"State", t.State},
				{"Duration", fmtDuration(t.Duration)},
				{"Client", app},
				{"Database", d.Database},
			},
		})
	}
}

// sizeGrowthLogThresholdBytes and sizeGrowthLogThresholdPct bound when a
// size change is worth its own log entry: below both, autovacuum-scale
// noise on a small/idle database would otherwise log constantly. Above
// either, it's a real, worth-noting change in footprint.
const sizeGrowthLogThresholdBytes = 10 * 1024 * 1024 // 10 MiB
const sizeGrowthLogThresholdPct = 0.05               // 5%

// RecordSize stores the latest snapshot, derives a signed growth rate from
// consecutive snapshots (size can shrink, unlike Activity's monotonic
// counters), and logs a baseline reading plus any subsequent change past
// the noise thresholds above.
func (d *DBState) RecordSize(stats collector.SizeStats, err error) {
	d.SizeStats, d.SizeErr, d.SizeLoaded = stats, err, true
	if err != nil {
		d.Bootstrapped = false
		if !d.sizeErrLogged {
			d.sizeErrLogged = true
			d.logEvent(PanelSize, LogEntry{
				Time: time.Now(), Severity: SeverityError, Panel: PanelSize,
				Summary: "collection error: " + err.Error(),
			})
		}
		return
	}
	d.sizeErrLogged = false

	now := time.Now()
	d.SizeHistory = appendHistory(d.SizeHistory, float64(stats.DatabaseBytes))
	if !d.prevSizeAt.IsZero() {
		if elapsed := now.Sub(d.prevSizeAt).Seconds(); elapsed > 0 {
			d.SizeGrowthPerSec = float64(stats.DatabaseBytes-d.prevSize.DatabaseBytes) / elapsed
			d.SizeGrowthLoaded = true
		}
	}
	d.prevSize, d.prevSizeAt = stats, now

	if !d.sizeBaselineLogged {
		d.sizeBaselineLogged = true
		d.sizeLastLoggedBytes = stats.DatabaseBytes
		d.logEvent(PanelSize, LogEntry{
			Time: now, Severity: SeverityInfo, Panel: PanelSize,
			Summary: fmt.Sprintf("database size %s (tables %s, indexes %s)",
				humanBytes(stats.DatabaseBytes), humanBytes(stats.TablesBytes), humanBytes(stats.IndexesBytes)),
		})
		return
	}

	delta := stats.DatabaseBytes - d.sizeLastLoggedBytes
	absDelta := delta
	if absDelta < 0 {
		absDelta = -absDelta
	}
	pctDelta := 0.0
	if d.sizeLastLoggedBytes > 0 {
		pctDelta = float64(absDelta) / float64(d.sizeLastLoggedBytes)
	}
	if absDelta >= sizeGrowthLogThresholdBytes || pctDelta >= sizeGrowthLogThresholdPct {
		dir := "grew"
		if delta < 0 {
			dir = "shrank"
		}
		d.logEvent(PanelSize, LogEntry{
			Time: now, Severity: SeverityInfo, Panel: PanelSize,
			Summary: fmt.Sprintf("database size %s by %s, now %s", dir, humanBytes(absDelta), humanBytes(stats.DatabaseBytes)),
		})
		d.sizeLastLoggedBytes = stats.DatabaseBytes
	}
}

// RecordCapabilities stores the latest snapshot and logs a baseline reading
// plus any subsequent change to what the server exposes or permits (an
// extension installed at runtime, a permission grant, a config reload
// flipping a track_* setting) -- all real, discrete events rather than a
// repeat of the same reading every tick.
func (d *DBState) RecordCapabilities(stats collector.CapabilityStats, err error) {
	d.CapStats, d.CapErr, d.CapLoaded = stats, err, true
	if err != nil {
		d.Bootstrapped = false
		if !d.capErrLogged {
			d.capErrLogged = true
			d.logEvent(PanelCapabilities, LogEntry{
				Time: time.Now(), Severity: SeverityError, Panel: PanelCapabilities,
				Summary: "collection error: " + err.Error(),
			})
		}
		return
	}
	d.capErrLogged = false

	visibility := "own queries only"
	if stats.IsSuperuser {
		visibility = "superuser"
	} else if stats.HasReadAllStats {
		visibility = "pg_read_all_stats"
	}
	key := fmt.Sprintf("%s|%t|%t|%t|%t|%d", visibility, stats.TrackActivities, stats.TrackCounts, stats.TrackIOTiming,
		stats.PgStatStatementsAvailable, len(stats.Extensions))
	if d.capLastLogged == key {
		return
	}
	d.capLastLogged = key

	sev := SeverityInfo
	if !stats.IsSuperuser && !stats.HasReadAllStats {
		sev = SeverityWarning
	}
	d.logEvent(PanelCapabilities, LogEntry{
		Time: time.Now(), Severity: sev, Panel: PanelCapabilities,
		Summary: fmt.Sprintf("visibility: %s · %d extension(s) installed", visibility, len(stats.Extensions)),
		Fields: []LogField{
			{"Visibility", visibility},
			{"track_activities", fmt.Sprintf("%t", stats.TrackActivities)},
			{"track_counts", fmt.Sprintf("%t", stats.TrackCounts)},
			{"track_io_timing", fmt.Sprintf("%t", stats.TrackIOTiming)},
			{"pg_stat_statements available", fmt.Sprintf("%t", stats.PgStatStatementsAvailable)},
		},
	})
}
