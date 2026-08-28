package tui

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/KakshiDEV56/DBwatch/internal/config"
)

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.confirmRemove >= 0 {
		return m.handleConfirmRemoveKey(msg)
	}
	if m.focus == focusSearch {
		return m.handleSearchKey(msg)
	}
	if m.focus == focusAddDB {
		return m.handleAddDBKey(msg)
	}
	if m.helpOpen {
		lines := len(guideContent())
		switch msg.String() {
		case "?", "esc", "q":
			m.helpOpen = false
		case "j", "down":
			if m.helpOffset < lines-1 {
				m.helpOffset++
			}
		case "k", "up":
			if m.helpOffset > 0 {
				m.helpOffset--
			}
		case "G":
			m.helpOffset = lines - 1
		case "g":
			m.helpOffset = 0
		}
		return m, nil
	}
	if m.detail != nil {
		switch msg.String() {
		case "esc", "q":
			m.detail = nil
		case "c":
			cmd := m.setFlash("copied to clipboard")
			Copy(m.detail.CopyText())
			return m, cmd
		}
		return m, nil
	}

	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "q":
		return m, tea.Quit
	case "?":
		m.helpOpen = true
		m.helpOffset = 0
		return m, nil
	case "m":
		m.readMode = !m.readMode
		return m, nil
	case "r":
		m.lastUpdate = time.Now()
		return m, tea.Batch(m.fetchAll()...)
	case "a":
		m.addDBWasIn = m.focus
		m.addDBInput = ""
		m.addDBErr = ""
		m.focus = focusAddDB
		return m, nil
	case "/":
		m.prevFocus = m.focus
		m.searchWasIn = m.focus
		m.searchInput = m.searchQuery
		m.focus = focusSearch
		return m, nil
	case "tab", "l":
		m.focus = nextFocus(m.focus)
		return m, nil
	case "shift+tab", "h":
		m.focus = prevFocus(m.focus)
		return m, nil
	case "n":
		return m.jumpSearch(1), nil
	case "N":
		return m.jumpSearch(-1), nil
	}

	switch m.focus {
	case focusDBs:
		return m.handleDBsKey(msg)
	case focusPanels:
		return m.handlePanelsKey(msg)
	case focusLogs:
		return m.handleLogsKey(msg)
	}
	return m, nil
}

func nextFocus(f focusArea) focusArea {
	switch f {
	case focusPanels:
		return focusDBs
	case focusDBs:
		return focusLogs
	default:
		return focusPanels
	}
}

func prevFocus(f focusArea) focusArea {
	switch f {
	case focusPanels:
		return focusLogs
	case focusLogs:
		return focusDBs
	default:
		return focusPanels
	}
}

func (m Model) handleDBsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		m.moveDB(1)
	case "k", "up":
		m.moveDB(-1)
	case "enter":
		m.focus = focusPanels
	case "d":
		if len(m.dbs) > 0 {
			m.confirmRemove = m.selectedDB
		}
	}
	return m, nil
}

func (m Model) handlePanelsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		m.movePanel(1)
	case "k", "up":
		m.movePanel(-1)
	case "enter":
		m.focus = focusLogs
	}
	return m, nil
}

func (m Model) handleLogsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	entries := m.filteredEntries()
	switch msg.String() {
	case "j", "down":
		m.moveLog(1)
	case "k", "up":
		m.moveLog(-1)
	case "G":
		m.logCursor = len(entries) - 1
		m.logAutoFollow = true
	case "enter":
		if m.logCursor >= 0 && m.logCursor < len(entries) {
			e := entries[m.logCursor]
			m.detail = &e
		}
	case "c":
		if m.logCursor >= 0 && m.logCursor < len(entries) {
			cmd := m.setFlash("copied to clipboard")
			Copy(entries[m.logCursor].CopyText())
			return m, cmd
		}
	}
	return m, nil
}

func (m *Model) resetLogPosition() {
	entries := m.filteredEntries()
	m.logCursor = len(entries) - 1
	m.logAutoFollow = true
}

