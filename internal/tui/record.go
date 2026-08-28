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

// RecordActivity derives per-second rates by diffing this snapshot
// against the previous one -- pg_stat_database's counters are cumulative
// since the last stats reset, not point-in-time, so a single snapshot
// alone can't say anything about current load.
func (d *DBState) RecordActivity(stats collector.ActivityStats, err error) {
	d.ActivityErr = err
	if err != nil {
		return
	}

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
		d.Logs[PanelConnections].Append(LogEntry{
			Time: time.Now(), Severity: sev, Panel: PanelConnections,
			Summary: "collection error: " + err.Error(),
		})
		return
	}

	pct := stats.UtilizationPercent()
	d.Logs[PanelConnections].Append(LogEntry{
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
		d.Logs[PanelCache].Append(LogEntry{
			Time: time.Now(), Severity: sev, Panel: PanelCache,
			Summary: "collection error: " + err.Error(),
		})
		return
	}

	d.Logs[PanelCache].Append(LogEntry{
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
		d.Logs[PanelQueries].Append(LogEntry{
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
		d.Logs[PanelQueries].Append(LogEntry{
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
		d.Logs[PanelLocks].Append(LogEntry{
			Time: time.Now(), Severity: sev, Panel: PanelLocks,
			Summary: "collection error: " + err.Error(),
		})
		return
	}

	if len(stats) == 0 {
		if !d.locksLogged || d.locksLastState != "empty" {
			d.Logs[PanelLocks].Append(LogEntry{
				Time: time.Now(), Severity: SeverityOK, Panel: PanelLocks,
				Summary: "no lock contention detected",
			})
		}
		d.locksLogged, d.locksLastState = true, "empty"
		return
	}
	d.locksLogged, d.locksLastState = true, "nonempty"

	for _, l := range stats {
		d.Logs[PanelLocks].Append(LogEntry{
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
		d.Logs[PanelTransactions].Append(LogEntry{
			Time: time.Now(), Severity: sev, Panel: PanelTransactions,
			Summary: "collection error: " + err.Error(),
		})
		return
	}

	if len(stats) == 0 {
		if !d.txLogged || d.txLastState != "empty" {
			d.Logs[PanelTransactions].Append(LogEntry{
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
		d.Logs[PanelTransactions].Append(LogEntry{
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
