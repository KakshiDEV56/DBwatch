// Package tui is the Bubble Tea terminal UI for `dbwatch start`.
package tui

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"dbwatch/internal/collector"
	"dbwatch/internal/config"
	"dbwatch/internal/store"
)

type focusArea int

const (
	focusDBs focusArea = iota
	focusPanels
	focusLogs
	focusSearch
	focusAddDB
)

const minWidth, minHeight = 50, 16

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
type activityResult struct {
	db    int
	stats collector.ActivityStats
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

	helpOpen   bool
	helpOffset int

	searchQuery string
	searchInput string
	searchWasIn focusArea

	// addDBInput is the DSN being typed into the welcome screen / "add
	// database" overlay. addDBWasIn is where focus returns to on Esc
	// (irrelevant on the welcome screen itself, since there's nowhere
	// else to go with zero databases).
	addDBInput string
	addDBErr   string
	addDBWasIn focusArea

	// confirmRemove is the index into dbs pending a remove confirmation,
	// or -1 when no confirmation is open.
	confirmRemove int

	logCursor     int
	logAutoFollow bool

	width, height int
	lastUpdate    time.Time

	flash string
}

func NewModel(ctx context.Context, dbs []*DBState, interval time.Duration) Model {
	m := Model{
		ctx:           ctx,
		dbs:           dbs,
		interval:      interval,
		focus:         focusPanels,
		logAutoFollow: true,
		confirmRemove: -1,
	}
	if len(dbs) == 0 {
		// Nothing configured yet -- open straight onto the welcome
		// screen's input instead of an empty dashboard.
		m.focus = focusAddDB
	}
	return m
}

// databaseTargets converts the live DBState list back to the config
// shape the store persists.
func (m Model) databaseTargets() []config.Database {
	targets := make([]config.Database, len(m.dbs))
	for i, db := range m.dbs {
		targets[i] = config.Database{Name: db.Name, Region: db.Region, DSN: db.DSN}
	}
	return targets
}

// saveDatabases persists the current database list. A failure (disk
// full, permissions) is returned rather than panicking -- it shouldn't
// take down a monitoring tool that's otherwise working fine; callers
// surface it as a flash message.
func (m *Model) saveDatabases() error {
	return store.Save(m.databaseTargets())
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
			func() tea.Msg {
				stats, err := db.Activity.Collect(m.ctx)
				return activityResult{i, stats, err}
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
	case activityResult:
		m.dbs[msg.db].RecordActivity(msg.stats, msg.err)
		return m, nil
	case bootstrapResult:
		if msg.err == nil {
			m.dbs[msg.db].Bootstrapped = true
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.MouseMsg:
		return m.handleMouse(msg)
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
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

	go func() {
		<-ctx.Done()
		p.Send(quitMsg{})
	}()

	_, err := p.Run()
	return err
}
