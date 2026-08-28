// Package tui is the Bubble Tea terminal UI for `dbwatch start`.
package tui

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"dbwatch/internal/collector"
)

type focusArea int

const (
	focusDBs focusArea = iota
	focusPanels
	focusLogs
	focusSearch
)

const minWidth, minHeight = 70, 20

type connResult struct {
	db    int
	stats collector.ConnectionStats
	err   error
}
type cacheResult struct {
	db    int
	stats collector.CacheStats
	err   error
}
type queryResult struct {
	db    int
	stats []collector.QueryStat
	err   error
}
type lockResult struct {
	db    int
	stats []collector.BlockedLock
	err   error
}
type txResult struct {
	db    int
	stats []collector.LongTransaction
	err   error
}

type bootstrapResult struct {
	db  int
	err error
}

type tickMsg time.Time
type quitMsg struct{}
type flashClearMsg struct{}

type Model struct {
	ctx      context.Context
	dbs      []*DBState
	interval time.Duration

	selectedDB    int
	selectedPanel int
	focus         focusArea
	prevFocus     focusArea

	readMode bool

	detail *LogEntry

	helpOpen bool

	searchQuery string
	searchInput string
	searchWasIn focusArea

	logOffset     int
	logCursor     int
	logAutoFollow bool

	width, height int
	lastUpdate    time.Time

	flash string
}

func NewModel(ctx context.Context, dbs []*DBState, interval time.Duration) Model {
	return Model{
		ctx:           ctx,
		dbs:           dbs,
		interval:      interval,
		focus:         focusPanels,
		logAutoFollow: true,
	}
}

func (m Model) currentDB() *DBState {
	if len(m.dbs) == 0 {
		return nil
	}
	return m.dbs[m.selectedDB]
}

func (m Model) currentPanel() PanelKind {
	return allPanels[m.selectedPanel]
}

// filteredEntries returns the currently viewed panel's log entries,
// filtered by the committed search query when one is set.
func (m Model) filteredEntries() []LogEntry {
	db := m.currentDB()
	if db == nil {
		return nil
	}
	entries := db.Logs[m.currentPanel()].Entries()
	if m.searchQuery == "" {
		return entries
	}
	q := strings.ToLower(m.searchQuery)
	filtered := make([]LogEntry, 0, len(entries))
	for _, e := range entries {
		if e.MatchesSearch(q) {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

func (m Model) fetchAll() []tea.Cmd {
	var cmds []tea.Cmd
	for i, db := range m.dbs {
		if db.Pool == nil {
			continue
		}
		cmds = append(cmds,
			func() tea.Msg {
				stats, err := db.Connections.Collect(m.ctx)
				return connResult{i, stats, err}
			},
			func() tea.Msg {
				stats, err := db.Cache.Collect(m.ctx)
				return cacheResult{i, stats, err}
			},
			func() tea.Msg {
				stats, err := db.Locks.Collect(m.ctx)
				return lockResult{i, stats, err}
			},
			func() tea.Msg {
				stats, err := db.Transactions.Collect(m.ctx)
				return txResult{i, stats, err}
			},
		)
		// Gated on Bootstrapped, not just QueryExtWarn=="" -- both start
		// empty/false together, so without this the very first tick fires
		// this fetch concurrently with the bootstrap that creates the
		// extension, racing a spurious "relation does not exist" error.
		if db.Bootstrapped && db.QueryExtWarn == "" {
			cmds = append(cmds, func() tea.Msg {
				stats, err := db.Queries.Collect(m.ctx)
				return queryResult{i, stats, err}
			})
		}
		if !db.Bootstrapped {
			cmds = append(cmds, func() tea.Msg {
				return bootstrapResult{i, db.Bootstrap(m.ctx)}
			})
		}
	}
	return cmds
}

func (m Model) tick() tea.Cmd {
	return tea.Tick(m.interval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m Model) Init() tea.Cmd {
	cmds := append(m.fetchAll(), m.tick())
	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case quitMsg:
		return m, tea.Quit

	case flashClearMsg:
		m.flash = ""
		return m, nil

	case tickMsg:
		var cmds []tea.Cmd
		if !m.readMode {
			m.lastUpdate = time.Now()
			cmds = m.fetchAll()
		}
		cmds = append(cmds, m.tick())
		return m, tea.Batch(cmds...)

	case connResult:
		m.dbs[msg.db].RecordConnections(msg.stats, msg.err)
		m.syncLogCursor()
		return m, nil
	case cacheResult:
		m.dbs[msg.db].RecordCache(msg.stats, msg.err)
		m.syncLogCursor()
		return m, nil
	case queryResult:
		m.dbs[msg.db].RecordQueries(msg.stats, msg.err)
		m.syncLogCursor()
		return m, nil
	case lockResult:
		m.dbs[msg.db].RecordLocks(msg.stats, msg.err)
		m.syncLogCursor()
		return m, nil
	case txResult:
		m.dbs[msg.db].RecordTransactions(msg.stats, msg.err)
		m.syncLogCursor()
		return m, nil
	case bootstrapResult:
		if msg.err == nil {
			m.dbs[msg.db].Bootstrapped = true
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// syncLogCursor keeps the cursor pinned to the newest entry when
// logAutoFollow is on, and otherwise leaves it exactly where the user put
// it — new data must never yank the viewport out from under someone
// reading.
func (m *Model) syncLogCursor() {
	entries := m.filteredEntries()
	if m.logAutoFollow {
		m.logCursor = len(entries) - 1
	} else if m.logCursor >= len(entries) {
		m.logCursor = len(entries) - 1
	}
	if m.logCursor < 0 {
		m.logCursor = 0
	}
}

func (m *Model) setFlash(s string) tea.Cmd {
	m.flash = s
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return flashClearMsg{} })
}

// Run starts the Bubble Tea program and blocks until the user quits or ctx
// is cancelled.
func Run(ctx context.Context, dbs []*DBState, interval time.Duration) error {
	m := NewModel(ctx, dbs, interval)
	p := tea.NewProgram(m, tea.WithAltScreen())

	go func() {
		<-ctx.Done()
		p.Send(quitMsg{})
	}()

	_, err := p.Run()
	return err
}
