package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

const sidebarWidth = 30

func (m Model) View() string {
	if m.width < minWidth || m.height < minHeight {
		msg := fmt.Sprintf("terminal too small\nresize to at least %dx%d (currently %dx%d)", minWidth, minHeight, m.width, m.height)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, warnStyle.Render(msg))
	}

	if m.helpOpen {
		return m.helpView()
	}
	if m.detail != nil {
		return m.detailView()
	}

	var b strings.Builder
	b.WriteString(m.headerView())
	b.WriteString("\n")

	bodyHeight := m.height - 6 // header(2) + identity(1) + gap(1) + status bar(1) + margin(1)
	if bodyHeight < 6 {
		bodyHeight = 6
	}

	left := m.sidebarView(bodyHeight)
	right := m.rightView(m.width-sidebarWidth-4, bodyHeight)
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, left, right))
	b.WriteString("\n")
	b.WriteString(m.statusBarView())

	return b.String()
}

func (m Model) headerView() string {
	db := m.currentDB()

	left := titleStyle.Render("DBWATCH")
	right := dimStyle.Render("PostgreSQL Monitoring")
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	line1 := left + strings.Repeat(" ", gap) + right

	mode := liveStyle.Render("● LIVE")
	if m.readMode {
		mode = readModeStyle.Render("■ READ MODE")
	}
	updated := "—"
	if !m.lastUpdate.IsZero() {
		updated = m.lastUpdate.Format("15:04:05")
	}
	statusLeft := dimStyle.Render(fmt.Sprintf("monitoring %d database(s) · last update %s", len(m.dbs), updated))
	gap2 := m.width - lipgloss.Width(statusLeft) - lipgloss.Width(mode)
	if gap2 < 1 {
		gap2 = 1
	}
	line2 := statusLeft + strings.Repeat(" ", gap2) + mode

	identity := dimStyle.Render("no database configured")
	if db != nil {
		sev := db.OverallSeverity()
		version := db.Version
		if version == "" {
			version = "connecting…"
		}
		uptimeStr := ""
		if u, ok := db.Uptime(); ok {
			uptimeStr = "  uptime " + fmtUptime(u)
		}
		identity = fmt.Sprintf("PostgreSQL %s  %s  %s:%d/%s  %s%s",
			version, labelStyle.Render(db.Name), db.Host, db.Port, db.Database,
			sev.Render(strings.ToUpper(sev.Label())), uptimeStr)
		if db.ConnectErr != nil {
			identity = fmt.Sprintf("%s  %s", labelStyle.Render(db.Name), errStyle.Render("⚠ "+db.ConnectErr.Error()))
		}
	}

	return line1 + "\n" + line2 + "\n" + identity
}

func fmtUptime(d time.Duration) string {
	d = d.Round(time.Minute)
	days := d / (24 * time.Hour)
	d -= days * 24 * time.Hour
	hours := d / time.Hour
	d -= hours * time.Hour
	mins := d / time.Minute
	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}

func (m Model) sidebarView(height int) string {
	var cards []string
	for i, p := range allPanels {
		focused := m.focus == focusPanels && i == m.selectedPanel
		cards = append(cards, m.panelCard(p, focused))
	}
	content := lipgloss.JoinVertical(lipgloss.Left, cards...)
	return lipgloss.NewStyle().Width(sidebarWidth).Height(height).Render(content)
}

func (m Model) panelCard(p PanelKind, focused bool) string {
	db := m.currentDB()
	style := boxStyleFor(focused).Width(sidebarWidth - 4)

	title := p.Title()
	if focused {
		title = "> " + title
	}
	var body string
	if db == nil {
		body = dimStyle.Render("no database")
	} else {
		sev := db.PanelSeverity(p)
		body = sev.Render(db.PanelSummary(p))
	}
	return style.Render(cardTitleStyle.Render(title) + "\n" + body)
}

