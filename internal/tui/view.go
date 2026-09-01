package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.width < minWidth || m.height < minHeight {
		msg := fmt.Sprintf("terminal too small\nresize to at least %dx%d (currently %dx%d)", minWidth, minHeight, m.width, m.height)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, warnStyle.Render(msg))
	}

	if m.helpOpen {
		return m.guideView()
	}
	if m.detail != nil {
		return m.detailView()
	}
	if len(m.dbs) == 0 {
		return m.welcomeView()
	}
	if m.focus == focusSearch {
		return m.searchView()
	}
	if m.focus == focusAddDB {
		return m.addDBView()
	}
	if m.confirmRemove >= 0 {
		return m.confirmRemoveView()
	}

	var b strings.Builder
	b.WriteString(m.headerView())
	b.WriteString("\n\n") // blank row separating header from body -- headerLines in layout.go must match this

	sw := sidebarWidthFor(m.width)
	bodyHeight := bodyHeightFor(m.height)

	left := m.sidebarView(sw, bodyHeight)
	right := m.rightView(m.width-sw-4, bodyHeight)
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, left, right))
	b.WriteString("\n")
	b.WriteString(m.statusBarView())

	return b.String()
}

// headerView is a single dense information line -- what btop's own
// header does: identity, aggregate state across every monitored
// database, mode, and clock, all in one scan, before anything else.
func (m Model) headerView() string {
	parts := []string{titleStyle.Render("DBWATCH")}

	if db := m.currentDB(); db != nil && db.Version != "" {
		parts = append(parts, dimStyle.Render("PostgreSQL "+db.Version))
	}
	parts = append(parts, dimStyle.Render(fmt.Sprintf("%d DBs", len(m.dbs))))

	warn, crit := 0, 0
	for _, db := range m.dbs {
		switch db.OverallSeverity() {
		case SeverityCritical, SeverityError:
			crit++
		case SeverityWarning, SeverityDegraded:
			warn++
		}
	}
	switch {
	case crit > 0 && warn > 0:
		parts = append(parts, critStyle.Render(fmt.Sprintf("✕ %d critical", crit)), warnStyle.Render(fmt.Sprintf("⚠ %d warning", warn)))
	case crit > 0:
		parts = append(parts, critStyle.Render(fmt.Sprintf("✕ %d critical", crit)))
	case warn > 0:
		parts = append(parts, warnStyle.Render(fmt.Sprintf("⚠ %d warning", warn)))
	default:
		parts = append(parts, okStyle.Render("✓ all healthy"))
	}

	mode := liveStyle.Render("● LIVE")
	if m.readMode {
		mode = readModeStyle.Render("■ READ")
	}
	parts = append(parts, mode)

	sep := dimStyle.Render(" │ ")
	left := strings.Join(parts, sep)

	updated := "—"
	if !m.lastUpdate.IsZero() {
		updated = m.lastUpdate.Format("15:04:05")
	}
	right := dimStyle.Render(updated)

	gap := max(1, m.width-lipgloss.Width(left)-lipgloss.Width(right))
	return left + strings.Repeat(" ", gap) + right
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

// kv renders one aligned "label   value" row, label in a fixed-width
// muted column so multiple rows line up -- this is what makes a panel
// read as a data table instead of a paragraph.
func kv(label, value string, labelWidth int) string {
	return dimStyle.Render(fmt.Sprintf("%-*s", labelWidth, label)) + value
}

func (m Model) sidebarView(width, height int) string {
	healthTotal, panelTotal := sidebarCardTotalHeights(height)

	boxes := []string{
		renderTitledBox("Health", m.healthCardBody(width-4, healthTotal-2), width, false, healthTotal-2),
	}
	for i, p := range allPanels {
		focused := m.focus == focusPanels && i == m.selectedPanel
		boxes = append(boxes, renderTitledBox(p.Title(), m.panelCardBody(p, width-4, panelTotal-2), width, focused, panelTotal-2))
	}
	content := lipgloss.JoinVertical(lipgloss.Left, boxes...)
	return lipgloss.NewStyle().Width(width).Height(height).Render(content)
}

// healthCardBody is the database identity + real-time load summary.
// Status is the most prominent line (this is "is something wrong?"),
// version is secondary/muted, the database name is prominent (which
// database am I looking at), and everything else is aligned key/value
// rows -- the same shape btop uses for its info boxes.
func (m Model) healthCardBody(width, maxLines int) string {
	db := m.currentDB()
	if db == nil {
		return dimStyle.Render("no database configured")
	}

	sev := db.OverallSeverity()
	version := db.Version
	if version == "" {
		version = "connecting…"
	}

	var lines []string
	lines = append(lines, sev.Style().Render(truncate(strings.ToUpper(sev.Label()), width)))
	lines = append(lines, dimStyle.Render(truncate("PostgreSQL "+version, width)))
	lines = append(lines, labelStyle.Render(truncate(db.Database, width)))

	if db.ConnectErr != nil {
		lines = append(lines, critStyle.Render(truncate(db.ConnectErr.Error(), width)))
	} else {
		if u, ok := db.Uptime(); ok {
			lines = append(lines, kv("uptime", fmtUptime(u), 9))
		}
		if db.LoadLoaded {
			lines = append(lines, kv("xact/s", fmt.Sprintf("%.1f", db.XactPerSec), 9))
			lines = append(lines, kv("tuples/s", fmt.Sprintf("%.0f", db.TuplesPerSec), 9))
			if db.TempBytesRate > 0 {
				lines = append(lines, warnStyle.Render(truncate("⚠ "+humanRate(db.TempBytesRate)+" temp spill", width)))
			} else if db.Deadlocks > 0 {
				lines = append(lines, critStyle.Render(fmt.Sprintf("✕ %d deadlock(s)", db.Deadlocks)))
			}
		} else {
			lines = append(lines, dimStyle.Render("load: measuring…"))
		}
	}

	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	return strings.Join(lines, "\n")
}

// panelCardBody dispatches to each panel's dense renderer. Every one
// degrades gracefully to fewer lines on a short terminal instead of
// overflowing its allotted box height -- rendering fewer lines than
// allocated is always safe; rendering more breaks alignment with
// everything below it.
func (m Model) panelCardBody(p PanelKind, width, maxLines int) string {
	db := m.currentDB()
	if db == nil {
		return dimStyle.Render("no database")
	}
	switch p {
	case PanelConnections:
		return connectionsCardBody(db, width, maxLines)
	case PanelCache:
		return cacheCardBody(db, width, maxLines)
	case PanelQueries:
		return queriesCardBody(db, width, maxLines)
	case PanelLocks:
		return locksCardBody(db, width, maxLines)
	case PanelTransactions:
		return transactionsCardBody(db, width, maxLines)
	case PanelSize:
		return sizeCardBody(db, width, maxLines)
	case PanelCapabilities:
		return capabilitiesCardBody(db, width, maxLines)
	case PanelErrors:
		return errorsCardBody(db, width, maxLines)
	}
	return ""
}

func connectionsCardBody(db *DBState, width, maxLines int) string {
	sev := db.PanelSeverity(PanelConnections)
	if db.ConnErr != nil {
		return critStyle.Render(truncate(db.ConnErr.Error(), width))
	}
	if !db.ConnLoaded {
		return dimStyle.Render("loading…")
	}
	s := db.ConnStats
	pct := s.UtilizationPercent()

	var lines []string
	valueLine := fmt.Sprintf("%d / %d", s.Total, s.MaxConnections)
	pctText := fmt.Sprintf("%.0f%%", pct)
	gap := max(1, width-len(valueLine)-len(pctText))
	lines = append(lines, sev.Style().Render(valueLine+strings.Repeat(" ", gap)+pctText))
	if maxLines >= 2 {
		lines = append(lines, sev.Style().Render(bar(pct, width)))
	}
	if maxLines >= 3 {
		lines = append(lines, dimStyle.Render(truncate(fmt.Sprintf("active %d · idle %d · idle-tx %d", s.Active, s.Idle, s.IdleInTransaction), width)))
	}
	if maxLines >= 4 && len(db.ConnHistory) >= 2 {
		lines = append(lines, dimStyle.Render(sparkline(db.ConnHistory, width)))
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	return strings.Join(lines, "\n")
}

func cacheCardBody(db *DBState, width, maxLines int) string {
	sev := db.PanelSeverity(PanelCache)
	if db.CacheErr != nil {
		return critStyle.Render(truncate(db.CacheErr.Error(), width))
	}
	if !db.CacheLoaded {
		return dimStyle.Render("loading…")
	}
	s := db.CacheStats
	pct := s.HitRatio()

	var lines []string
	lines = append(lines, sev.Style().Render(fmt.Sprintf("%.1f%% hit", pct)))
	if maxLines >= 2 {
		lines = append(lines, sev.Style().Render(bar(pct, width)))
	}
	if maxLines >= 3 {
		total := s.BlocksHit + s.BlocksRead
		missPct := 0.0
		if total > 0 {
			missPct = float64(s.BlocksRead) / float64(total) * 100
		}
		lines = append(lines, dimStyle.Render(truncate(fmt.Sprintf("hit %d · miss %d (%.1f%%)", s.BlocksHit, s.BlocksRead, missPct), width)))
	}
	if maxLines >= 4 && len(db.CacheHistory) >= 2 {
		lines = append(lines, dimStyle.Render(sparkline(db.CacheHistory, width)))
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	return strings.Join(lines, "\n")
}

func queriesCardBody(db *DBState, width, maxLines int) string {
	if db.QueryExtWarn != "" {
		return warnStyle.Render(truncate("disabled — "+db.QueryExtWarn, width))
	}
	if db.QueryErr != nil {
		return critStyle.Render(truncate(db.QueryErr.Error(), width))
	}
	if !db.QueryLoaded {
		return dimStyle.Render("loading…")
	}
	if len(db.QueryStats) == 0 {
		return okStyle.Render(truncate("no query activity yet", width))
	}
	var lines []string
	lines = append(lines, infoStyle.Render(fmt.Sprintf("%d tracked", len(db.QueryStats))))
	if maxLines >= 2 {
		top := db.QueryStats[0]
		lines = append(lines, dimStyle.Render(truncate(fmt.Sprintf("slowest: %s mean", fmtDuration(time.Duration(top.MeanExecMs*float64(time.Millisecond)))), width)))
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	return strings.Join(lines, "\n")
}

func locksCardBody(db *DBState, width, maxLines int) string {
	if db.LockErr != nil {
		return critStyle.Render(truncate(db.LockErr.Error(), width))
	}
	if !db.LockLoaded {
		return dimStyle.Render("loading…")
	}
	if len(db.LockStats) == 0 {
		return okStyle.Render(truncate("no lock contention", width))
	}
	var lines []string
	lines = append(lines, warnStyle.Render(fmt.Sprintf("%d blocked", len(db.LockStats))))
	for _, l := range db.LockStats {
		if len(lines) >= maxLines {
			break
		}
		lines = append(lines, dimStyle.Render(truncate(fmt.Sprintf("%d → %d", l.BlockedPID, l.BlockingPID), width)))
	}
	return strings.Join(lines, "\n")
}

func transactionsCardBody(db *DBState, width, maxLines int) string {
	if db.TxErr != nil {
		return critStyle.Render(truncate(db.TxErr.Error(), width))
	}
	if !db.TxLoaded {
		return dimStyle.Render("loading…")
	}
	if len(db.TxStats) == 0 {
		return okStyle.Render("none")
	}
	var lines []string
	lines = append(lines, warnStyle.Render(fmt.Sprintf("%d running", len(db.TxStats))))
	shown := 0
	for _, t := range db.TxStats {
		if len(lines) >= maxLines {
			break
		}
		lines = append(lines, dimStyle.Render(truncate(fmt.Sprintf("%d %s %s", t.PID, t.State, fmtDuration(t.Duration)), width)))
		shown++
	}
	if shown < len(db.TxStats) && len(lines) < maxLines {
		lines = append(lines, dimStyle.Render(fmt.Sprintf("+%d more", len(db.TxStats)-shown)))
	}
	return strings.Join(lines, "\n")
}

func sizeCardBody(db *DBState, width, maxLines int) string {
	if db.SizeErr != nil {
		return critStyle.Render(truncate(db.SizeErr.Error(), width))
	}
	if !db.SizeLoaded {
		return dimStyle.Render("loading…")
	}
	s := db.SizeStats

	var lines []string
	lines = append(lines, labelStyle.Render(truncate(humanBytes(s.DatabaseBytes), width)))
	if maxLines >= 2 {
		lines = append(lines, dimStyle.Render(truncate(fmt.Sprintf("tables %s · indexes %s", humanBytes(s.TablesBytes), humanBytes(s.IndexesBytes)), width)))
	}
	if maxLines >= 3 {
		switch {
		case !db.SizeGrowthLoaded:
			lines = append(lines, dimStyle.Render("growth: measuring…"))
		case db.SizeGrowthPerSec > 0.5:
			lines = append(lines, infoStyle.Render("↑")+dimStyle.Render(" "+truncate(humanRate(db.SizeGrowthPerSec)+" growing", width-2)))
		case db.SizeGrowthPerSec < -0.5:
			lines = append(lines, infoStyle.Render("↓")+dimStyle.Render(" "+truncate(humanRate(-db.SizeGrowthPerSec)+" shrinking", width-2)))
		default:
			lines = append(lines, dimStyle.Render("steady"))
		}
	}
	if maxLines >= 4 && len(db.SizeHistory) >= 2 {
		lines = append(lines, dimStyle.Render(sparkline(db.SizeHistory, width)))
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	return strings.Join(lines, "\n")
}

func boolMark(b bool) string {
	if b {
		return "✓"
	}
	return "✗"
}

func capabilitiesCardBody(db *DBState, width, maxLines int) string {
	if db.CapErr != nil {
		return critStyle.Render(truncate(db.CapErr.Error(), width))
	}
	if !db.CapLoaded {
		return dimStyle.Render("loading…")
	}
	c := db.CapStats

	var lines []string
	lines = append(lines, labelStyle.Render(truncate("PostgreSQL "+db.Version, width)))
	if maxLines >= 2 {
		visibility := "limited — own queries only"
		style := warnStyle
		if c.IsSuperuser {
			visibility, style = "full visibility (superuser)", dimStyle
		} else if c.HasReadAllStats {
			visibility, style = "full visibility (pg_read_all_stats)", dimStyle
		}
		lines = append(lines, style.Render(truncate(visibility, width)))
	}
	if maxLines >= 3 {
		lines = append(lines, dimStyle.Render(truncate(fmt.Sprintf("track: activities%s counts%s io%s",
			boolMark(c.TrackActivities), boolMark(c.TrackCounts), boolMark(c.TrackIOTiming)), width)))
	}
	if maxLines >= 4 {
		pss := "pg_stat_statements: not available"
		switch {
		case c.HasExtension("pg_stat_statements"):
			pss = "pg_stat_statements: enabled"
		case c.PgStatStatementsAvailable:
			pss = "pg_stat_statements: available, not enabled"
		}
		lines = append(lines, dimStyle.Render(truncate(pss, width)))
	}
	if maxLines >= 5 {
		lines = append(lines, dimStyle.Render(fmt.Sprintf("%d extension(s) installed", len(c.Extensions))))
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	return strings.Join(lines, "\n")
}

func errorsCardBody(db *DBState, width, maxLines int) string {
	entries := db.Logs[PanelErrors].Entries()
	if len(entries) == 0 {
		return okStyle.Render(truncate("no errors or events", width))
	}
	last := entries[len(entries)-1]

	var lines []string
	lines = append(lines, dimStyle.Render(fmt.Sprintf("%d event(s) logged", len(entries))))
	if maxLines >= 2 {
		lines = append(lines, last.Severity.Style().Render(truncate(last.Summary, width)))
	}
	if maxLines >= 3 {
		crit, warn := 0, 0
		for _, e := range entries {
			switch {
			case e.Severity >= SeverityCritical:
				crit++
			case e.Severity == SeverityWarning || e.Severity == SeverityDegraded:
				warn++
			}
		}
		lines = append(lines, dimStyle.Render(fmt.Sprintf("critical %d · warning %d", crit, warn)))
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	return strings.Join(lines, "\n")
}

func (m Model) rightView(width, height int) string {
	dbTableHeight := dbTableHeightFor(len(m.dbs))
	logsHeight := height - dbTableHeight - 1
	if logsHeight < 5 {
		logsHeight = 5
	}

	top := renderTitledBox("Databases", m.dbTableView(width-4, dbTableHeight-2), width, m.focus == focusDBs, dbTableHeight-2)
	bottom := renderTitledBox("Logs / Events", m.logsView(width-4, logsHeight), width, m.focus == focusLogs, logsHeight-2)

	return lipgloss.JoinVertical(lipgloss.Left, top, bottom)
}

// dbTableView renders a scrolling window of at most `rows`-1 database rows
// (one row reserved for the column header), keeping the selected row
// in view with the same minimal-scroll logic the logs panel uses --
// otherwise a saved list longer than the box's fixed height would silently
// render past its border instead of scrolling, breaking the box below it.
func (m Model) dbTableView(width, rows int) string {
	var b strings.Builder
	header := fmt.Sprintf("%-3s %-20s %-12s %s", "", "NAME", "REGION", "STATUS")
	b.WriteString(dimStyle.Render(header) + "\n")

	visibleRows := max(1, rows-1)
	offset := logsWindowOffset(m.selectedDB, visibleRows)
	end := min(offset+visibleRows, len(m.dbs))

	q := strings.ToLower(m.searchQuery)
	for i := offset; i < end; i++ {
		db := m.dbs[i]
		cursor := "  "
		if m.focus == focusDBs && i == m.selectedDB {
			cursor = "> "
		}
		marker := "  "
		if i == m.selectedDB {
			marker = "▸ "
		}
		sev := db.OverallSeverity()
		namePart := fmt.Sprintf("%s%-20s", marker, truncate(db.Name, 20))
		rest := fmt.Sprintf(" %-12s %s %s", truncate(db.Region, 12), sev.Symbol(), sev.Label())

		nameStyle := lipgloss.NewStyle()
		if m.focus == focusDBs && i == m.selectedDB {
			nameStyle = selectedRowStyle
		} else if i == m.selectedDB {
			nameStyle = labelStyle
		} else if q != "" && (strings.Contains(strings.ToLower(db.Name), q) || strings.Contains(strings.ToLower(db.Region), q)) {
			nameStyle = matchStyle
		}
		b.WriteString(cursor + nameStyle.Render(namePart) + sev.Style().Render(rest) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (m Model) logsView(width, height int) string {
	entries := m.filteredEntries()

	status := dimStyle.Render(fmt.Sprintf("[%s]", m.currentPanel().Title()))
	if m.searchQuery != "" {
		status += dimStyle.Render(fmt.Sprintf("  (filter: %q, %d match)", m.searchQuery, len(entries)))
	}

	rows := height - 1
	if rows < 1 {
		rows = 1
	}

	if len(entries) == 0 {
		return status + "\n" + dimStyle.Render("no events yet")
	}

	offset := logsWindowOffset(m.logCursor, rows)
	end := min(offset+rows, len(entries))

	// Every piece is truncated as PLAIN text first, then styled -- styling
	// before truncating fed ANSI escape bytes into the rune budget,
	// undercounting space and risking a truncation landing mid-escape
	// sequence.
	var lines []string
	for i := offset; i < end; i++ {
		e := entries[i]
		selected := m.focus == focusLogs && i == m.logCursor
		cursor := "  "
		if selected {
			cursor = "> "
		}

		tsPlain := e.Time.Format("15:04:05")
		prefixPlain := tsPlain + " " + e.Severity.Symbol() + " "
		summaryBudget := max(5, width-lipgloss.Width(prefixPlain)-2)
		summaryPlain := truncate(e.Summary, summaryBudget)

		var rendered string
		if selected {
			plain := prefixPlain + summaryPlain
			if queryBudget := width - lipgloss.Width(plain) - 4; e.Query != "" && queryBudget > 8 {
				plain += "  " + truncate(e.Query, queryBudget)
			}
			rendered = selectedRowStyle.Render(plain)
		} else {
			rendered = dimStyle.Render(tsPlain) + " " + e.Severity.Style().Render(e.Severity.Symbol()+" "+summaryPlain)
			used := lipgloss.Width(prefixPlain) + lipgloss.Width(summaryPlain)
			if queryBudget := width - used - 4; e.Query != "" && queryBudget > 8 {
				rendered += "  " + queryStyle.Render(truncate(e.Query, queryBudget))
			}
		}
		lines = append(lines, cursor+rendered)
	}

	return status + "\n" + strings.Join(lines, "\n")
}

func (m Model) detailView() string {
	e := m.detail
	var b strings.Builder
	b.WriteString(e.Severity.Render(e.Summary) + "\n\n")
	for _, f := range e.Fields {
		fmt.Fprintf(&b, "%-20s %s\n", labelStyle.Render(f.Key), f.Value)
	}
	if e.Query != "" {
		b.WriteString("\n" + labelStyle.Render("QUERY") + "\n")
		b.WriteString(dimStyle.Render(strings.Repeat("─", 50)) + "\n")
		b.WriteString(e.Query + "\n")
	}
	b.WriteString("\n" + keyStyle.Render("c") + " copy query   " + keyStyle.Render("esc") + " back")

	width := min(m.width-10, 90)
	box := renderTitledBox("Log Detail", strings.TrimRight(b.String(), "\n"), width, true, 0)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

// addDBInputBody is the shared "type a DSN" input body used by both the
// first-run welcome screen and the later "add another database" overlay
// -- one input, one validation/error path, wherever it's shown from.
func (m Model) addDBInputBody(width int) string {
	var b strings.Builder
	b.WriteString("> " + m.addDBInput + "▏")
	if m.addDBErr != "" {
		b.WriteString("\n\n" + critStyle.Render(truncate("✕ "+m.addDBErr, width)))
	}
	b.WriteString("\n\n" + dimStyle.Render("postgres://user:password@host:5432/database"))
	return b.String()
}

// welcomeView is what a fresh install sees: no config file to write by
// hand, no flags required -- type a connection string here and dbwatch
// starts watching it immediately.
func (m Model) welcomeView() string {
	center := func(s string) string {
		return lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(s)
	}

	wordmark := lipgloss.NewStyle().Bold(true).Foreground(theme.Foreground).
		Render(strings.Join(strings.Split("DBWATCH", ""), " "))
	tagline := dimStyle.Render("PostgreSQL observability, live in your terminal")
	label := labelStyle.Render("Enter a PostgreSQL connection string to get started")

	boxWidth := min(m.width-10, 60)
	box := renderTitledBox("Connect", m.addDBInputBody(boxWidth-4), boxWidth, true, 0)

	hint := keyStyle.Render("enter") + " " + dimStyle.Render("connect and start monitoring") + "   " +
		keyStyle.Render("esc") + " " + dimStyle.Render("quit")

	content := strings.Join([]string{
		"",
		center(wordmark),
		center(tagline),
		"",
		center(label),
		"",
		lipgloss.PlaceHorizontal(m.width, lipgloss.Center, box),
		"",
		center(hint),
	}, "\n")

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

// addDBView is the same input as the welcome screen, opened later with
// 'a' to add a second/third database once one is already running.
func (m Model) addDBView() string {
	width := min(m.width-10, 60)
	box := renderTitledBox("Add Database", m.addDBInputBody(width-4)+"\n\n"+
		keyStyle.Render("enter")+" "+dimStyle.Render("connect and save")+"   "+
		keyStyle.Render("esc")+" "+dimStyle.Render("cancel"), width, true, 0)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

// confirmRemoveView guards deletion behind an explicit yes/no -- nothing
// is ever removed from the dashboard or the saved store without this.
func (m Model) confirmRemoveView() string {
	if m.confirmRemove < 0 || m.confirmRemove >= len(m.dbs) {
		return ""
	}
	db := m.dbs[m.confirmRemove]

	var b strings.Builder
	fmt.Fprintf(&b, "Remove %s from dbwatch?\n", labelStyle.Render(db.Name))
	b.WriteString(dimStyle.Render(truncate(db.DSN, 50)) + "\n\n")
	b.WriteString(dimStyle.Render("This stops monitoring it and removes it from your saved database list. It does not affect the database itself.") + "\n\n")
	b.WriteString(warnStyle.Render("y") + " " + dimStyle.Render("yes, remove it") + "    " + okStyle.Render("n") + " " + dimStyle.Render("no, keep it"))

	width := min(m.width-10, 64)
	box := renderTitledBox("Remove Database", b.String(), width, true, 0)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

// searchView is a centered input box -- typing filters logs in the
// current panel and highlights matching databases live, before you even
// press Enter.
func (m Model) searchView() string {
	var b strings.Builder
	b.WriteString("> " + m.searchInput + "▏\n\n")

	dbMatches := 0
	q := strings.ToLower(m.searchInput)
	if q != "" {
		for _, db := range m.dbs {
			if strings.Contains(strings.ToLower(db.Name), q) || strings.Contains(strings.ToLower(db.Region), q) {
				dbMatches++
			}
		}
	}
	logMatches := 0
	if db := m.currentDB(); db != nil && q != "" {
		for _, e := range db.Logs[m.currentPanel()].Entries() {
			if e.MatchesSearch(q) {
				logMatches++
			}
		}
	}
	b.WriteString(dimStyle.Render(fmt.Sprintf("%d database(s) match  ·  %d log entries match in %s", dbMatches, logMatches, m.currentPanel().Title())))
	b.WriteString("\n\n" + dimStyle.Render("searches database names/regions and the selected panel's logs (summary, fields, and query text)"))
	b.WriteString("\n\n" + keyStyle.Render("enter") + " confirm   " + keyStyle.Render("esc") + " cancel")

	width := min(m.width-10, 64)
	box := renderTitledBox("Search", strings.TrimRight(b.String(), "\n"), width, true, 0)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

// guideContent is the full reference guide, rendered once and scrolled --
// it's too long to fit any single terminal, so unlike the other overlays
// this one is a full-screen, scrollable page rather than a small centered
// box.
func guideContent() []string {
	text := dimStyle.Render("everything dbwatch shows, and what it means") + "\n\n" +

		labelStyle.Render("NAVIGATION") + "\n" +
		keyRow("h / l", "move focus between the panel sidebar, the database table, and the logs panel") +
		keyRow("j / k", "move within whatever is focused (also: mouse wheel, or click a row directly)") +
		keyRow("Enter", "on a database row: select it. on a log entry: open its full detail view") +
		keyRow("Esc", "close whatever is open (detail view, guide, search) / step back a level") + "\n" +

		labelStyle.Render("DATA") + "\n" +
		keyRow("c", "copy the selected log entry, or the open detail view, to the clipboard") +
		keyRow("/", "open search (filters the current panel's logs, highlights matching databases)") +
		keyRow("n / N", "jump to the next / previous match after a search is confirmed") +
		keyRow("m", "toggle read mode: freezes all polling so the screen holds still while you read") +
		keyRow("r", "poll immediately instead of waiting for the next interval") + "\n" +

		labelStyle.Render("APPLICATION") + "\n" +
		keyRow("?", "toggle this guide") +
		keyRow("q", "quit") + "\n" +

		labelStyle.Render("THE DATABASE TABLE") + "\n" +
		"Every database you've added is polled on the same interval, all the\n" +
		"time -- not just the one you're currently looking at. The dot and\n" +
		"status word next to each name always reflect that database's real,\n" +
		"live state, even while you're looking at a different one. Selecting a\n" +
		"row switches which database's panels and logs are shown below.\n" +
		keyRow("a", "add another database (same input as the first-run welcome screen)") +
		keyRow("d", "remove the selected database, after a yes/no confirmation") +
		"There's no config file to hand-write: dbwatch keeps its own list of\n" +
		"databases in your per-user config directory and loads it automatically\n" +
		"every time you start it -- adding or removing one here is the only\n" +
		"step. Removing a database only stops dbwatch watching it; it never\n" +
		"touches the database itself.\n\n" +

		labelStyle.Render("CONNECTIONS") + "\n" +
		"How many connections PostgreSQL currently has open to THIS specific\n" +
		"database (not the whole server), split into:\n" +
		bullet("active", "currently executing a query") +
		bullet("idle", "connected, waiting for the next command") +
		bullet("idle in transaction", "inside BEGIN...COMMIT but nothing is running right now --\n"+
			"      worth watching: this state blocks autovacuum and can hold\n"+
			"      locks open indefinitely if the client forgot to commit") +
		"The total is compared against max_connections (a server-wide\n" +
		"setting, shared across every database on that server). Warning at\n" +
		"70%, degraded at 80%, critical at 90%.\n\n" +

		labelStyle.Render("CACHE HIT RATIO") + "\n" +
		"The percentage of reads PostgreSQL satisfied from memory (shared\n" +
		"buffers) instead of going to disk, from pg_stat_database. Higher is\n" +
		"better -- near 100% means your working set fits in memory. Warning\n" +
		"below 99%, degraded below 95%. A freshly created or barely-used\n" +
		"database can show a low ratio simply from not having enough traffic\n" +
		"yet to form a meaningful sample -- not necessarily a real problem.\n\n" +

		labelStyle.Render("TOP QUERIES") + "\n" +
		"The slowest statements by total accumulated execution time, from the\n" +
		"pg_stat_statements extension (dbwatch tries to enable it\n" +
		"automatically on connect -- if that fails, this panel says why).\n" +
		bullet("calls", "how many times this exact statement shape has run") +
		bullet("mean", "average time per call") +
		bullet("total", "cumulative time across every call") +
		bullet("rows", "rows returned or affected, summed across calls") +
		"Note: pg_stat_statements does not expose true P95/P99 latency, only\n" +
		"mean/min/max/stddev -- dbwatch shows what's actually there rather\n" +
		"than approximate a percentile it can't back up.\n\n" +

		labelStyle.Render("LOCKS & BLOCKING") + "\n" +
		"Detects one query waiting on a lock held by another, via\n" +
		"PostgreSQL's pg_blocking_pids(). Shows which PID is blocked and\n" +
		"which PID is blocking it. Empty means no contention right now. This\n" +
		"is shown as a warning, not critical -- one query waiting on another\n" +
		"is common and often resolves itself; critical is reserved for the\n" +
		"database being genuinely unreachable. Because this is poll-based, a\n" +
		"lock conflict that resolves faster than your polling interval\n" +
		"(PostgreSQL's own deadlock detector typically resolves in under a\n" +
		"second) may never be observed -- that's a real limit of polling,\n" +
		"not a bug.\n\n" +

		labelStyle.Render("LONG-RUNNING TRANSACTIONS") + "\n" +
		"Any transaction open longer than the configured threshold (default\n" +
		"30s), whichever state it's in -- active (a query is genuinely in\n" +
		"flight) or idle in transaction (nothing running, just left open).\n" +
		"That distinction matters: idle-in-transaction is very often an\n" +
		"application bug (a transaction opened and never committed or rolled\n" +
		"back), and it blocks vacuum from cleaning up dead rows for as long\n" +
		"as it stays open.\n\n" +

		labelStyle.Render("DATABASE SIZE") + "\n" +
		"Real, queryable disk usage for this database -- nothing sampled or\n" +
		"estimated:\n" +
		bullet("current size", "pg_database_size() -- every relation plus catalogs, excludes WAL") +
		bullet("tables", "sum of pg_table_size() over user tables (heap + toast), no indexes") +
		bullet("indexes", "sum of pg_indexes_size() over user tables") +
		bullet("growth", "a signed rate from consecutive readings this session -- size\n"+
			"      can shrink (VACUUM FULL, TRUNCATE, DROP), so a negative rate is\n"+
			"      a real reading, not clamped away like a reset counter would be") +
		"There's no warning/critical threshold on size itself -- dbwatch has no\n" +
		"way to know this server's actual disk capacity via SQL, so it won't\n" +
		"invent a capacity alarm it can't back up. The log records a baseline\n" +
		"reading, then any later change of at least 10 MiB or 5% (whichever\n" +
		"comes first), so the event stream stays a record of real change\n" +
		"instead of autovacuum-scale noise.\n\n" +

		labelStyle.Render("CAPABILITIES") + "\n" +
		"What this server actually exposes, and permits for the connected\n" +
		"role -- every other panel is only as complete as these allow:\n" +
		bullet("visibility", "superuser or pg_read_all_stats sees every backend in\n"+
			"      pg_stat_activity; without either, a role only sees its own,\n"+
			"      so Connections/Locks/Transactions above may be under-reporting.\n"+
			"      This is the one case this panel raises a real warning -- it's a\n"+
			"      real, verifiable consequence for the rest of the dashboard") +
		bullet("track_activities / track_counts / track_io_timing", "server-side\n"+
			"      settings controlling whether pg_stat_activity and pg_stat_*\n"+
			"      collect anything at all -- off means empty data, not zero activity") +
		bullet("pg_stat_statements", "whether it's merely available (would need\n"+
			"      shared_preload_libraries in postgresql.conf, then a restart), or\n"+
			"      already enabled and backing the Top Queries panel") +
		"A baseline reading is logged once, then again only when something\n" +
		"actually changes (an extension installed at runtime, a role grant, a\n" +
		"config reload flipping a track_* setting) -- not every tick.\n\n" +

		labelStyle.Render("ERRORS / EVENTS") + "\n" +
		"This panel has no collector of its own -- it's every other panel's\n" +
		"Warning-severity-or-worse log entry, cross-posted here the moment it\n" +
		"happens, plus deadlock events (an exact pg_stat_database counter, so\n" +
		"an increase is logged directly) that don't otherwise have a home.\n" +
		"One merged timeline of query failures, connection failures, lock\n" +
		"contention, long transactions, and deadlocks, instead of checking\n" +
		"six panels to find out what actually went wrong. Its status dot\n" +
		"reflects the most recent event, not a running worst -- there's no\n" +
		"clean way to say a past discrete event has \"resolved\", so it shows\n" +
		"what just happened rather than accumulating alarm forever.\n\n" +

		labelStyle.Render("HEALTH CARD & DATABASE LOAD") + "\n" +
		"The top box in the sidebar: status, version, database name, uptime,\n" +
		"and a live load summary derived from pg_stat_database counters:\n" +
		bullet("xact/s", "committed + rolled-back transactions per second") +
		bullet("tuples/s", "rows returned, fetched, inserted, updated, and deleted per second") +
		bullet("temp spill", "shown only when queries are writing temp files to disk --\n"+
			"      usually means work_mem is too small for what's running") +
		bullet("deadlocks", "cumulative count since PostgreSQL's stats were last reset") +
		"These are real rates (this snapshot minus the last one, divided by\n" +
		"elapsed time) -- not estimates. What's deliberately NOT here is CPU,\n" +
		"memory, or disk hardware usage: PostgreSQL's statistics views don't\n" +
		"expose OS-level metrics, only what's happening inside the database\n" +
		"itself. Real CPU% needs an agent reading the host's /proc (or a\n" +
		"cloud provider's own monitoring API) -- a different data source\n" +
		"than anything SQL can query, so dbwatch won't fake a number for it.\n\n" +

		labelStyle.Render("SPARKLINES") + "\n" +
		"Connections and cache panels show a small trend line once dbwatch\n" +
		"has collected at least two readings for this session. It's built\n" +
		"only from real samples taken since dbwatch started watching this\n" +
		"database -- there's no historical data before that, so nothing is\n" +
		"backfilled or estimated to fill the gap.\n\n" +

		labelStyle.Render("LOGS / EVENTS PANEL") + "\n" +
		"A running history of what dbwatch has observed for whichever panel\n" +
		"is selected on the left -- switching panels swaps which history you\n" +
		"see. It has its own cursor and scroll position, completely\n" +
		"independent of the periodic refresh: moving through history while\n" +
		"new data keeps arriving never jumps you back to the bottom. Press\n" +
		"Enter on any entry for full detail (every field, plus the complete\n" +
		"query text if there is one); press c to copy it.\n\n" +

		labelStyle.Render("READ MODE VS. LIVE MODE") + "\n" +
		"Live mode (the default) polls every database on the configured\n" +
		"interval, continuously. Read mode (m) freezes that entirely --\n" +
		"nothing refreshes -- so you can study something without the screen\n" +
		"changing under you. Press m again to resume; polling picks up\n" +
		"immediately on the next tick.\n\n" +

		labelStyle.Render("MOUSE") + "\n" +
		"Scroll wheel moves the selection in whatever section your cursor is\n" +
		"over. Clicking a sidebar card, a database row, or a log entry\n" +
		"selects it directly, the same as navigating there with j/k.\n\n" +

		dimStyle.Render("press ? or esc to close")

	return strings.Split(text, "\n")
}

func keyRow(key, desc string) string {
	return fmt.Sprintf("  %s %s\n", keyStyle.Render(fmt.Sprintf("%-9s", key)), dimStyle.Render(desc))
}

func bullet(term, desc string) string {
	return fmt.Sprintf("  %s %s\n", labelStyle.Render(term+":"), desc)
}

func (m Model) guideView() string {
	lines := guideContent()
	// Two rows held back: one for the titled border's own top+bottom, one
	// for the scroll indicator, so it always lands on its own line
	// instead of trailing after whatever padding the last content line
	// happens to have.
	height := m.height - 4
	if height < 5 {
		height = 5
	}
	offset := logsWindowOffset(m.helpOffset, height)
	end := min(offset+height, len(lines))

	visible := strings.Join(lines[offset:end], "\n")
	hint := dimStyle.Render(fmt.Sprintf("line %d/%d — j/k or wheel to scroll, g/G top/bottom", m.helpOffset+1, len(lines)))
	body := visible + "\n" + hint

	return renderTitledBox("DBWatch Guide", body, m.width, true, height)
}

func (m Model) statusBarView() string {
	sep := dimStyle.Render(" │ ")
	kbd := func(key, desc string) string {
		return keyStyle.Render(key) + " " + dimStyle.Render(desc)
	}

	var segs []string
	switch {
	case m.flash != "":
		return statusBarStyle.Render(flashStyle.Render("✓ " + m.flash))
	case m.readMode && m.focus == focusLogs:
		segs = []string{readModeStyle.Render("READ MODE ■"), "LOGS", kbd("j/k", "select"), kbd("Enter", "inspect"), kbd("c", "copy"), kbd("/", "search"), kbd("m", "live"), kbd("h", "back"), kbd("q", "quit")}
	case m.readMode:
		segs = []string{readModeStyle.Render("READ MODE ■"), kbd("j/k", "scroll"), kbd("h/l", "panel"), kbd("c", "copy"), kbd("m", "live"), kbd("q", "quit")}
	case m.focus == focusLogs:
		segs = []string{liveStyle.Render("LIVE ●"), kbd("j/k", "select"), kbd("Enter", "inspect"), kbd("c", "copy"), kbd("/", "search"), kbd("m", "read/live"), kbd("h", "back"), kbd("q", "quit")}
	case m.focus == focusDBs:
		segs = []string{liveStyle.Render("LIVE ●"), kbd("j/k", "select"), kbd("a", "add db"), kbd("d", "remove db"), kbd("Enter", "panels"), kbd("/", "search"), kbd("m", "read"), kbd("q", "quit")}
	default:
		segs = []string{liveStyle.Render("LIVE ●"), kbd("h/l", "panel"), kbd("j/k", "navigate"), kbd("Enter", "inspect"), kbd("a", "add db"), kbd("/", "search"), kbd("r", "refresh"), kbd("m", "read"), kbd("?", "help"), kbd("q", "quit")}
	}
	return statusBarStyle.Render(strings.Join(segs, sep))
}
