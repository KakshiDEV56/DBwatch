package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.focus == focusSearch {
		return m.handleSearchKey(msg)
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