func (m Model) rightView(width, height int) string {
	dbTableHeight := 4 + len(m.dbs)
	if dbTableHeight > 8 {
		dbTableHeight = 8
	}
	logsHeight := height - dbTableHeight - 1
	if logsHeight < 5 {
		logsHeight = 5
	}

	top := boxStyleFor(m.focus == focusDBs).Width(width - 2).Height(dbTableHeight).Render(m.dbTableView(width - 4))
	bottom := boxStyleFor(m.focus == focusLogs).Width(width - 2).Height(logsHeight).Render(m.logsView(width-4, logsHeight))

	return lipgloss.JoinVertical(lipgloss.Left, top, bottom)
}

func (m Model) dbTableView(width int) string {
	var b strings.Builder
	b.WriteString(cardTitleStyle.Render("DATABASES — search across regions / names / status") + "\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", width)) + "\n")
	header := fmt.Sprintf("%-3s %-20s %-12s %s", "", "NAME", "REGION", "STATUS")
	b.WriteString(dimStyle.Render(header) + "\n")

	for i, db := range m.dbs {
		cursor := "  "
		if m.focus == focusDBs && i == m.selectedDB {
			cursor = "> "
		}
		marker := "  "
		if i == m.selectedDB {
			marker = "● "
		}
		sev := db.OverallSeverity()
		row := fmt.Sprintf("%s%-20s %-12s %s %s", marker, truncate(db.Name, 20), truncate(db.Region, 12), sev.Symbol(), sev.Label())
		style := lipgloss.NewStyle()
		if m.focus == focusDBs && i == m.selectedDB {
			style = selectedRowStyle
		}
		b.WriteString(cursor + style.Render(row) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m Model) logsView(width, height int) string {
	entries := m.filteredEntries()

	var header strings.Builder
	header.WriteString(cardTitleStyle.Render("LOGS / EVENTS"))
	header.WriteString("  ")
	header.WriteString(dimStyle.Render(fmt.Sprintf("[%s]", m.currentPanel().Title())))
	hint := "  " + keyStyle.Render("c") + dimStyle.Render(" copy   ") + keyStyle.Render("/") + dimStyle.Render(" search   ") + keyStyle.Render("enter") + dimStyle.Render(" inspect")
	header.WriteString(hint)
	if m.searchQuery != "" {
		header.WriteString("  " + dimStyle.Render(fmt.Sprintf("(filter: %q, %d match)", m.searchQuery, len(entries))))
	}

	rows := height - 2
	if rows < 1 {
		rows = 1
	}

	if len(entries) == 0 {
		return header.String() + "\n" + dimStyle.Render(strings.Repeat("─", width)) + "\n" + dimStyle.Render("no events yet")
	}

	offset := m.logOffset
	if m.logCursor < offset {
		offset = m.logCursor
	}
	if m.logCursor >= offset+rows {
		offset = m.logCursor - rows + 1
	}
	if offset < 0 {
		offset = 0
	}
	end := offset + rows
	if end > len(entries) {
		end = len(entries)
	}

	var lines []string
	for i := offset; i < end; i++ {
		e := entries[i]
		cursor := "  "
		lineStyle := lipgloss.NewStyle()
		if m.focus == focusLogs && i == m.logCursor {
			cursor = "> "
			lineStyle = selectedRowStyle
		}
		ts := dimStyle.Render(e.Time.Format("15:04:05"))
		text := fmt.Sprintf("%s %s %s", ts, e.Severity.Symbol(), e.Summary)
		if e.Query != "" {
			budget := width - lipgloss.Width(text) - 5
			if budget > 10 {
				text += "  " + dimStyle.Render(truncate(e.Query, budget))
			}
		}
		lines = append(lines, cursor+lineStyle.Render(truncate(text, width-2)))
	}

	return header.String() + "\n" + dimStyle.Render(strings.Repeat("─", width)) + "\n" + strings.Join(lines, "\n")
}

func (m Model) detailView() string {
	e := m.detail
	var b strings.Builder
	b.WriteString(helpTitleStyle.Render("LOG DETAIL") + "\n\n")
	b.WriteString(e.Severity.Render(e.Summary) + "\n\n")
	for _, f := range e.Fields {
		b.WriteString(fmt.Sprintf("%-20s %s\n", labelStyle.Render(f.Key), f.Value))
	}
	if e.Query != "" {
		b.WriteString("\n" + labelStyle.Render("QUERY") + "\n")
		b.WriteString(dimStyle.Render(strings.Repeat("─", 50)) + "\n")
		b.WriteString(e.Query + "\n")
	}
	b.WriteString("\n" + keyStyle.Render("c") + " copy query   " + keyStyle.Render("esc") + " back")

	box := boxFocusedStyle.Width(min(m.width-10, 90)).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m Model) helpView() string {
	var b strings.Builder
	b.WriteString(helpTitleStyle.Render("DBWATCH — KEYS") + "\n\n")

	rows := [][2]string{
		{"h / l", "move between panels · databases · logs"},
		{"j / k", "navigate within the focused section"},
		{"enter", "inspect (drill into logs / confirm database)"},
		{"esc", "back / cancel"},
		{"c", "copy selected log or detail to clipboard"},
		{"/", "search"},
		{"n / N", "next / previous search match"},
		{"m", "toggle read mode (pause updates) / live mode"},
		{"r", "refresh now"},
		{"?", "toggle this help"},
		{"q", "quit"},
	}
	for _, r := range rows {
		b.WriteString(fmt.Sprintf("  %-10s %s\n", keyStyle.Render(r[0]), r[1]))
	}
	b.WriteString("\n" + dimStyle.Render("press ? or esc to close"))

	box := boxFocusedStyle.Width(min(m.width-10, 70)).Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m Model) statusBarView() string {
	var text string
	switch {
	case m.flash != "":
		text = flashStyle.Render("✓ " + m.flash)
	case m.focus == focusSearch:
		text = fmt.Sprintf("SEARCH ●   %s%s   %s cancel   %s confirm", m.searchInput, "▏", keyStyle.Render("esc"), keyStyle.Render("enter"))
	case m.readMode && m.focus == focusLogs:
		text = fmt.Sprintf("READ MODE ■ · LOGS ●   %s select   %s inspect   %s copy   %s search   %s live   %s back   %s quit",
			keyStyle.Render("j/k"), keyStyle.Render("enter"), keyStyle.Render("c"), keyStyle.Render("/"), keyStyle.Render("m"), keyStyle.Render("h"), keyStyle.Render("q"))
	case m.readMode:
		text = fmt.Sprintf("READ MODE ■   %s scroll   %s panel   %s copy   %s live   %s quit",
			keyStyle.Render("j/k"), keyStyle.Render("h/l"), keyStyle.Render("c"), keyStyle.Render("m"), keyStyle.Render("q"))
	case m.focus == focusLogs:
		text = fmt.Sprintf("%s ●   %s select   %s inspect   %s copy   %s search   %s read/live   %s back   %s quit",
			"LOGS", keyStyle.Render("j/k"), keyStyle.Render("enter"), keyStyle.Render("c"), keyStyle.Render("/"), keyStyle.Render("m"), keyStyle.Render("h"), keyStyle.Render("q"))
	default:
		text = fmt.Sprintf("LIVE ●   %s panel   %s navigate   %s inspect   %s search   %s refresh   %s read   %s help   %s quit",
			keyStyle.Render("h/l"), keyStyle.Render("j/k"), keyStyle.Render("enter"), keyStyle.Render("/"), keyStyle.Render("r"), keyStyle.Render("m"), keyStyle.Render("?"), keyStyle.Render("q"))
	}
	return statusBarStyle.Width(m.width).Render(text)
}