func (m Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.focus = m.searchWasIn
		return m, nil
	case "enter":
		m.searchQuery = m.searchInput
		m.focus = m.searchWasIn
		m.resetLogPosition()
		return m, nil
	case "backspace":
		if len(m.searchInput) > 0 {
			r := []rune(m.searchInput)
			m.searchInput = string(r[:len(r)-1])
		}
		return m, nil
	default:
		if msg.Type == tea.KeyRunes {
			m.searchInput += string(msg.Runes)
		} else if msg.Type == tea.KeySpace {
			m.searchInput += " "
		}
		return m, nil
	}
}

func (m Model) jumpSearch(dir int) Model {
	if m.searchQuery == "" {
		return m
	}
	entries := m.filteredEntries()
	if len(entries) == 0 {
		return m
	}
	next := m.logCursor + dir
	if next < 0 {
		next = 0
	}
	if next > len(entries)-1 {
		next = len(entries) - 1
	}
	m.logCursor = next
	m.logAutoFollow = false
	return m
}

// handleAddDBKey drives the welcome screen / "add database" overlay --
// the same UI either way, the only difference is whether there's a
// dashboard behind it to cancel back to.
func (m Model) handleAddDBKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		if len(m.dbs) == 0 {
			return m, nil // nothing to cancel back to yet
		}
		m.focus = m.addDBWasIn
		m.addDBInput = ""
		m.addDBErr = ""
		return m, nil
	case "enter":
		return m.submitAddDB()
	case "backspace":
		if len(m.addDBInput) > 0 {
			r := []rune(m.addDBInput)
			m.addDBInput = string(r[:len(r)-1])
		}
		return m, nil
	default:
		if msg.Type == tea.KeyRunes {
			m.addDBInput += string(msg.Runes)
		} else if msg.Type == tea.KeySpace {
			m.addDBInput += " "
		}
		return m, nil
	}
}

func (m Model) submitAddDB() (tea.Model, tea.Cmd) {
	dsn := strings.TrimSpace(m.addDBInput)
	if dsn == "" {
		m.addDBErr = "enter a postgres:// connection string"
		return m, nil
	}

	name := fmt.Sprintf("db%d", len(m.dbs)+1)
	if u, err := url.Parse(dsn); err == nil {
		if dbName := strings.TrimPrefix(u.Path, "/"); dbName != "" {
			name = dbName
		}
	}
	target := config.Database{Name: name, Region: "local", DSN: dsn}

	// pgxpool.New only parses the DSN (no I/O), so this is safe to call
	// synchronously without blocking the UI. A malformed DSN is caught
	// immediately here; an unreachable-but-valid DSN is not an error --
	// it's added and shown as "connecting..." like any other database,
	// self-healing via the same Bootstrap retry every other database
	// uses (see DBState.Bootstrap).
	db := ConnectDatabase(m.ctx, target)
	if db.ConnectErr != nil {
		m.addDBErr = db.ConnectErr.Error()
		return m, nil
	}

	m.dbs = append(m.dbs, db)
	m.selectedDB = len(m.dbs) - 1
	m.addDBInput = ""
	m.addDBErr = ""
	m.focus = focusPanels

	flashMsg := "added " + target.Name
	if err := m.saveDatabases(); err != nil {
		flashMsg = "added, but could not save: " + err.Error()
	}
	cmds := m.fetchAll() // start collecting for it immediately, don't wait for the next tick
	cmds = append(cmds, m.setFlash(flashMsg))
	return m, tea.Batch(cmds...)
}

func (m Model) handleConfirmRemoveKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		idx := m.confirmRemove
		m.confirmRemove = -1
		if idx < 0 || idx >= len(m.dbs) {
			return m, nil
		}
		removed := m.dbs[idx]
		if removed.Pool != nil {
			removed.Pool.Close()
		}
		m.dbs = append(m.dbs[:idx], m.dbs[idx+1:]...)
		if m.selectedDB >= len(m.dbs) {
			m.selectedDB = len(m.dbs) - 1
		}
		if m.selectedDB < 0 {
			m.selectedDB = 0
		}
		if len(m.dbs) == 0 {
			m.focus = focusAddDB
		}

		flashMsg := "removed " + removed.Name
		if err := m.saveDatabases(); err != nil {
			flashMsg = "removed, but could not save: " + err.Error()
		}
		return m, m.setFlash(flashMsg)
	case "n", "esc":
		m.confirmRemove = -1
	}
	return m, nil
}
