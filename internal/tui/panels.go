package tui

// PanelKind identifies one of the sidebar monitoring sections. Each maps
// 1:1 to a collector.
type PanelKind int

const (
	PanelConnections PanelKind = iota
	PanelCache
	PanelQueries
	PanelLocks
	PanelTransactions
	PanelSize
	PanelCapabilities
	// PanelErrors aggregates -- it never has its own collector. Every other
	// panel's Warning-or-worse log entries are cross-posted into it (see
	// DBState.logEvent), plus deadlock events from RecordActivity, so it
	// reads as one merged "what actually went wrong" timeline instead of a
	// seventh thing to separately poll.
	PanelErrors
)

var allPanels = []PanelKind{
	PanelConnections,
	PanelCache,
	PanelQueries,
	PanelLocks,
	PanelTransactions,
	PanelSize,
	PanelCapabilities,
	PanelErrors,
}

func (k PanelKind) Title() string {
	switch k {
	case PanelConnections:
		return "Connections"
	case PanelCache:
		return "Cache Hit Ratio"
	case PanelQueries:
		return "Top Queries"
	case PanelLocks:
		return "Locks & Blocking"
	case PanelTransactions:
		return "Long-Running Transactions"
	case PanelSize:
		return "Database Size"
	case PanelCapabilities:
		return "Capabilities"
	case PanelErrors:
		return "Errors / Events"
	default:
		return "?"
	}
}
