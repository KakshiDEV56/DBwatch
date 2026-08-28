package tui

import tea "github.com/charmbracelet/bubbletea"

// handleMouse implements two things: wheel-scroll (moves the selection in
// whatever section the pointer is currently over, "hover to scroll" —
// like any normal application, not just the section that happens to have
// keyboard focus) and click-to-select (clicking a card, a database row,
// or a log entry selects it directly, the same as navigating there with
// j/k). hitTest in layout.go is the single source of truth for which
// on-screen row a coordinate belongs to, shared with the renderer.
func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.helpOpen {
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			if m.helpOffset > 0 {
				m.helpOffset--
			}
		case tea.MouseButtonWheelDown:
			if m.helpOffset < len(guideContent())-1 {
				m.helpOffset++
			}
		}
		return m, nil
	}
	if m.detail != nil || m.focus == focusSearch || m.focus == focusAddDB || m.confirmRemove >= 0 {
		return m, nil
	}
	if msg.Action != tea.MouseActionPress {
		return m, nil
	}

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		return m.scrollRegionAt(msg.X, msg.Y, -1), nil
	case tea.MouseButtonWheelDown:
		return m.scrollRegionAt(msg.X, msg.Y, 1), nil
	case tea.MouseButtonLeft:
		return m.clickAt(msg.X, msg.Y), nil
	}
	return m, nil
}

func (m Model) scrollRegionAt(x, y, delta int) Model {
	reg, _ := m.hitTest(x, y)
	switch reg {
	case regionSidebar:
		m.focus = focusPanels
		m.movePanel(delta)
	case regionDBTable:
		m.focus = focusDBs
		m.moveDB(delta)
	case regionLogs:
		m.focus = focusLogs
		m.moveLog(delta)
	}
	return m
}

func (m Model) clickAt(x, y int) Model {
	reg, idx := m.hitTest(x, y)
	switch reg {
	case regionSidebar:
		m.focus = focusPanels
		if idx >= 0 {
			m.selectedPanel = idx
			m.resetLogPosition()
		}
	case regionDBTable:
		m.focus = focusDBs
		if idx >= 0 {
			m.selectedDB = idx
			m.resetLogPosition()
		}
	case regionLogs:
		m.focus = focusLogs
		if idx >= 0 {
			m.logCursor = idx
			entries := m.filteredEntries()
			m.logAutoFollow = idx >= len(entries)-1
		}
	}
	return m
}

func (m *Model) movePanel(delta int) {
	next := m.selectedPanel + delta
	if next < 0 || next >= len(allPanels) {
		return
	}
	m.selectedPanel = next
	m.resetLogPosition()
}

func (m *Model) moveDB(delta int) {
	next := m.selectedDB + delta
	if next < 0 || next >= len(m.dbs) {
		return
	}
	m.selectedDB = next
	m.resetLogPosition()
}

func (m *Model) moveLog(delta int) {
	entries := m.filteredEntries()
	next := m.logCursor + delta
	if next < 0 {
		next = 0
	}
	if next > len(entries)-1 {
		next = len(entries) - 1
	}
	m.logCursor = next
	m.logAutoFollow = next >= len(entries)-1
}
