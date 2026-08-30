package tui

// This file is the single source of truth for where things are drawn on
// screen. View() and the mouse click/wheel handler both call into it, so
// clicking on a row and rendering that row can never drift apart.

// headerLines is the number of screen rows headerView() renders (one
// dense info line), plus the blank line View() adds after it, before the
// body starts. Database identity lives in the Health card in the
// sidebar, not the header -- see healthCardBody.
const headerLines = 2

// minSidebarWidth / maxSidebarWidth bound how far the sidebar shrinks on a
// narrow terminal before the layout gives up and shows the "too small"
// message instead of a cramped, unreadable one.
const minSidebarWidth = 20
const maxSidebarWidth = 30

func sidebarWidthFor(termWidth int) int {
	w := termWidth / 4
	if w > maxSidebarWidth {
		return maxSidebarWidth
	}
	if w < minSidebarWidth {
		return minSidebarWidth
	}
	return w
}

func bodyHeightFor(termHeight int) int {
	h := termHeight - 3 // header(1) + blank(1, see headerLines) + status bar(1)
	if h < 6 {
		h = 6
	}
	return h
}

func dbTableHeightFor(dbCount int) int {
	h := 3 + dbCount
	if h > 8 {
		h = 8
	}
	return h
}

// sidebarCardTotalHeights returns the total rendered height (title line +
// content + bottom border) of the Health card and of each panel card below
// it, sized so the whole stack fills bodyHeight rather than leaving dead
// space under a handful of small boxes. The Health card gets twice the
// weight of a panel card -- it has more to say. The unit divides bodyHeight
// by 2 (Health's weight) plus one per panel, so adding a panel to allPanels
// never needs this math touched again.
func sidebarCardTotalHeights(bodyHeight int) (healthTotal, panelTotal int) {
	unit := bodyHeight / (2 + len(allPanels))
	if unit < 3 {
		unit = 3
	}
	return unit * 2, unit
}

// region identifies which part of the dashboard a screen coordinate falls
// into, for both click and wheel-scroll handling.
type region int

const (
	regionNone region = iota
	regionSidebar
	regionDBTable
	regionLogs
)

// hitTest maps a screen coordinate to a region and, where applicable, the
// index within it (panel card index, database row index, or log entry
// index). index is -1 when the click landed in the region's chrome
// (border, header, separator, or the non-interactive Health card) rather
// than a specific selectable row.
func (m Model) hitTest(x, y int) (region, int) {
	if y < headerLines {
		return regionNone, -1
	}
	sw := sidebarWidthFor(m.width)
	bodyTop := headerLines

	if x < sw {
		healthTotal, panelTotal := sidebarCardTotalHeights(bodyHeightFor(m.height))
		relY := y - bodyTop
		if relY < healthTotal {
			return regionSidebar, -1
		}
		panelIdx := (relY - healthTotal) / panelTotal
		if panelIdx < 0 || panelIdx >= len(allPanels) {
			return regionSidebar, -1
		}
		return regionSidebar, panelIdx
	}

	rightY := y - bodyTop
	dbTableHeight := dbTableHeightFor(len(m.dbs))
	dbBoxHeight := dbTableHeight + 2 // +2 border
	if rightY < dbBoxHeight {
		contentY := rightY - 1 // top border
		// dbTableView renders: column header, then a scrolled window of
		// rows -- rowInView must go through the same offset the renderer
		// used, or a click while scrolled would select the wrong database.
		rowInView := contentY - 1
		if rowInView < 0 {
			return regionDBTable, -1
		}
		visibleRows := max(1, dbTableHeight-2-1)
		offset := logsWindowOffset(m.selectedDB, visibleRows)
		dbIdx := offset + rowInView
		if dbIdx < 0 || dbIdx >= len(m.dbs) {
			return regionDBTable, -1
		}
		return regionDBTable, dbIdx
	}

	logsTop := dbBoxHeight
	logsContentY := rightY - logsTop - 1 // top border
	// logsView renders: a status line, then one line per entry.
	rowInView := logsContentY - 1
	if rowInView < 0 {
		return regionLogs, -1
	}
	entries := m.filteredEntries()
	logsBoxHeight := bodyHeightFor(m.height) - dbBoxHeight
	rows := logsBoxHeight - 2 - 1 // border(2) + status line(1)
	offset := logsWindowOffset(m.logCursor, rows)
	idx := offset + rowInView
	if idx < 0 || idx >= len(entries) {
		return regionLogs, -1
	}
	return regionLogs, idx
}

// logsWindowOffset returns the index of the first visible log entry for a
// viewport `rows` tall with the cursor at `cursor` -- minimal-scroll: the
// cursor stays in view, nothing shifts more than necessary.
func logsWindowOffset(cursor, rows int) int {
	if rows <= 0 || cursor < rows {
		return 0
	}
	return cursor - rows + 1
}
